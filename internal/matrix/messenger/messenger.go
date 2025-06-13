// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messenger

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/matrix"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const roomsCacheSize = 100

var (
	_ messaging.Messenger = (*messenger)(nil)

	errDecompressFailed = errors.New("failed to decompress assembled camino matrix msg")
	errUnmarshalContent = errors.New("failed to unmarshal content")
)

func NewMessenger(
	logger *zap.SugaredLogger,
	matrixClient Client,
	decompressor compression.Decompressor,
	botKey *ecdsa.PrivateKey,
	botUserID id.UserID,
) (messaging.Messenger, error) {
	roomsCache, err := lru.New[id.UserID, id.RoomID](roomsCacheSize)
	if err != nil {
		logger.Errorf("failed to create rooms cache: %v", err)
		return nil, err
	}

	m := &messenger{
		msgChannel:   make(chan types.Message),
		logger:       logger,
		tracer:       otel.GetTracerProvider().Tracer(""),
		client:       matrixClient,
		rooms:        roomsCache,
		decompressor: decompressor,
		messages:     make(map[string][]*matrix.CaminoMatrixMessageEventContent),
		botKey:       botKey,
		botUserID:    botUserID,
	}

	m.client.SetEventHandler(matrix.EventTypeC4TMessage, m.c4tMessageEventHandler)
	m.client.SetEventHandler(event.StateMember, m.stateMemberEventHandler)

	return m, nil
}

type messenger struct {
	botKey    *ecdsa.PrivateKey
	botUserID id.UserID

	msgChannel     chan types.Message
	rooms          *lru.Cache[id.UserID, id.RoomID]
	messages       map[string][]*matrix.CaminoMatrixMessageEventContent
	messagesMutex  sync.RWMutex
	cancelSync     func()
	syncerDoneChan chan struct{}

	logger       *zap.SugaredLogger
	tracer       trace.Tracer
	client       Client
	decompressor compression.Decompressor
}

func (m *messenger) Inbound() chan types.Message {
	return m.msgChannel
}

func (m *messenger) checkpoint() string {
	return "messenger-gateway"
}

func (m *messenger) Start(ctx context.Context) (chan error, error) {
	// Initially, messenger used matrix e2e encryption. Starting from v12, messenger will use its own e2e encryption.
	// Because of that, we need to remove all encrypted rooms and only use unencrypted rooms from now on.
	if err := m.removeEncryptedRooms(ctx); err != nil {
		err := fmt.Errorf("failed to remove encrypted rooms: %w", err)
		m.logger.Error(err)
		return nil, err
	}

	// Start syncer, that will listen for incoming events

	syncCtx, cancelSync := context.WithCancel(ctx)
	errChan := make(chan error)
	m.cancelSync = cancelSync
	m.syncerDoneChan = make(chan struct{})

	// by default, the syncer will get all events from the last hour
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("matrix event syncer panicked: %v", r)
				m.logger.Errorf("recovered from panic: %v", err)
				errChan <- err
			}
			close(errChan)
			close(m.syncerDoneChan)
		}()

		if err := m.client.SyncWithContext(syncCtx); err != nil && !errors.Is(err, context.Canceled) {
			err := fmt.Errorf("matrix event syncer exited with error: %w", err)
			m.logger.Error(err)
			errChan <- err
		}
	}()

	return errChan, nil
}

func (m *messenger) Stop() error {
	if m.cancelSync != nil {
		m.logger.Info("Stopping matrix syncer...")
		// if cancelSync is not nil, it means that the syncer is running
		m.cancelSync()
		// we only wait for the syncer to stop if it was running,
		// otherwise no one will close syncerStopChan
		// and the goroutine will block forever
		<-m.syncerDoneChan
		m.logger.Info("Matrix syncer stopped")
	}

	if err := m.client.Close(); err != nil {
		m.logger.Errorf("Failed to close matrix client: %v", err)
		return err
	}

	return nil
}

func (m *messenger) SendMessage(ctx context.Context, msg *types.Message, sendTo id.UserID) error {
	m.logger.Infof("Sending message (requestID %s) to %s", msg.Metadata.RequestID, sendTo)

	ctx, span := m.tracer.Start(ctx, "messenger.SendMessage", trace.WithSpanKind(trace.SpanKindProducer), trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer span.End()

	ctx, roomSpan := m.tracer.Start(ctx, "messenger.getRoomForRecipient", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	roomID, err := m.getRoomForRecipient(ctx, sendTo)
	if err != nil {
		return err
	}
	roomSpan.End()

	return m.sendMessageEvents(ctx, roomID, matrix.EventTypeC4TMessage, createMessageEventContents(msg))
}

func (m *messenger) sendMessageEvents(ctx context.Context, roomID id.RoomID, eventType event.Type, messageEvents []matrix.CaminoMatrixMessageEventContent) error {
	// TODO @nikos add retry logic?
	for _, msg := range messageEvents {
		if err := m.client.SendMessageEvent(ctx, roomID, eventType, msg); err != nil {
			return err
		}
	}
	return nil
}

func (m *messenger) c4tMessageEventHandler(ctx context.Context, evt *event.Event) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Errorf("failed to process %s event, recovered from panic: %v", matrix.EventTypeC4TMessage.Type, r)
		}
	}()
	m.logger.Debugf("Received %s event %s from %s in room %s", matrix.EventTypeC4TMessage.Type, evt.ID, evt.Sender, evt.RoomID)

	if evt.Sender == m.botUserID { // ignore own messages
		m.logger.Debugf("Ignoring own message from %s in room %s", evt.Sender, evt.RoomID)
		return
	}

	msgEventContent := evt.Content.Parsed.(*matrix.CaminoMatrixMessageEventContent)

	traceID, err := trace.TraceIDFromHex(msgEventContent.Metadata.RequestID)
	if err != nil {
		m.logger.Warnf("failed to parse traceID from hex [requestID:%s]: %v", msgEventContent.Metadata.RequestID, err)
	}
	ctx = trace.ContextWithRemoteSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID}))

	_, span := m.tracer.Start(ctx, "messenger.OnC4TMessageReceive", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attribute.String("type", evt.Type.Type)))
	defer span.End()

	receivedAt := time.Now()

	msg, completed, err := m.tryAssembleMessage(msgEventContent)
	if err != nil {
		m.logger.Errorf("failed to assemble message: %v", err)
		return
	}
	if !completed {
		m.logger.Debugf("Received partial message with requestID %s, waiting for more chunks", msgEventContent.Metadata.RequestID)
		return // partial messages are not passed down to the msgChannel
	}

	msg.SenderBotUserID = evt.Sender

	msg.Metadata.StampOn(fmt.Sprintf("matrix-sent-%s", msgEventContent.MsgType), evt.Timestamp)
	msg.Metadata.StampOn(fmt.Sprintf("%s-%s-%s", m.checkpoint(), "received", msgEventContent.MsgType), receivedAt.UnixMilli())

	m.msgChannel <- msg
}

func (m *messenger) tryAssembleMessage(msgEventContent *matrix.CaminoMatrixMessageEventContent) (types.Message, bool, error) {
	// if the message is not chunked, we can assemble it immediately
	if msgEventContent.Metadata.NumberOfChunks == 1 {
		msg, err := m.assembleMessage(
			msgEventContent.CompressedContent,
			msgEventContent.Metadata,
			types.MessageType(msgEventContent.MsgType),
		)
		return msg, err == nil, err
	}

	msgEventContents, complete := m.tryCompleteChunks(msgEventContent)
	if !complete {
		return types.Message{}, false, nil
	}

	// assemble payload from all chunks
	sort.Sort(matrix.ByChunkIndex(msgEventContents))
	compressedPayloads := make([][]byte, 0, len(msgEventContents))
	for _, msg := range msgEventContents {
		compressedPayloads = append(compressedPayloads, msg.CompressedContent)
	}
	payload := bytes.Join(compressedPayloads, nil)

	msg, err := m.assembleMessage(
		payload,
		msgEventContents[0].Metadata,
		types.MessageType(msgEventContents[0].MsgType),
	)
	return msg, err == nil, err
}

func (m *messenger) tryCompleteChunks(msgEventContent *matrix.CaminoMatrixMessageEventContent) ([]*matrix.CaminoMatrixMessageEventContent, bool) {
	m.messagesMutex.Lock()
	defer m.messagesMutex.Unlock()

	id := msgEventContent.Metadata.RequestID

	msgEventContents := append(m.messages[id], msgEventContent) //nolint:gocritic

	if uint64(len(msgEventContents)) != msgEventContent.Metadata.NumberOfChunks {
		// don't have all chunks yet, store for later assembly and return
		m.messages[id] = msgEventContents
		return nil, false
	}

	delete(m.messages, id)

	return msgEventContents, true
}

func (m *messenger) assembleMessage(payload []byte, metadata metadata.Metadata, msgType types.MessageType) (types.Message, error) {
	contentBytes, err := m.decompressor.Decompress(payload)
	if err != nil {
		return types.Message{}, fmt.Errorf("%w: %w", errDecompressFailed, err)
	}

	msg := types.Message{
		Metadata: metadata,
		Type:     msgType,
	}

	if err := generated.UnmarshalContent(contentBytes, msgType, &msg.Content); err != nil {
		return types.Message{}, fmt.Errorf("%w: %w %v", errUnmarshalContent, err, msgType)
	}

	return msg, nil
}

func createMessageEventContents(msg *types.Message) []matrix.CaminoMatrixMessageEventContent {
	messages := make([]matrix.CaminoMatrixMessageEventContent, 0, len(msg.CompressedContent))

	// add first chunk to messages slice
	caminoMatrixMsg := matrix.CaminoMatrixMessageEventContent{
		MessageEventContent: event.MessageEventContent{MsgType: event.MessageType(msg.Type)},
		Metadata:            msg.Metadata,
	}
	caminoMatrixMsg.Metadata.NumberOfChunks = uint64(len(msg.CompressedContent))
	caminoMatrixMsg.Metadata.ChunkIndex = 0
	caminoMatrixMsg.CompressedContent = msg.CompressedContent[0]
	messages = append(messages, caminoMatrixMsg)

	// if multiple chunks were produced upon compression, add them to messages slice
	for i, chunk := range msg.CompressedContent[1:] {
		messages = append(messages, matrix.CaminoMatrixMessageEventContent{
			MessageEventContent: event.MessageEventContent{MsgType: event.MessageType(msg.Type)},
			Metadata:            metadata.Metadata{RequestID: msg.Metadata.RequestID, NumberOfChunks: uint64(len(msg.CompressedContent)), ChunkIndex: uint64(i + 1)},
			CompressedContent:   chunk,
		})
	}

	return messages
}

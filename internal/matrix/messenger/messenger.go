// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package matrix

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v11/config"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/matrix"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/ethereum/go-ethereum/crypto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	// required to initialize the sqlite driver
	_ "github.com/mattn/go-sqlite3"
)

var _ messaging.Messenger = (*messenger)(nil)

func NewMessenger(
	logger *zap.SugaredLogger,
	cfg config.MatrixConfig,
	botKey *ecdsa.PrivateKey,
	expectedBotUserID id.UserID,
) (messaging.Messenger, error) {
	c, err := mautrix.NewClient(cfg.Host, "", "")
	if err != nil {
		logger.Errorf("failed to create matrix client: %v", err)
		return nil, err
	}
	return &messenger{
		msgChannel: make(chan types.Message),
		logger:     logger,
		tracer:     otel.GetTracerProvider().Tracer(""),
		client: client{
			Client:         c,
			syncerStopChan: make(chan struct{}),
		},
		roomHandler:       NewRoomHandler(NewClient(c), logger),
		msgAssembler:      NewMessageAssembler(),
		botKey:            botKey,
		expectedBotUserID: expectedBotUserID,
		dbPath:            cfg.Store,
	}, nil
}

type messenger struct {
	msgChannel chan types.Message

	dbPath            string
	botKey            *ecdsa.PrivateKey
	expectedBotUserID id.UserID
	logger            *zap.SugaredLogger
	tracer            trace.Tracer

	client       client
	roomHandler  RoomHandler
	msgAssembler MessageAssembler
}

type client struct {
	*mautrix.Client
	ctx            context.Context
	cancelSync     context.CancelFunc
	syncerStopChan chan struct{}
	cryptoHelper   *cryptohelper.CryptoHelper
}

func (m *messenger) checkpoint() string {
	return "messenger-gateway"
}

func (m *messenger) StartReceiver(ctx context.Context) (chan error, error) {
	syncer := m.client.Syncer.(*mautrix.DefaultSyncer)

	syncer.OnEventType(matrix.EventTypeC4TMessage, func(ctx context.Context, evt *event.Event) {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Errorf("failed to process %s event, recovered from panic: %v", matrix.EventTypeC4TMessage.Type, r)
			}
		}()
		m.logger.Debugf("Received %s event %s from %s in room %s", matrix.EventTypeC4TMessage.Type, evt.ID, evt.Sender, evt.RoomID)

		if evt.Sender == m.client.UserID { // ignore own messages
			return
		}

		msg := evt.Content.Parsed.(*matrix.CaminoMatrixMessage)
		traceID, err := trace.TraceIDFromHex(msg.Metadata.RequestID)
		if err != nil {
			m.logger.Warnf("failed to parse traceID from hex [requestID:%s]: %v", msg.Metadata.RequestID, err)
		}
		ctx = trace.ContextWithRemoteSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID}))
		_, span := m.tracer.Start(ctx, "messenger.OnC4TMessageReceive", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attribute.String("type", evt.Type.Type)))
		defer span.End()
		t := time.Now()
		completeMsg, completed, err := m.msgAssembler.AssembleMessage(msg)
		if err != nil {
			m.logger.Errorf("failed to assemble message: %v", err)
			return
		}
		if !completed {
			return // partial messages are not passed down to the msgChannel
		}
		completeMsg.Metadata.StampOn(fmt.Sprintf("matrix-sent-%s", completeMsg.MsgType), evt.Timestamp)
		completeMsg.Metadata.StampOn(fmt.Sprintf("%s-%s-%s", m.checkpoint(), "received", completeMsg.MsgType), t.UnixMilli())
		m.msgChannel <- types.Message{
			Metadata:        completeMsg.Metadata,
			Content:         completeMsg.Content,
			Type:            types.MessageType(msg.MsgType),
			SenderBotUserID: evt.Sender,
		}
	})
	syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Errorf("failed to process %s event, recovered from panic: %v", event.StateMember.Type, r)
			}
		}()
		m.logger.Debugf("Received %s event %s from %s in room %s", event.StateMember.Type, evt.ID, evt.Sender, evt.RoomID)

		if evt.GetStateKey() == m.client.UserID.String() && evt.Content.AsMember().Membership == event.MembershipInvite {
			_, err := m.client.JoinRoomByID(ctx, evt.RoomID)
			if err == nil {
				m.logger.Info("Joined room after invite",
					zap.String("room_id", evt.RoomID.String()),
					zap.String("inviter", evt.Sender.String()))
			} else {
				m.logger.Error("Failed to join room after invite",
					zap.String("room_id", evt.RoomID.String()),
					zap.String("inviter", evt.Sender.String()))
			}
		}
	})

	cryptoHelper, err := cryptohelper.NewCryptoHelper(m.client.Client, []byte("meow"), m.dbPath) // TODO @nikos refactor
	if err != nil {
		return nil, err
	}

	signature, message, err := SignPublicKey(m.botKey)
	if err != nil {
		return nil, err
	}

	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:      mautrix.AuthTypeCamino,
		PublicKey: message[2:],   // removing 0x prefix
		Signature: signature[2:], // removing 0x prefix
	}

	if err = cryptoHelper.Init(ctx); err != nil {
		return nil, err
	}

	if m.client.Client.UserID != m.expectedBotUserID {
		return nil, fmt.Errorf("expected user ID %s, got %s", m.expectedBotUserID, m.client.Client.UserID)
	}

	// Set the wrappedClient crypto helper in order to automatically encrypt outgoing messages
	m.client.Crypto = cryptoHelper
	m.client.cryptoHelper = cryptoHelper // nikos: we need the struct cause stop method is not available on the interface level

	m.logger.Infof("Successfully logged in as: %s", m.client.UserID)
	m.client.ctx, m.client.cancelSync = context.WithCancel(ctx)
	errChan := make(chan error)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("matrix event syncer panicked: %v", r)
				m.logger.Errorf("recovered from panic: %v", err)
				errChan <- err
			}
			close(errChan)
			close(m.client.syncerStopChan)
		}()

		if err := m.client.SyncWithContext(m.client.ctx); err != nil && !errors.Is(err, context.Canceled) {
			err := fmt.Errorf("matrix event syncer exited with error: %w", err)
			m.logger.Error(err)
			errChan <- err
		}
	}()

	return errChan, nil
}

func (m *messenger) StopReceiver() error {
	m.logger.Info("Stopping matrix syncer...")
	if m.client.cancelSync != nil {
		// if cancelSync is not nil, it means that the syncer is running
		m.client.cancelSync()
		// we only wait for the syncer to stop if it was running,
		// otherwise no one will close syncerStopChan
		// and the goroutine will block forever
		<-m.client.syncerStopChan
	}
	if err := m.client.cryptoHelper.Close(); err != nil {
		m.logger.Errorf("Failed to close crypto helper: %v", err)
	}
	m.logger.Info("Matrix syncer stopped")
	return nil
}

func (m *messenger) SendAsync(ctx context.Context, msg *types.Message, sendTo id.UserID) error {
	m.logger.Info("Sending async message", zap.String("msg", msg.Metadata.RequestID))
	ctx, span := m.tracer.Start(ctx, "messenger.SendAsync", trace.WithSpanKind(trace.SpanKindProducer), trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer span.End()

	ctx, roomSpan := m.tracer.Start(ctx, "roomHandler.GetOrCreateRoom", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	roomID, err := m.roomHandler.GetOrCreateRoomForRecipient(ctx, sendTo)
	if err != nil {
		return err
	}
	roomSpan.End()

	return m.sendMessageEvents(ctx, roomID, matrix.EventTypeC4TMessage, createMatrixMessages(msg))
}

func (m *messenger) sendMessageEvents(ctx context.Context, roomID id.RoomID, eventType event.Type, messages []matrix.CaminoMatrixMessage) error {
	// TODO @nikos add retry logic?
	for _, msg := range messages {
		_, err := m.client.SendMessageEvent(ctx, roomID, eventType, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *messenger) Inbound() chan types.Message {
	return m.msgChannel
}

func SignPublicKey(key *ecdsa.PrivateKey) (signature string, message string, err error) {
	pubKeyBytes := crypto.FromECDSAPub(&key.PublicKey)
	signatureBytes, err := sign(pubKeyBytes, key)
	if err != nil {
		return "", "", err
	}

	signature, err = hexWithChecksum(signatureBytes)
	if err != nil {
		return "", "", err
	}
	message, err = hexWithChecksum(pubKeyBytes)
	if err != nil {
		return "", "", err
	}
	return signature, message, nil
}

func sign(msg []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	// TODO @evlekht use crypto.keccak256 in conduit, here and in asb
	hash256 := sha256.Sum256(msg)

	signature, err := crypto.Sign(hash256[:], key)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

func hexWithChecksum(bytes []byte) (string, error) {
	const checksumLen = 4
	bytesLen := len(bytes)
	if bytesLen > math.MaxInt32-checksumLen {
		return "", errors.New("encoding overflow")
	}
	checked := make([]byte, bytesLen+checksumLen)
	copy(checked, bytes)
	hash := sha256.Sum256(bytes)
	copy(checked[len(bytes):], hash[len(hash)-checksumLen:])
	bytes = checked
	return fmt.Sprintf("0x%x", bytes), nil
}

func createMatrixMessages(msg *types.Message) []matrix.CaminoMatrixMessage {
	messages := make([]matrix.CaminoMatrixMessage, 0, len(msg.CompressedContent))

	// add first chunk to messages slice
	caminoMatrixMsg := matrix.CaminoMatrixMessage{
		MessageEventContent: event.MessageEventContent{MsgType: event.MessageType(msg.Type)},
		Metadata:            msg.Metadata,
	}
	caminoMatrixMsg.Metadata.NumberOfChunks = uint64(len(msg.CompressedContent))
	caminoMatrixMsg.Metadata.ChunkIndex = 0
	caminoMatrixMsg.CompressedContent = msg.CompressedContent[0]
	messages = append(messages, caminoMatrixMsg)

	// if multiple chunks were produced upon compression, add them to messages slice
	for i, chunk := range msg.CompressedContent[1:] {
		messages = append(messages, matrix.CaminoMatrixMessage{
			MessageEventContent: event.MessageEventContent{MsgType: event.MessageType(msg.Type)},
			Metadata:            metadata.Metadata{RequestID: msg.Metadata.RequestID, NumberOfChunks: uint64(len(msg.CompressedContent)), ChunkIndex: uint64(i + 1)},
			CompressedContent:   chunk,
		})
	}

	return messages
}

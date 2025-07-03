// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messenger

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/conversion"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/matrix"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type chunkedMessage struct {
	msgType  types.MessageType
	metadata metadata.Metadata
	chunks   []messageChunk
}

type messageChunk struct {
	index uint32
	data  []byte
}

type byChunkIndex []messageChunk

func (b byChunkIndex) Len() int           { return len(b) }
func (b byChunkIndex) Less(i, j int) bool { return b[i].index < b[j].index }
func (b byChunkIndex) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func (m *messenger) SendMessage(ctx context.Context, msg *types.Message, sendTo id.UserID) error {
	m.logger.Debugf("Sending message (id %s) to %s", msg.Metadata.RequestID, sendTo)

	ctx, span := m.tracer.Start(ctx, "messenger.SendMessage", trace.WithSpanKind(trace.SpanKindProducer), trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer span.End()

	ctx, roomSpan := m.tracer.Start(ctx, "messenger.getRoomForRecipient", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	roomID, err := m.getRoomForRecipient(ctx, sendTo)
	if err != nil {
		return err
	}
	roomSpan.End()

	messageEvent := matrix.MessageEventContent{
		MsgType:  msg.Type,
		Metadata: msg.Metadata,
		Data:     msg.CompressedContent[0],
	}

	messageEvent.Metadata.NumberOfChunks = uint64(len(msg.CompressedContent))

	var chunkEvents []matrix.MessageChunkEventContent
	if msg.Metadata.NumberOfChunks > 1 {
		chunkEvents = make([]matrix.MessageChunkEventContent, 0, msg.Metadata.NumberOfChunks-1)
		for i, chunk := range msg.CompressedContent[1:] {
			chunkEvents = append(chunkEvents, matrix.MessageChunkEventContent{
				RequestID:  msg.Metadata.RequestID,
				ChunkIndex: conversion.MustIntToUInt32(i + 1),
				Data:       chunk,
			})
		}
	}

	// TODO @nikos add retry logic?
	if err := m.client.SendMessageEvent(ctx, roomID, messageEvent); err != nil {
		return err
	}

	for _, chunkEvent := range chunkEvents {
		if err := m.client.SendMessageChunkEvent(ctx, roomID, chunkEvent); err != nil {
			return err
		}
	}
	return nil
}

func (m *messenger) messageEventHandler(ctx context.Context, evt *event.Event) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Errorf("failed to process %s event, recovered from panic: %v", &matrix.EventTypeMessage.Type, r)
		}
	}()
	m.logger.Debugf("Received %s event %s from %s in room %s", matrix.EventTypeMessage.Type, evt.ID, evt.Sender, evt.RoomID)

	if evt.Sender == m.botUserID { // ignore own messages
		m.logger.Debugf("Ignoring own message from %s in room %s", evt.Sender, evt.RoomID)
		return
	}

	eventContent := evt.Content.Parsed.(*matrix.MessageEventContent)

	traceID, err := trace.TraceIDFromHex(eventContent.Metadata.RequestID)
	if err != nil {
		m.logger.Warnf("failed to parse traceID from hex [requestID: %s]: %v", eventContent.Metadata.RequestID, err)
	}
	ctx = trace.ContextWithRemoteSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID}))

	_, span := m.tracer.Start(ctx, "messenger.messageEventHandler", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attribute.String("type", evt.Type.Type)))
	defer span.End()

	if err := eventContent.Verify(); err != nil {
		m.logger.Warnf("Received invalid %s event (id %s): %v", &matrix.EventTypeMessage.Type, eventContent.Metadata.RequestID, err)
		return
	}

	receivedAt := time.Now()

	msg, completed, err := m.tryCompleteMessageWithFirstChunk(eventContent)
	if err != nil {
		m.logger.Warnf("Failed to assemble message from %s event (id %s): %v", &matrix.EventTypeMessage.Type, eventContent.Metadata.RequestID, err)
		return
	} else if !completed {
		m.logger.Debugf("Received message (id %s) first chunk, waiting for more chunks", eventContent.Metadata.RequestID)
		return
	}

	msg.SenderBotUserID = evt.Sender

	msg.Metadata.StampOn(fmt.Sprintf("matrix-sent-%s", msg.Type), evt.Timestamp)
	msg.Metadata.StampOn(fmt.Sprintf("%s-%s-%s", m.checkpoint(), "received", msg.Type), receivedAt.UnixMilli())

	m.msgChannel <- msg
}

func (m *messenger) messageChunkEventHandler(ctx context.Context, evt *event.Event) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Errorf("failed to process %s event, recovered from panic: %v", &matrix.EventTypeMessageChunk.Type, r)
		}
	}()
	m.logger.Debugf("Received %s event %s from %s in room %s", matrix.EventTypeMessageChunk.Type, evt.ID, evt.Sender, evt.RoomID)

	if evt.Sender == m.botUserID { // ignore own messages
		m.logger.Debugf("Ignoring own message from %s in room %s", evt.Sender, evt.RoomID)
		return
	}

	eventContent := evt.Content.Parsed.(*matrix.MessageChunkEventContent)

	traceID, err := trace.TraceIDFromHex(eventContent.RequestID)
	if err != nil {
		m.logger.Warnf("failed to parse traceID from hex [requestID: %s]: %v", eventContent.RequestID, err)
	}
	ctx = trace.ContextWithRemoteSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID}))

	_, span := m.tracer.Start(ctx, "messenger.messageChunkEventHandler", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attribute.String("type", evt.Type.Type)))
	defer span.End()

	if err := eventContent.Verify(); err != nil {
		m.logger.Warnf("Received invalid %s event: %v", &matrix.EventTypeMessageChunk.Type, err)
		return
	}

	receivedAt := time.Now()

	msg, completed, err := m.tryCompleteMessage(eventContent)
	if err != nil {
		m.logger.Warnf("Failed to assemble message from %s event (id %s): %v", &matrix.EventTypeMessage.Type, eventContent.RequestID, err)
		return
	} else if !completed {
		m.logger.Debugf("Received message (id %s) chunk %d, waiting for more chunks", eventContent.RequestID, eventContent.ChunkIndex)
		return
	}

	msg.SenderBotUserID = evt.Sender

	msg.Metadata.StampOn(fmt.Sprintf("matrix-sent-%s", msg.Type), evt.Timestamp)
	msg.Metadata.StampOn(fmt.Sprintf("%s-%s-%s", m.checkpoint(), "received", msg.Type), receivedAt.UnixMilli())

	m.msgChannel <- msg
}

func (m *messenger) tryCompleteMessageWithFirstChunk(eventContent *matrix.MessageEventContent) (types.Message, bool, error) {
	// if the message is not chunked, we can already complete it
	if eventContent.Metadata.NumberOfChunks == 1 {
		msg, err := m.assembleMessage(eventContent.Data, eventContent.Metadata, eventContent.MsgType)
		return msg, err == nil, err
	}

	chunkedMessage, complete := m.addMessageFirstChunk(eventContent)
	if !complete {
		return types.Message{}, false, nil
	}
	msg, err := m.assembleMessageFromChunks(chunkedMessage)
	return msg, err == nil, err
}

func (m *messenger) tryCompleteMessage(eventContent *matrix.MessageChunkEventContent) (types.Message, bool, error) {
	chunkedMessage, complete := m.addMessageNextChunk(eventContent)
	if !complete {
		return types.Message{}, false, nil
	}
	msg, err := m.assembleMessageFromChunks(chunkedMessage)
	return msg, err == nil, err
}

func (m *messenger) addMessageFirstChunk(eventContent *matrix.MessageEventContent) (*chunkedMessage, bool) {
	m.messagesMutex.Lock()
	defer m.messagesMutex.Unlock()

	message, ok := m.chunkedMessages[eventContent.Metadata.RequestID]
	if !ok {
		message = &chunkedMessage{chunks: make([]messageChunk, 0, eventContent.Metadata.NumberOfChunks)}
		m.chunkedMessages[eventContent.Metadata.RequestID] = message
	}

	message.metadata = eventContent.Metadata
	message.msgType = eventContent.MsgType

	return m.addMessageChunk(message, &matrix.MessageChunkEventContent{
		RequestID: eventContent.Metadata.RequestID,
		Data:      eventContent.Data,
	})
}

func (m *messenger) addMessageNextChunk(eventContent *matrix.MessageChunkEventContent) (*chunkedMessage, bool) {
	m.messagesMutex.Lock()
	defer m.messagesMutex.Unlock()

	message, ok := m.chunkedMessages[eventContent.RequestID]
	if !ok {
		message = &chunkedMessage{chunks: []messageChunk{}}
		m.chunkedMessages[eventContent.RequestID] = message
	}

	return m.addMessageChunk(message, eventContent)
}

func (m *messenger) addMessageChunk(message *chunkedMessage, eventContent *matrix.MessageChunkEventContent) (*chunkedMessage, bool) {
	message.chunks = append(message.chunks, messageChunk{
		index: eventContent.ChunkIndex,
		data:  eventContent.Data,
	})

	if message.metadata.NumberOfChunks == 0 || uint64(len(message.chunks)) < message.metadata.NumberOfChunks {
		return nil, false
	}

	delete(m.chunkedMessages, eventContent.RequestID)

	return message, true
}

func (m *messenger) assembleMessageFromChunks(chunkedMessage *chunkedMessage) (types.Message, error) {
	sort.Sort(byChunkIndex(chunkedMessage.chunks))
	compressedPayloads := make([][]byte, 0, len(chunkedMessage.chunks))
	for _, chunk := range chunkedMessage.chunks {
		compressedPayloads = append(compressedPayloads, chunk.data)
	}
	return m.assembleMessage(bytes.Join(compressedPayloads, nil), chunkedMessage.metadata, chunkedMessage.msgType)
}

func (m *messenger) assembleMessage(payload []byte, metadata metadata.Metadata, msgType types.MessageType) (types.Message, error) {
	contentBytes, err := m.decompressor.Decompress(payload)
	if err != nil {
		return types.Message{}, fmt.Errorf("%w: %w", errDecompressFailed, err)
	}

	msg := types.Message{
		Type:     msgType,
		Metadata: metadata,
	}

	if err := generated.UnmarshalContent(contentBytes, msgType, &msg.Content); err != nil {
		return types.Message{}, fmt.Errorf("%w: %w %v", errUnmarshalContent, err, msgType)
	}

	return msg, nil
}

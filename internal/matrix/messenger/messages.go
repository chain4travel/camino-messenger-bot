// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messenger

import (
	"context"
	"sort"

	"github.com/chain4travel/camino-messenger-bot/v13/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/cheques"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/conversion"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/matrix"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"maunium.net/go/mautrix/event"
)

type chunkedMessage struct {
	chunksCount   uint32
	fromCMAccount common.Address
	signature     []byte
	chunks        []messageChunk
}

type messageChunk struct {
	index uint32
	data  []byte
}

type byChunkIndex []messageChunk

func (b byChunkIndex) Len() int           { return len(b) }
func (b byChunkIndex) Less(i, j int) bool { return b[i].index < b[j].index }
func (b byChunkIndex) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func (m *messenger) SendMessage(ctx context.Context, msg *messaging.EncodedSignedMessage, sendTo common.Address, networkFeeCheque *cheques.SignedCheque) error {
	messageID := uuid.New().String()

	m.logger.Debugf("Sending message (id %s) to %s", messageID, sendTo)

	roomID, err := m.getRoomForRecipient(ctx, matrix.UserIDFromAddress(sendTo, m.matrixHost))
	if err != nil {
		return err
	}

	messageEvent := matrix.SignedMessageEventContent{
		ChunkData: matrix.ChunkData{
			MessageID: messageID,
			Data:      msg.ChunkedEncodedMessage[0],
		},
		Signature:        msg.Signature,
		NetworkFeeCheque: *networkFeeCheque,
		ChunksCount:      conversion.MustIntToUInt32(len(msg.ChunkedEncodedMessage)),
	}

	var chunkEvents []matrix.MessageChunkEventContent
	if messageEvent.ChunksCount > 1 {
		chunkEvents = make([]matrix.MessageChunkEventContent, 0, len(msg.ChunkedEncodedMessage)-1)
		for i, chunk := range msg.ChunkedEncodedMessage[1:] {
			chunkEvents = append(chunkEvents, matrix.MessageChunkEventContent{
				ChunkData: matrix.ChunkData{
					MessageID: messageID,
					Data:      chunk,
				},
				ChunkIndex: conversion.MustIntToUInt32(i + 1),
			})
		}
	}

	// TODO @nikos add retry logic?
	if err := m.client.SendSignedMessageEvent(ctx, roomID, messageEvent); err != nil {
		return err
	}

	for _, chunkEvent := range chunkEvents {
		if err := m.client.SendMessageChunkEvent(ctx, roomID, chunkEvent); err != nil {
			return err
		}
	}
	return nil
}

func (m *messenger) signedMessageEventHandler(_ context.Context, evt *event.Event) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Errorf("failed to process %s event, recovered from panic: %v", &matrix.EventTypeSignedMessage.Type, r)
		}
	}()
	m.logger.Debugf("Received %s event %s from %s in room %s", matrix.EventTypeSignedMessage.Type, evt.ID, evt.Sender, evt.RoomID)

	if evt.Sender == m.botUserID { // ignore own messages
		m.logger.Debugf("Ignoring own message from %s in room %s", evt.Sender, evt.RoomID)
		return
	}

	eventContent := evt.Content.Parsed.(*matrix.SignedMessageEventContent)

	if err := eventContent.Verify(); err != nil {
		m.logger.Warnf("Received invalid %s event: %v", &matrix.EventTypeSignedMessage.Type, err)
		return
	}

	msg, completed := m.tryCompleteMessageWithFirstChunk(eventContent)
	if !completed {
		m.logger.Debugf("Received message (id %s) first chunk, waiting for more chunks", eventContent.MessageID)
		return
	}

	msg.SenderBotAddress = matrix.AddressFromUserID(evt.Sender)
	m.msgChannel <- msg
}

func (m *messenger) messageChunkEventHandler(_ context.Context, evt *event.Event) {
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

	if err := eventContent.Verify(); err != nil {
		m.logger.Warnf("Received invalid %s event: %v", &matrix.EventTypeMessageChunk.Type, err)
		return
	}

	msg, completed := m.tryCompleteMessage(eventContent)
	if !completed {
		m.logger.Debugf("Received message (id %s) chunk %d, waiting for more chunks", eventContent.MessageID, eventContent.ChunkIndex)
		return
	}

	msg.SenderBotAddress = matrix.AddressFromUserID(evt.Sender)
	m.msgChannel <- msg
}

func (m *messenger) tryCompleteMessageWithFirstChunk(eventContent *matrix.SignedMessageEventContent) (messaging.EncodedSignedMessageWithSender, bool) {
	// if the message is not chunked, we can already complete it
	if eventContent.ChunksCount == 1 {
		return messaging.EncodedSignedMessageWithSender{
			Message: messaging.EncodedSignedMessage{
				ChunkedEncodedMessage: [][]byte{eventContent.Data},
				Signature:             eventContent.Signature,
			},
			SenderCMAccountAddress: eventContent.NetworkFeeCheque.FromCMAccount,
		}, true
	}

	chunkedMessage, complete := m.addMessageFirstChunk(eventContent)
	if !complete {
		return messaging.EncodedSignedMessageWithSender{}, false
	}
	return m.assembleEncodedMessage(chunkedMessage), true
}

func (m *messenger) tryCompleteMessage(eventContent *matrix.MessageChunkEventContent) (messaging.EncodedSignedMessageWithSender, bool) {
	chunkedMessage, complete := m.addMessageNextChunk(eventContent)
	if !complete {
		return messaging.EncodedSignedMessageWithSender{}, false
	}
	return m.assembleEncodedMessage(chunkedMessage), true
}

func (m *messenger) addMessageFirstChunk(eventContent *matrix.SignedMessageEventContent) (*chunkedMessage, bool) {
	m.messagesMutex.Lock()
	defer m.messagesMutex.Unlock()

	message, ok := m.chunkedMessages[eventContent.MessageID]
	if !ok {
		message = &chunkedMessage{chunks: make([]messageChunk, 0, eventContent.ChunksCount)}
		m.chunkedMessages[eventContent.MessageID] = message
	}

	message.signature = eventContent.Signature
	message.chunksCount = eventContent.ChunksCount
	message.fromCMAccount = eventContent.NetworkFeeCheque.FromCMAccount

	return m.addMessageChunk(message, &eventContent.ChunkData, 0)
}

func (m *messenger) addMessageNextChunk(eventContent *matrix.MessageChunkEventContent) (*chunkedMessage, bool) {
	m.messagesMutex.Lock()
	defer m.messagesMutex.Unlock()

	message, ok := m.chunkedMessages[eventContent.MessageID]
	if !ok {
		message = &chunkedMessage{chunks: []messageChunk{}}
		m.chunkedMessages[eventContent.MessageID] = message
	}

	return m.addMessageChunk(message, &eventContent.ChunkData, eventContent.ChunkIndex)
}

func (m *messenger) addMessageChunk(message *chunkedMessage, chunkData *matrix.ChunkData, chunkIndex uint32) (*chunkedMessage, bool) {
	message.chunks = append(message.chunks, messageChunk{
		index: chunkIndex,
		data:  chunkData.Data,
	})

	if message.chunksCount == 0 || conversion.MustIntToUInt32(len(message.chunks)) < message.chunksCount {
		return nil, false
	}

	delete(m.chunkedMessages, chunkData.MessageID)

	return message, true
}

func (m *messenger) assembleEncodedMessage(message *chunkedMessage) messaging.EncodedSignedMessageWithSender {
	sort.Sort(byChunkIndex(message.chunks))
	chunkedData := make([][]byte, 0, len(message.chunks))
	for _, chunk := range message.chunks {
		chunkedData = append(chunkedData, chunk.data)
	}

	return messaging.EncodedSignedMessageWithSender{
		Message: messaging.EncodedSignedMessage{
			ChunkedEncodedMessage: chunkedData,
			Signature:             message.signature,
		},
		SenderCMAccountAddress: message.fromCMAccount,
	}
}

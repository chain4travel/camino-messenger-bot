// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package matrix

import (
	"errors"
	"reflect"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/cheques"
	"maunium.net/go/mautrix/event"
)

var (
	EventTypeSignedMessage = event.Type{Type: "m.room.c4t-signed-msg", Class: event.MessageEventType}
	EventTypeMessageChunk  = event.Type{Type: "m.room.c4t-msg-chunk", Class: event.MessageEventType}

	ErrWrongChunkIndex = errors.New("wrong chunk index")
	ErrNoChunks        = errors.New("zero chunks count")
	ErrNoData          = errors.New("no data in message chunk")
	ErrNoMessageID     = errors.New("missing message ID")
)

func init() {
	event.TypeMap[EventTypeSignedMessage] = reflect.TypeOf(SignedMessageEventContent{})
	event.TypeMap[EventTypeMessageChunk] = reflect.TypeOf(MessageChunkEventContent{})
}

type MessageChunkEventContent struct {
	ChunkData

	ChunkIndex uint32
}

func (e *MessageChunkEventContent) Verify() error {
	if e.ChunkIndex == 0 {
		return ErrWrongChunkIndex
	}
	return e.ChunkData.Verify()
}

type SignedMessageEventContent struct {
	ChunkData

	ChunksCount      uint32
	Signature        []byte
	NetworkFeeCheque cheques.SignedCheque
}

func (e *SignedMessageEventContent) Verify() error {
	if e.ChunksCount == 0 {
		return ErrNoChunks
	}
	return e.ChunkData.Verify()
}

type ChunkData struct {
	MessageID string
	Data      []byte
}

func (c *ChunkData) Verify() error {
	if len(c.Data) == 0 {
		return ErrNoData
	}
	if c.MessageID == "" {
		return ErrNoMessageID
	}
	return nil
}

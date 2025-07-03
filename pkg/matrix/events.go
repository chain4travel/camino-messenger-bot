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

	ErrNoChunks    = errors.New("zero chunks count")
	ErrNoData      = errors.New("no data in message chunk")
	ErrNoMessageID = errors.New("missing message ID")
)

func init() {
	event.TypeMap[EventTypeSignedMessage] = reflect.TypeOf(SignedMessageEventContent{})
	event.TypeMap[EventTypeMessageChunk] = reflect.TypeOf(MessageChunkEventContent{})
}

type MessageChunkEventContent struct {
	MessageID  string `json:"message_id"`
	ChunkIndex uint32 `json:"chunk_index,omitempty"`
	Data       []byte `json:"data"`
}

func (e *MessageChunkEventContent) Verify() error {
	if len(e.Data) == 0 {
		return ErrNoData
	}
	if e.MessageID == "" {
		return ErrNoMessageID
	}
	return nil
}

type SignedMessageEventContent struct {
	MessageChunkEventContent

	ChunksCount      uint32               `json:"chunks_count"`
	Signature        []byte               `json:"signature"`
	NetworkFeeCheque cheques.SignedCheque `json:"network_fee_cheque"`
}

func (e *SignedMessageEventContent) Verify() error {
	if e.ChunksCount == 0 {
		return ErrNoChunks
	}
	return e.MessageChunkEventContent.Verify()
}

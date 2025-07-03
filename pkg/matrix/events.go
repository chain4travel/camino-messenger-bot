// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package matrix

import (
	"errors"
	"reflect"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/cheques"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"maunium.net/go/mautrix/event"
)

var (
	EventTypeMessage      = event.Type{Type: "m.room.c4t-msg", Class: event.MessageEventType}
	EventTypeMessageChunk = event.Type{Type: "m.room.c4t-msg-chunk", Class: event.MessageEventType}

	ErrNoChunks    = errors.New("zero chunks count")
	ErrNoData      = errors.New("no data in message chunk")
	ErrNoRequestID = errors.New("missing request ID")
)

func init() {
	event.TypeMap[EventTypeMessage] = reflect.TypeOf(MessageEventContent{})
	event.TypeMap[EventTypeMessageChunk] = reflect.TypeOf(MessageChunkEventContent{})
}

type MessageChunkEventContent struct {
	RequestID  string `json:"request_id"`
	ChunkIndex uint32 `json:"chunk_index,omitempty"`
	Data       []byte `json:"data"`
}

func (e *MessageChunkEventContent) Verify() error {
	if len(e.Data) == 0 {
		return ErrNoData
	}
	if e.RequestID == "" {
		return ErrNoRequestID
	}
	return nil
}

type MessageEventContent struct {
	MessageChunkEventContent

	MsgType          types.MessageType     `json:"msgtype"`
	ChunksCount      uint32                `json:"chunks_count"`
	Timestamps       metadata.Timestamps   `json:"timestamps"`
	ServiceFeeCheque *cheques.SignedCheque `json:"service_fee_cheque,omitempty"`
	NetworkFeeCheque cheques.SignedCheque  `json:"network_fee_cheque"`
}

func (e *MessageEventContent) Verify() error {
	if e.ChunksCount == 0 {
		return ErrNoChunks
	}
	return e.MessageChunkEventContent.Verify()
}

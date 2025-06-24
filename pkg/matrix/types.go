// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package matrix

import (
	"reflect"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/cheques"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/reflect/protoreflect"
	"maunium.net/go/mautrix/event"
)

var EventTypeC4TMessage = event.Type{Type: "m.room.c4t-msg", Class: event.MessageEventType}

func init() {
	event.TypeMap[EventTypeC4TMessage] = reflect.TypeOf(CaminoMatrixMessageEventContent{})
}

// CaminoMatrixMessageEventContent is a matrix-specific message format used for communication between the messenger and the service
type CaminoMatrixMessageEventContent struct {
	event.MessageEventContent
	Content           protoreflect.ProtoMessage `json:"content"`
	CompressedContent []byte                    `json:"compressed_content"`
	Metadata          metadata.Metadata         `json:"metadata"`
}

type ByChunkIndex []*CaminoMatrixMessageEventContent

func (b ByChunkIndex) Len() int { return len(b) }
func (b ByChunkIndex) Less(i, j int) bool {
	return b[i].Metadata.ChunkIndex < b[j].Metadata.ChunkIndex
}
func (b ByChunkIndex) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func (m *CaminoMatrixMessageEventContent) UnmarshalContent(src []byte) error {
	return generated.UnmarshalContent(src, types.MessageType(m.MsgType), &m.Content)
}

func (m *CaminoMatrixMessageEventContent) GetChequeFor(addr common.Address) *cheques.SignedCheque {
	for _, cheque := range m.Metadata.Cheques {
		if cheque.Cheque.ToCMAccount == addr {
			return &cheque
		}
	}
	return nil
}

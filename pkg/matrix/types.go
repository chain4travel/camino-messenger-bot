package matrix

import (
	"reflect"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/clients"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"maunium.net/go/mautrix/event"
)

var EventTypeC4TMessage = event.Type{Type: "m.room.c4t-msg", Class: event.MessageEventType}

func init() {
	event.TypeMap[EventTypeC4TMessage] = reflect.TypeOf(CaminoMatrixMessage{})
}

// CaminoMatrixMessage is a matrix-specific message format used for communication between the messenger and the service
type CaminoMatrixMessage struct {
	event.MessageEventContent
	Content           protoreflect.ProtoMessage `json:"content"`
	CompressedContent []byte                    `json:"compressed_content"`
	Metadata          metadata.Metadata         `json:"metadata"`
}

type ByChunkIndex []*CaminoMatrixMessage

func (b ByChunkIndex) Len() int { return len(b) }
func (b ByChunkIndex) Less(i, j int) bool {
	return b[i].Metadata.ChunkIndex < b[j].Metadata.ChunkIndex
}
func (b ByChunkIndex) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func (m *CaminoMatrixMessage) UnmarshalContent(src []byte) error {
	switch types.MessageType(m.MsgType) {
	case clients.PingServiceV1Request:
		m.Content = &pingv1.PingRequest{}
	case clients.PingServiceV1Response:
		m.Content = &pingv1.PingResponse{}
	default:
		return types.ErrUnknownMessageType
	}
	return proto.Unmarshal(src, m.Content)
}

func (m *CaminoMatrixMessage) GetChequeFor(addr common.Address) *cheques.SignedCheque {
	for _, cheque := range m.Metadata.Cheques {
		if cheque.Cheque.ToCMAccount == addr {
			return &cheque
		}
	}
	return nil
}

package matrix

import (
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/golang/protobuf/proto"
	"maunium.net/go/mautrix/event"
)

// CaminoMatrixMessage is a matrix-specific message format used for communication between the messenger and the service
type CaminoMatrixMessage struct {
	event.MessageEventContent
	Content           messaging.MessageContent `json:"content"`
	CompressedContent []byte                   `json:"compressed_content"`
	Metadata          metadata.Metadata        `json:"metadata"`
}

type ByChunkIndex []CaminoMatrixMessage

func (b ByChunkIndex) Len() int { return len(b) }
func (b ByChunkIndex) Less(i, j int) bool {
	return b[i].Metadata.ChunkIndex < b[j].Metadata.ChunkIndex
}
func (b ByChunkIndex) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func (m *CaminoMatrixMessage) UnmarshalContent(src []byte) error {
	switch messaging.MessageType(m.MsgType) {
	case messaging.ActivityProductListRequest:
		return proto.Unmarshal(src, &m.Content.RequestContent.ActivityProductListRequest)
	case messaging.ActivityProductListResponse:
		return proto.Unmarshal(src, &m.Content.ResponseContent.ActivityProductListResponse)
	case messaging.ActivitySearchRequest:
		return proto.Unmarshal(src, &m.Content.RequestContent.ActivitySearchRequest)
	case messaging.ActivitySearchResponse:
		return proto.Unmarshal(src, &m.Content.ResponseContent.ActivitySearchResponse)
	case messaging.AccommodationProductInfoRequest:
		return proto.Unmarshal(src, &m.Content.RequestContent.AccommodationProductInfoRequest)
	case messaging.AccommodationProductInfoResponse:
		return proto.Unmarshal(src, &m.Content.ResponseContent.AccommodationProductInfoResponse)
	case messaging.AccommodationProductListRequest:
		return proto.Unmarshal(src, &m.Content.RequestContent.AccommodationProductListRequest)
	case messaging.AccommodationProductListResponse:
		return proto.Unmarshal(src, &m.Content.ResponseContent.AccommodationProductListResponse)
	case messaging.AccommodationSearchRequest:
		return proto.Unmarshal(src, &m.Content.RequestContent.AccommodationSearchRequest)
	case messaging.AccommodationSearchResponse:
		return proto.Unmarshal(src, &m.Content.ResponseContent.AccommodationSearchResponse)
	case messaging.GetNetworkFeeRequest:
		return proto.Unmarshal(src, &m.Content.RequestContent.GetNetworkFeeRequest)
	case messaging.GetNetworkFeeResponse:
		return proto.Unmarshal(src, &m.Content.ResponseContent.GetNetworkFeeResponse)
	case messaging.GetPartnerConfigurationRequest:
		return proto.Unmarshal(src, &m.Content.RequestContent.GetPartnerConfigurationRequest)
	case messaging.GetPartnerConfigurationResponse:
		return proto.Unmarshal(src, &m.Content.ResponseContent.GetPartnerConfigurationResponse)
	case messaging.PingRequest:
		return proto.Unmarshal(src, &m.Content.RequestContent.PingRequest)
	case messaging.PingResponse:
		return proto.Unmarshal(src, &m.Content.ResponseContent.PingResponse)
	case messaging.TransportSearchRequest:
		return proto.Unmarshal(src, &m.Content.RequestContent.TransportSearchRequest)
	case messaging.TransportSearchResponse:
		return proto.Unmarshal(src, &m.Content.ResponseContent.TransportSearchResponse)
	default:
		return messaging.ErrInvalidMessageType
	}
}

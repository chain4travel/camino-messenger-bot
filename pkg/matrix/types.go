package matrix

import (
	"reflect"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	infov1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/info/v1"
	networkv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1"
	partnerv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	seat_mapv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v1"
	transportv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/proto"
	"maunium.net/go/mautrix/event"
)

var EventTypeC4TMessage = event.Type{Type: "m.room.c4t-msg", Class: event.MessageEventType}

func init() {
	event.TypeMap[EventTypeC4TMessage] = reflect.TypeOf(CaminoMatrixMessage{})
}

// CaminoMatrixMessage is a matrix-specific message format used for communication between the messenger and the service
type CaminoMatrixMessage struct {
	event.MessageEventContent
	Content           messaging.MessageContent `json:"content"`
	CompressedContent []byte                   `json:"compressed_content"`
	Metadata          metadata.Metadata        `json:"metadata"`
}

type ByChunkIndex []*CaminoMatrixMessage

func (b ByChunkIndex) Len() int { return len(b) }
func (b ByChunkIndex) Less(i, j int) bool {
	return b[i].Metadata.ChunkIndex < b[j].Metadata.ChunkIndex
}
func (b ByChunkIndex) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func (m *CaminoMatrixMessage) UnmarshalContent(src []byte) error {
	switch messaging.MessageType(m.MsgType) {
	case messaging.ActivityProductInfoRequest:
		m.Content.RequestContent.ActivityProductInfoRequest = &activityv1.ActivityProductInfoRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.ActivityProductInfoRequest)
	case messaging.ActivityProductInfoResponse:
		m.Content.ResponseContent.ActivityProductInfoResponse = &activityv1.ActivityProductInfoResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.ActivityProductInfoResponse)
	case messaging.ActivityProductListRequest:
		m.Content.RequestContent.ActivityProductListRequest = &activityv1.ActivityProductListRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.ActivityProductListRequest)
	case messaging.ActivityProductListResponse:
		m.Content.ResponseContent.ActivityProductListResponse = &activityv1.ActivityProductListResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.ActivityProductListResponse)
	case messaging.ActivitySearchRequest:
		m.Content.RequestContent.ActivitySearchRequest = &activityv1.ActivitySearchRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.ActivitySearchRequest)
	case messaging.ActivitySearchResponse:
		m.Content.ResponseContent.ActivitySearchResponse = &activityv1.ActivitySearchResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.ActivitySearchResponse)
	case messaging.AccommodationProductInfoRequest:
		m.Content.RequestContent.AccommodationProductInfoRequest = &accommodationv1.AccommodationProductInfoRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.AccommodationProductInfoRequest)
	case messaging.AccommodationProductInfoResponse:
		m.Content.ResponseContent.AccommodationProductInfoResponse = &accommodationv1.AccommodationProductInfoResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.AccommodationProductInfoResponse)
	case messaging.AccommodationProductListRequest:
		m.Content.RequestContent.AccommodationProductListRequest = &accommodationv1.AccommodationProductListRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.AccommodationProductListRequest)
	case messaging.AccommodationProductListResponse:
		m.Content.ResponseContent.AccommodationProductListResponse = &accommodationv1.AccommodationProductListResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.AccommodationProductListResponse)
	case messaging.AccommodationSearchRequest:
		m.Content.RequestContent.AccommodationSearchRequest = &accommodationv1.AccommodationSearchRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.AccommodationSearchRequest)
	case messaging.AccommodationSearchResponse:
		m.Content.ResponseContent.AccommodationSearchResponse = &accommodationv1.AccommodationSearchResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.AccommodationSearchResponse)
	case messaging.GetNetworkFeeRequest:
		m.Content.RequestContent.GetNetworkFeeRequest = &networkv1.GetNetworkFeeRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.GetNetworkFeeRequest)
	case messaging.GetNetworkFeeResponse:
		m.Content.ResponseContent.GetNetworkFeeResponse = &networkv1.GetNetworkFeeResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.GetNetworkFeeResponse)
	case messaging.GetPartnerConfigurationRequest:
		m.Content.RequestContent.GetPartnerConfigurationRequest = &partnerv1.GetPartnerConfigurationRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.GetPartnerConfigurationRequest)
	case messaging.GetPartnerConfigurationResponse:
		m.Content.ResponseContent.GetPartnerConfigurationResponse = &partnerv1.GetPartnerConfigurationResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.GetPartnerConfigurationResponse)
	case messaging.MintRequest:
		m.Content.RequestContent.MintRequest = &bookv1.MintRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.MintRequest)
	case messaging.MintResponse:
		m.Content.ResponseContent.MintResponse = &bookv1.MintResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.MintResponse)
	case messaging.ValidationRequest:
		m.Content.RequestContent.ValidationRequest = &bookv1.ValidationRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.ValidationRequest)
	case messaging.ValidationResponse:
		m.Content.ResponseContent.ValidationResponse = &bookv1.ValidationResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.ValidationResponse)
	case messaging.PingRequest:
		m.Content.RequestContent.PingRequest = &pingv1.PingRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.PingRequest)
	case messaging.PingResponse:
		m.Content.ResponseContent.PingResponse = &pingv1.PingResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.PingResponse)
	case messaging.TransportSearchRequest:
		m.Content.RequestContent.TransportSearchRequest = &transportv1.TransportSearchRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.TransportSearchRequest)
	case messaging.TransportSearchResponse:
		m.Content.ResponseContent.TransportSearchResponse = &transportv1.TransportSearchResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.TransportSearchResponse)
	case messaging.SeatMapRequest:
		m.Content.RequestContent.SeatMapRequest = &seat_mapv1.SeatMapRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.SeatMapRequest)
	case messaging.SeatMapResponse:
		m.Content.ResponseContent.SeatMapResponse = &seat_mapv1.SeatMapResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.SeatMapResponse)
	case messaging.SeatMapAvailabilityRequest:
		m.Content.RequestContent.SeatMapAvailabilityRequest = &seat_mapv1.SeatMapAvailabilityRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.SeatMapAvailabilityRequest)
	case messaging.SeatMapAvailabilityResponse:
		m.Content.ResponseContent.SeatMapAvailabilityResponse = &seat_mapv1.SeatMapAvailabilityResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.SeatMapResponse)
	case messaging.CountryEntryRequirementsRequest:
		m.Content.RequestContent.CountryEntryRequirementsRequest = &infov1.CountryEntryRequirementsRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.CountryEntryRequirementsRequest)
	case messaging.CountryEntryRequirementsResponse:
		m.Content.ResponseContent.CountryEntryRequirementsResponse = &infov1.CountryEntryRequirementsResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.CountryEntryRequirementsResponse)
	default:
		return messaging.ErrUnknownMessageType
	}
}

func (m *CaminoMatrixMessage) GetChequeFor(addr common.Address) *cheques.SignedCheque {
	for _, cheque := range m.Metadata.Cheques {
		if cheque.Cheque.ToCMAccount == addr {
			return &cheque
		}
	}
	return nil
}

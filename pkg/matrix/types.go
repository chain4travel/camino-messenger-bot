package matrix

import (
	"reflect"

	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	infov2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/info/v2"
	insurancev1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/insurance/v1"
	networkv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1"
	partnerv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v2"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	seat_mapv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v2"
	transportv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v2"
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
	// TODO: @VjeraTurk support multiple versions per MessageType
	switch messaging.MessageType(m.MsgType) {
	case messaging.ActivityProductInfoRequest:
		m.Content.RequestContent.ActivityProductInfoRequest = &activityv2.ActivityProductInfoRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.ActivityProductInfoRequest)
	case messaging.ActivityProductInfoResponse:
		m.Content.ResponseContent.ActivityProductInfoResponse = &activityv2.ActivityProductInfoResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.ActivityProductInfoResponse)
	case messaging.ActivityProductListRequest:
		m.Content.RequestContent.ActivityProductListRequest = &activityv2.ActivityProductListRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.ActivityProductListRequest)
	case messaging.ActivityProductListResponse:
		m.Content.ResponseContent.ActivityProductListResponse = &activityv2.ActivityProductListResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.ActivityProductListResponse)
	case messaging.ActivitySearchRequest:
		m.Content.RequestContent.ActivitySearchRequest = &activityv2.ActivitySearchRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.ActivitySearchRequest)
	case messaging.ActivitySearchResponse:
		m.Content.ResponseContent.ActivitySearchResponse = &activityv2.ActivitySearchResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.ActivitySearchResponse)
	case messaging.AccommodationProductInfoRequest:
		m.Content.RequestContent.AccommodationProductInfoRequest = &accommodationv2.AccommodationProductInfoRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.AccommodationProductInfoRequest)
	case messaging.AccommodationProductInfoResponse:
		m.Content.ResponseContent.AccommodationProductInfoResponse = &accommodationv2.AccommodationProductInfoResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.AccommodationProductInfoResponse)
	case messaging.AccommodationProductListRequest:
		m.Content.RequestContent.AccommodationProductListRequest = &accommodationv2.AccommodationProductListRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.AccommodationProductListRequest)
	case messaging.AccommodationProductListResponse:
		m.Content.ResponseContent.AccommodationProductListResponse = &accommodationv2.AccommodationProductListResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.AccommodationProductListResponse)
	case messaging.AccommodationSearchRequest:
		m.Content.RequestContent.AccommodationSearchRequest = &accommodationv2.AccommodationSearchRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.AccommodationSearchRequest)
	case messaging.AccommodationSearchResponse:
		m.Content.ResponseContent.AccommodationSearchResponse = &accommodationv2.AccommodationSearchResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.AccommodationSearchResponse)
	case messaging.GetNetworkFeeRequest:
		m.Content.RequestContent.GetNetworkFeeRequest = &networkv1.GetNetworkFeeRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.GetNetworkFeeRequest)
	case messaging.GetNetworkFeeResponse:
		m.Content.ResponseContent.GetNetworkFeeResponse = &networkv1.GetNetworkFeeResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.GetNetworkFeeResponse)
	case messaging.GetPartnerConfigurationRequest:
		m.Content.RequestContent.GetPartnerConfigurationRequest = &partnerv2.GetPartnerConfigurationRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.GetPartnerConfigurationRequest)
	case messaging.GetPartnerConfigurationResponse:
		m.Content.ResponseContent.GetPartnerConfigurationResponse = &partnerv2.GetPartnerConfigurationResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.GetPartnerConfigurationResponse)
	case messaging.MintRequest:
		m.Content.RequestContent.MintRequest = &bookv2.MintRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.MintRequest)
	case messaging.MintResponse:
		m.Content.ResponseContent.MintResponse = &bookv2.MintResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.MintResponse)
	case messaging.ValidationRequest:
		m.Content.RequestContent.ValidationRequest = &bookv2.ValidationRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.ValidationRequest)
	case messaging.ValidationResponse:
		m.Content.ResponseContent.ValidationResponse = &bookv2.ValidationResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.ValidationResponse)
	case messaging.PingRequest:
		m.Content.RequestContent.PingRequest = &pingv1.PingRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.PingRequest)
	case messaging.PingResponse:
		m.Content.ResponseContent.PingResponse = &pingv1.PingResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.PingResponse)
	case messaging.TransportSearchRequest:
		m.Content.RequestContent.TransportSearchRequest = &transportv2.TransportSearchRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.TransportSearchRequest)
	case messaging.TransportSearchResponse:
		m.Content.ResponseContent.TransportSearchResponse = &transportv2.TransportSearchResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.TransportSearchResponse)
	case messaging.SeatMapRequest:
		m.Content.RequestContent.SeatMapRequest = &seat_mapv2.SeatMapRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.SeatMapRequest)
	case messaging.SeatMapResponse:
		m.Content.ResponseContent.SeatMapResponse = &seat_mapv2.SeatMapResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.SeatMapResponse)
	case messaging.SeatMapAvailabilityRequest:
		m.Content.RequestContent.SeatMapAvailabilityRequest = &seat_mapv2.SeatMapAvailabilityRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.SeatMapAvailabilityRequest)
	case messaging.SeatMapAvailabilityResponse:
		m.Content.ResponseContent.SeatMapAvailabilityResponse = &seat_mapv2.SeatMapAvailabilityResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.SeatMapResponse)
	case messaging.CountryEntryRequirementsRequest:
		m.Content.RequestContent.CountryEntryRequirementsRequest = &infov2.CountryEntryRequirementsRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.CountryEntryRequirementsRequest)
	case messaging.CountryEntryRequirementsResponse:
		m.Content.ResponseContent.CountryEntryRequirementsResponse = &infov2.CountryEntryRequirementsResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.CountryEntryRequirementsResponse)
	case messaging.InsuranceProductInfoRequest:
		m.Content.RequestContent.InsuranceProductInfoRequest = &insurancev1.InsuranceProductInfoRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.InsuranceProductInfoRequest)
	case messaging.InsuranceProductInfoResponse:
		m.Content.ResponseContent.InsuranceProductInfoResponse = &insurancev1.InsuranceProductInfoResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.InsuranceProductInfoResponse)
	case messaging.InsuranceProductListRequest:
		m.Content.RequestContent.InsuranceProductListRequest = &insurancev1.InsuranceProductListRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.InsuranceProductListRequest)
	case messaging.InsuranceProductListResponse:
		m.Content.ResponseContent.InsuranceProductListResponse = &insurancev1.InsuranceProductListResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.InsuranceProductListResponse)
	case messaging.InsuranceSearchRequest:
		m.Content.RequestContent.InsuranceSearchRequest = &insurancev1.InsuranceSearchRequest{}
		return proto.Unmarshal(src, m.Content.RequestContent.InsuranceSearchRequest)
	case messaging.InsuranceSearchResponse:
		m.Content.ResponseContent.InsuranceSearchResponse = &insurancev1.InsuranceSearchResponse{}
		return proto.Unmarshal(src, m.Content.ResponseContent.InsuranceSearchResponse)
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

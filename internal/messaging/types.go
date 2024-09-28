package messaging

import (
	"errors"

	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	infov2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/info/v2"
	insurancev1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/insurance/v1"
	networkv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1"
	notificationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v1"
	partnerv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v2"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	seat_mapv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v2"
	transportv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v2"
	"maunium.net/go/mautrix/id"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"

	"google.golang.org/protobuf/proto"
)

var ErrUnknownMessageType = errors.New("unknown message type")

type RequestContent struct {
	*networkv1.GetNetworkFeeRequest
	*pingv1.PingRequest

	*insurancev1.InsuranceProductInfoRequest
	*insurancev1.InsuranceProductListRequest
	*insurancev1.InsuranceSearchRequest

	*activityv2.ActivityProductInfoRequest
	*activityv2.ActivityProductListRequest
	*activityv2.ActivitySearchRequest
	*accommodationv2.AccommodationProductInfoRequest
	*accommodationv2.AccommodationProductListRequest
	*accommodationv2.AccommodationSearchRequest
	*partnerv2.GetPartnerConfigurationRequest
	*bookv2.MintRequest
	*bookv2.ValidationRequest
	*transportv2.TransportSearchRequest
	*seat_mapv2.SeatMapRequest
	*seat_mapv2.SeatMapAvailabilityRequest
	*infov2.CountryEntryRequirementsRequest
	*notificationv1.TokenBought
	*notificationv1.TokenExpired
}

type ResponseContent struct {
	*networkv1.GetNetworkFeeResponse
	*pingv1.PingResponse
	*insurancev1.InsuranceProductInfoResponse
	*insurancev1.InsuranceProductListResponse
	*insurancev1.InsuranceSearchResponse
	*activityv2.ActivityProductInfoResponse
	*activityv2.ActivityProductListResponse
	*activityv2.ActivitySearchResponse
	*accommodationv2.AccommodationProductInfoResponse
	*accommodationv2.AccommodationProductListResponse
	*accommodationv2.AccommodationSearchResponse
	*partnerv2.GetPartnerConfigurationResponse
	*bookv2.MintResponse
	*bookv2.ValidationResponse
	*transportv2.TransportSearchResponse
	*seat_mapv2.SeatMapResponse
	*seat_mapv2.SeatMapAvailabilityResponse
	*infov2.CountryEntryRequirementsResponse
}

type MessageContent struct {
	RequestContent
	ResponseContent
}

// Message is the message format used for communication between the messenger and the service
type Message struct {
	Type     MessageType       `json:"msgtype"`
	Content  MessageContent    `json:"content"`
	Metadata metadata.Metadata `json:"metadata"`
	Sender   id.UserID
}

type (
	MessageCategory byte
	MessageType     string
)

const (
	// message categories
	Request MessageCategory = iota
	Response
	Unknown

	// message types
	ActivityProductInfoRequest       MessageType = "ActivityProductInfoRequest"
	ActivityProductInfoResponse      MessageType = "ActivityProductInfoResponse"
	ActivityProductListRequest       MessageType = "ActivityProductListRequest"
	ActivityProductListResponse      MessageType = "ActivityProductListResponse"
	ActivitySearchRequest            MessageType = "ActivitySearchRequest"
	ActivitySearchResponse           MessageType = "ActivitySearchResponse"
	AccommodationProductInfoRequest  MessageType = "AccommodationProductInfoRequest"
	AccommodationProductInfoResponse MessageType = "AccommodationProductInfoResponse"
	AccommodationProductListRequest  MessageType = "AccommodationProductListRequest"
	AccommodationProductListResponse MessageType = "AccommodationProductListResponse"
	AccommodationSearchRequest       MessageType = "AccommodationSearchRequest"
	AccommodationSearchResponse      MessageType = "AccommodationSearchResponse"
	GetNetworkFeeRequest             MessageType = "GetNetworkFeeRequest"
	GetNetworkFeeResponse            MessageType = "GetNetworkFeeResponse"
	GetPartnerConfigurationRequest   MessageType = "GetPartnerConfigurationRequest"
	GetPartnerConfigurationResponse  MessageType = "GetPartnerConfigurationResponse"
	MintRequest                      MessageType = "MintRequest"
	MintResponse                     MessageType = "MintResponse"
	ValidationRequest                MessageType = "ValidationRequest"
	ValidationResponse               MessageType = "ValidationResponse"
	PingRequest                      MessageType = "PingRequest"
	PingResponse                     MessageType = "PingResponse"
	TransportSearchRequest           MessageType = "TransportSearchRequest"
	TransportSearchResponse          MessageType = "TransportSearchResponse"
	SeatMapRequest                   MessageType = "SeatMapRequest"
	SeatMapResponse                  MessageType = "SeatMapResponse"
	SeatMapAvailabilityRequest       MessageType = "SeatMapAvailabilityRequest"
	SeatMapAvailabilityResponse      MessageType = "SeatMapAvailabilityResponse"
	CountryEntryRequirementsRequest  MessageType = "CountryEntryRequirementsRequest"
	CountryEntryRequirementsResponse MessageType = "CountryEntryRequirementsResponse"
	InsuranceProductInfoRequest      MessageType = "InsuranceProductInfoRequest"
	InsuranceProductInfoResponse     MessageType = "InsuranceProductInfoResponse"
	InsuranceProductListRequest      MessageType = "InsuranceProductListRequest"
	InsuranceProductListResponse     MessageType = "InsuranceProductListResponse"
	InsuranceSearchRequest           MessageType = "InsuranceSearchRequest"
	InsuranceSearchResponse          MessageType = "InsuranceSearchResponse"
)

func (mt MessageType) Category() MessageCategory {
	switch mt {
	case ActivityProductInfoRequest,
		ActivityProductListRequest,
		ActivitySearchRequest,
		AccommodationProductInfoRequest,
		AccommodationProductListRequest,
		AccommodationSearchRequest,
		MintRequest,
		ValidationRequest,
		GetNetworkFeeRequest,
		GetPartnerConfigurationRequest,
		PingRequest,
		TransportSearchRequest,
		SeatMapRequest,
		SeatMapAvailabilityRequest,
		CountryEntryRequirementsRequest,
		InsuranceProductInfoRequest,
		InsuranceProductListRequest,
		InsuranceSearchRequest:
		return Request
	case ActivityProductInfoResponse,
		ActivityProductListResponse,
		ActivitySearchResponse,
		AccommodationProductInfoResponse,
		AccommodationProductListResponse,
		AccommodationSearchResponse,
		GetNetworkFeeResponse,
		GetPartnerConfigurationResponse,
		MintResponse,
		ValidationResponse,
		PingResponse,
		TransportSearchResponse,
		SeatMapResponse,
		SeatMapAvailabilityResponse,
		CountryEntryRequirementsResponse,
		InsuranceProductInfoResponse,
		InsuranceProductListResponse,
		InsuranceSearchResponse:
		return Response
	default:
		return Unknown
	}
}

func (m *Message) MarshalContent() ([]byte, error) {
	switch m.Type {
	case ActivityProductListRequest:
		return proto.Marshal(m.Content.ActivityProductListRequest)
	case ActivityProductListResponse:
		return proto.Marshal(m.Content.ActivityProductListResponse)
	case ActivitySearchRequest:
		return proto.Marshal(m.Content.ActivitySearchRequest)
	case ActivitySearchResponse:
		return proto.Marshal(m.Content.ActivitySearchResponse)
	case ActivityProductInfoRequest:
		return proto.Marshal(m.Content.ActivityProductInfoRequest)
	case ActivityProductInfoResponse:
		return proto.Marshal(m.Content.ActivityProductInfoResponse)
	case AccommodationProductInfoRequest:
		return proto.Marshal(m.Content.AccommodationProductInfoRequest)
	case AccommodationProductInfoResponse:
		return proto.Marshal(m.Content.AccommodationProductInfoResponse)
	case AccommodationProductListRequest:
		return proto.Marshal(m.Content.AccommodationProductListRequest)
	case AccommodationProductListResponse:
		return proto.Marshal(m.Content.AccommodationProductListResponse)
	case AccommodationSearchRequest:
		return proto.Marshal(m.Content.AccommodationSearchRequest)
	case AccommodationSearchResponse:
		return proto.Marshal(m.Content.AccommodationSearchResponse)
	case GetNetworkFeeRequest:
		return proto.Marshal(m.Content.GetNetworkFeeRequest)
	case GetNetworkFeeResponse:
		return proto.Marshal(m.Content.GetNetworkFeeResponse)
	case GetPartnerConfigurationRequest:
		return proto.Marshal(m.Content.GetPartnerConfigurationRequest)
	case GetPartnerConfigurationResponse:
		return proto.Marshal(m.Content.GetPartnerConfigurationResponse)
	case MintRequest:
		return proto.Marshal(m.Content.MintRequest)
	case MintResponse:
		return proto.Marshal(m.Content.MintResponse)
	case ValidationRequest:
		return proto.Marshal(m.Content.ValidationRequest)
	case ValidationResponse:
		return proto.Marshal(m.Content.ValidationResponse)
	case PingRequest:
		return proto.Marshal(m.Content.PingRequest)
	case PingResponse:
		return proto.Marshal(m.Content.PingResponse)
	case TransportSearchRequest:
		return proto.Marshal(m.Content.TransportSearchRequest)
	case TransportSearchResponse:
		return proto.Marshal(m.Content.TransportSearchResponse)
	case SeatMapRequest:
		return proto.Marshal(m.Content.SeatMapRequest)
	case SeatMapResponse:
		return proto.Marshal(m.Content.SeatMapResponse)
	case SeatMapAvailabilityRequest:
		return proto.Marshal(m.Content.SeatMapAvailabilityRequest)
	case SeatMapAvailabilityResponse:
		return proto.Marshal(m.Content.SeatMapAvailabilityResponse)
	case CountryEntryRequirementsRequest:
		return proto.Marshal(m.Content.CountryEntryRequirementsRequest)
	case CountryEntryRequirementsResponse:
		return proto.Marshal(m.Content.CountryEntryRequirementsResponse)
	case InsuranceProductInfoRequest:
		return proto.Marshal(m.Content.InsuranceProductInfoRequest)
	case InsuranceProductInfoResponse:
		return proto.Marshal(m.Content.InsuranceProductInfoResponse)
	case InsuranceProductListRequest:
		return proto.Marshal(m.Content.InsuranceProductListRequest)
	case InsuranceProductListResponse:
		return proto.Marshal(m.Content.InsuranceProductListResponse)
	case InsuranceSearchRequest:
		return proto.Marshal(m.Content.InsuranceSearchRequest)
	case InsuranceSearchResponse:
		return proto.Marshal(m.Content.InsuranceSearchResponse)

	default:
		return nil, ErrUnknownMessageType
	}
}

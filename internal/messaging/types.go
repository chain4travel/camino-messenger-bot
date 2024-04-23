package messaging

import (
	"errors"

	accommodationv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha"
	activityv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1alpha"
	bookv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1alpha"
	networkv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1alpha"
	partnerv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1alpha"
	pingv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1alpha"
	transportv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1alpha"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"

	"google.golang.org/protobuf/proto"
)

var ErrUnknownMessageType = errors.New("unknown message type")

//nolint:govet // struct can only contain on of the embedded types
type RequestContent struct {
	activityv1alpha.ActivityProductListRequest
	activityv1alpha.ActivitySearchRequest
	accommodationv1alpha.AccommodationProductInfoRequest
	accommodationv1alpha.AccommodationProductListRequest
	accommodationv1alpha.AccommodationSearchRequest
	networkv1alpha.GetNetworkFeeRequest
	partnerv1alpha.GetPartnerConfigurationRequest
	bookv1alpha.MintRequest
	bookv1alpha.ValidationRequest
	pingv1alpha.PingRequest
	transportv1alpha.TransportSearchRequest
}

//nolint:govet // struct can only contain on of the embedded types
type ResponseContent struct {
	activityv1alpha.ActivityProductListResponse
	activityv1alpha.ActivitySearchResponse
	accommodationv1alpha.AccommodationProductInfoResponse
	accommodationv1alpha.AccommodationProductListResponse
	accommodationv1alpha.AccommodationSearchResponse
	networkv1alpha.GetNetworkFeeResponse
	partnerv1alpha.GetPartnerConfigurationResponse
	bookv1alpha.MintResponse
	bookv1alpha.ValidationResponse
	pingv1alpha.PingResponse
	transportv1alpha.TransportSearchResponse
}

//nolint:govet // struct can only contain on of the embedded types
type MessageContent struct {
	RequestContent
	ResponseContent
}

// Message is the message format used for communication between the messenger and the service
type Message struct {
	Type     MessageType       `json:"msgtype"`
	Content  MessageContent    `json:"content"`
	Metadata metadata.Metadata `json:"metadata"`
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
)

func (mt MessageType) Category() MessageCategory {
	switch mt {
	case ActivityProductListRequest,
		ActivitySearchRequest,
		AccommodationProductInfoRequest,
		AccommodationProductListRequest,
		AccommodationSearchRequest,
		MintRequest,
		ValidationRequest,
		PingRequest,
		TransportSearchRequest:
		return Request
	case ActivityProductListResponse,
		ActivitySearchResponse,
		AccommodationProductInfoResponse,
		AccommodationProductListResponse,
		AccommodationSearchResponse,
		GetNetworkFeeResponse,
		GetPartnerConfigurationResponse,
		MintResponse,
		ValidationResponse,
		PingResponse,
		TransportSearchResponse:
		return Response
	default:
		return Unknown
	}
}

func (m *Message) MarshalContent() ([]byte, error) {
	switch m.Type {
	case ActivityProductListRequest:
		return proto.Marshal(&m.Content.ActivityProductListRequest)
	case ActivityProductListResponse:
		return proto.Marshal(&m.Content.ActivityProductListResponse)
	case ActivitySearchRequest:
		return proto.Marshal(&m.Content.ActivitySearchRequest)
	case ActivitySearchResponse:
		return proto.Marshal(&m.Content.ActivitySearchResponse)
	case AccommodationProductInfoRequest:
		return proto.Marshal(&m.Content.AccommodationProductInfoRequest)
	case AccommodationProductInfoResponse:
		return proto.Marshal(&m.Content.AccommodationProductInfoResponse)
	case AccommodationProductListRequest:
		return proto.Marshal(&m.Content.AccommodationProductListRequest)
	case AccommodationProductListResponse:
		return proto.Marshal(&m.Content.AccommodationProductListResponse)
	case AccommodationSearchRequest:
		return proto.Marshal(&m.Content.AccommodationSearchRequest)
	case AccommodationSearchResponse:
		return proto.Marshal(&m.Content.AccommodationSearchResponse)
	case GetNetworkFeeRequest:
		return proto.Marshal(&m.Content.GetNetworkFeeRequest)
	case GetNetworkFeeResponse:
		return proto.Marshal(&m.Content.GetNetworkFeeResponse)
	case GetPartnerConfigurationRequest:
		return proto.Marshal(&m.Content.GetPartnerConfigurationRequest)
	case GetPartnerConfigurationResponse:
		return proto.Marshal(&m.Content.GetPartnerConfigurationResponse)
	case MintRequest:
		return proto.Marshal(&m.Content.MintRequest)
	case MintResponse:
		return proto.Marshal(&m.Content.MintResponse)
	case ValidationRequest:
		return proto.Marshal(&m.Content.ValidationRequest)
	case ValidationResponse:
		return proto.Marshal(&m.Content.ValidationResponse)
	case PingRequest:
		return proto.Marshal(&m.Content.PingRequest)
	case PingResponse:
		return proto.Marshal(&m.Content.PingResponse)
	case TransportSearchRequest:
		return proto.Marshal(&m.Content.TransportSearchRequest)
	case TransportSearchResponse:
		return proto.Marshal(&m.Content.TransportSearchResponse)
	default:
		return nil, ErrUnknownMessageType
	}
}

package messaging

import (
	accommodationv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha"
	activityv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1alpha"
	networkv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1alpha"
	partnerv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1alpha"
	pingv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1alpha"
	transportv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1alpha"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/golang/protobuf/proto"
)

type RequestContent struct {
	activityv1alpha.ActivitySearchRequest
	accommodationv1alpha.AccommodationSearchRequest
	networkv1alpha.GetNetworkFeeRequest
	partnerv1alpha.GetPartnerConfigurationRequest
	pingv1alpha.PingRequest
	transportv1alpha.TransportSearchRequest
}
type ResponseContent struct {
	activityv1alpha.ActivitySearchResponse
	accommodationv1alpha.AccommodationSearchResponse
	networkv1alpha.GetNetworkFeeResponse
	partnerv1alpha.GetPartnerConfigurationResponse
	pingv1alpha.PingResponse
	transportv1alpha.TransportSearchResponse
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
}

type MessageCategory byte
type MessageType string

const (
	// message categories
	Request MessageCategory = iota
	Response
	Unknown

	// message types

	ActivitySearchRequest           MessageType = "ActivitySearchRequest"
	ActivitySearchResponse          MessageType = "ActivitySearchResponse"
	AccommodationSearchRequest      MessageType = "AccommodationSearchRequest"
	AccommodationSearchResponse     MessageType = "AccommodationSearchResponse"
	GetNetworkFeeRequest            MessageType = "GetNetworkFeeRequest"
	GetNetworkFeeResponse           MessageType = "GetNetworkFeeResponse"
	GetPartnerConfigurationRequest  MessageType = "GetPartnerConfigurationRequest"
	GetPartnerConfigurationResponse MessageType = "GetPartnerConfigurationResponse"
	PingRequest                     MessageType = "PingRequest"
	PingResponse                    MessageType = "PingResponse"
	TransportSearchRequest          MessageType = "TransportSearchRequest"
	TransportSearchResponse         MessageType = "TransportSearchResponse"
)

func (mt MessageType) Category() MessageCategory {
	switch mt {
	case ActivitySearchRequest,
		AccommodationSearchRequest,
		PingRequest,
		TransportSearchRequest:
		return Request
	case ActivitySearchResponse,
		AccommodationSearchResponse,
		GetNetworkFeeResponse,
		GetPartnerConfigurationResponse,
		PingResponse,
		TransportSearchResponse:
		return Response
	default:
		return Unknown
	}
}

func (m *Message) MarshalContent() ([]byte, error) {

	switch m.Type {
	case ActivitySearchRequest:
		return proto.Marshal(&m.Content.ActivitySearchRequest)
	case ActivitySearchResponse:
		return proto.Marshal(&m.Content.ActivitySearchResponse)
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
	case PingRequest:
		return proto.Marshal(&m.Content.PingRequest)
	case PingResponse:
		return proto.Marshal(&m.Content.PingResponse)
	case TransportSearchRequest:
		return proto.Marshal(&m.Content.TransportSearchRequest)
	case TransportSearchResponse:
		return proto.Marshal(&m.Content.TransportSearchResponse)
	default:
		return nil, ErrInvalidMessageType
	}
}

package messages

import (
	"errors"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"maunium.net/go/mautrix/id"
)

var ErrUnknownMessageType = errors.New("unknown message type")

// Message is the message format used for communication between the messenger and the service
type Message struct {
	Type     MessageType               `json:"msgtype"`
	Content  protoreflect.ProtoMessage `json:"content"`
	Metadata metadata.Metadata         `json:"metadata"`
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
	return proto.Marshal(m.Content)
}

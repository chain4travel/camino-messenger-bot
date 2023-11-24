package messaging

import (
	accommodationv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha1"
	pingv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1alpha1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/golang/protobuf/proto"
)

type RequestContent struct {
	accommodationv1alpha1.AccommodationSearchRequest
	pingv1alpha1.PingRequest
}
type ResponseContent struct {
	accommodationv1alpha1.AccommodationSearchResponse
	pingv1alpha1.PingResponse
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

	AccommodationSearchRequest  MessageType = "AccommodationSearchRequest"
	AccommodationSearchResponse MessageType = "AccommodationSearchResponse"
	PingRequest                 MessageType = "PingRequest"
	PingResponse                MessageType = "PingResponse"
)

func (mt MessageType) Category() MessageCategory {
	switch mt {
	case AccommodationSearchRequest,
		PingRequest:
		return Request
	case AccommodationSearchResponse,
		PingResponse:
		return Response
	default:
		return Unknown
	}
}

func (m *Message) MarshalContent() ([]byte, error) {

	switch m.Type {
	case AccommodationSearchRequest:
		return proto.Marshal(&m.Content.AccommodationSearchRequest)
	case AccommodationSearchResponse:
		return proto.Marshal(&m.Content.AccommodationSearchResponse)
	case PingRequest:
		return proto.Marshal(&m.Content.PingRequest)
	case PingResponse:
		return proto.Marshal(&m.Content.PingResponse)
	default:
		return nil, ErrInvalidMessageType
	}
}

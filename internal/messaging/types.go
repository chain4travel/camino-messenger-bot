package messaging

import (
	"encoding/json"

	accommodationv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
)

type RequestContent struct {
	accommodationv1alpha1.AccommodationSearchRequest
}
type ResponseContent struct {
	accommodationv1alpha1.AccommodationSearchResponse
}
type MessageContent struct {
	RequestContent
	ResponseContent
}
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
)

func (mt MessageType) Category() MessageCategory {
	switch mt {
	case AccommodationSearchRequest:
		return Request
	case AccommodationSearchResponse:
		return Response
	default:
		return Unknown
	}
}

func (mc *MessageContent) ToJSON() (string, error) {
	jsonData, err := json.Marshal(mc)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

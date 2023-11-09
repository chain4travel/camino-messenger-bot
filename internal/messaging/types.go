package messaging

import (
	"encoding/json"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/proto/pb/messages"
)

type RequestContent struct {
	messages.FlightSearchRequest
}
type ResponseContent struct {
	messages.FlightSearchResponse
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

	HotelAvailRequest  MessageType = "C4TContentHotelAvailRequest"
	HotelAvailResponse MessageType = "C4TContentHotelAvailResponse"

	HotelBookRequest  MessageType = "C4TContentHotelBookRequest"
	HotelBookResponse MessageType = "C4TContentHotelBookResponse"

	HotelMappingsRequest  MessageType = "C4TContentHotelMappingsRequest"
	HotelMappingsResponse MessageType = "C4TContentHotelMappingsResponse"

	FlightSearchRequest  MessageType = "C4TContentFlightSearchRequest"
	FlightSearchResponse MessageType = "C4TContentFlightSearchResponse"

	FlightBookRequest  MessageType = "C4TContentFlightBookRequest"
	FlightBookResponse MessageType = "C4TContentFlightBookResponse"

	FlightInfoRequest  MessageType = "C4TContentFlightInfoRequest"
	FlightInfoResponse MessageType = "C4TContentFlightInfoResponse"
)

func (mt MessageType) Category() MessageCategory {
	switch mt {
	case HotelAvailRequest,
		HotelBookRequest,
		HotelMappingsRequest,
		FlightSearchRequest,
		FlightBookRequest,
		FlightInfoRequest:
		return Request
	case HotelAvailResponse,
		HotelBookResponse,
		HotelMappingsResponse,
		FlightSearchResponse,
		FlightBookResponse,
		FlightInfoResponse:
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

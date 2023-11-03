package messaging

type MessageCategory byte
type MessageType string

const (
	// message categories
	Request MessageCategory = iota
	Response
	Unknown

	// message types
	DummyResponse MessageType = "m.text" // TODO remove

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
		FlightInfoResponse,
		DummyResponse:
		return Response
	default:
		return Unknown
	}
}

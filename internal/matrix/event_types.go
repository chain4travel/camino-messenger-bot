package matrix

type MessageCategory byte
type MessageType string
type RoomEventType string

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

	// event types
	RoomMember RoomEventType = "m.room.member"
	CaminoMsg  RoomEventType = "m.room.c4t-msg"
)

type MessageRequestContent struct {
	Body    string                   `json:"body,omitempty"`
	Cheques []map[string]interface{} `json:"cheques,omitempty"`
}
type MessageResponseContent struct {
	ContentUri string `json:"contentUri,omitempty"`
	roomID     string `json:"inReplyToRoomId,omitempty"`
	eventID    string `json:"inReplyToEventId,omitempty"`
}
type TimelineEventContent struct {
	Type MessageType `json:"msgtype,omitempty"`
	MessageRequestContent
	MessageResponseContent
}

type StateEventContent struct {
	Membership string `json:"membership,omitempty"`
}
type RoomEventContent struct {
	StateEventContent
	TimelineEventContent
}

type RoomEvent struct {
	Type     RoomEventType    `json:"type"`
	Content  RoomEventContent `json:"content"`
	Sender   string           `json:"sender"`
	StateKey string           `json:"state_key"`
}

type Room struct {
	Timeline struct {
		Events []RoomEvent `json:"events"`
	} `json:"timeline"`
	Invite struct {
		Events []RoomEvent `json:"events"`
	} `json:"invite_state"`
}

//	type RoomInviteEvents struct {
//		Room `json:"invite_state.events"`
//	}
//
//	type RoomTimelineEvents struct {
//		Room `json:"timeline.events"`
//	}

type JoinedRooms map[string]Room
type InviteRooms map[string]Room
type Rooms struct {
	Join   JoinedRooms `json:"join"`
	Invite InviteRooms `json:"invite"`
}

type SyncResponse struct {
	NextBatch string `json:"next_batch"`
	Rooms     Rooms  `json:"rooms"`
}

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

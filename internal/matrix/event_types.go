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
	DummyRequest  MessageType = "Request"
	DummyResponse MessageType = "Response"

	// event types
	RoomMember RoomEventType = "m.room.member"
	CaminoMsg  RoomEventType = "m.room.camino-msg"
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

type RoomEvents struct {
	Events []RoomEvent
}
type RoomInviteEvents struct {
	RoomEvents `json:"invite_state.events"`
}
type RoomTimelineEvents struct {
	RoomEvents `json:"timeline.events"`
}
type Rooms map[string]RoomEvents

func (mt MessageType) Category() MessageCategory {
	switch mt {
	case DummyRequest:
		return Request
	case DummyResponse:
		return Response
	default:
		return Unknown
	}
}

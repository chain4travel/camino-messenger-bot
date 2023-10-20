package messaging

import (
	"camino-messenger-provider/internal/matrix"
	wrappers "camino-messenger-provider/internal/utils"
	"errors"
)

var (
	_                             Processor = (*processor)(nil)
	ErrUserIDNotSet                         = errors.New("user id not set")
	ErrOnlyRequestMessagesAllowed           = errors.New("only request messages allowed")
)

type Processor interface {
	ProcessRoomInvitation(events matrix.Rooms) error
	ProcessMessage(message matrix.Rooms) error
}

type processor struct {
	matrixClient matrix.Client
	userID       string

	requester requester
	responder responder
}

func NewProcessor(matrixClient matrix.Client, userID string) Processor {
	return &processor{
		matrixClient: matrixClient,
		userID:       userID,
		requester:    requester{matrixClient: matrixClient},
		responder:    responder{},
	}
}

func (p *processor) ProcessRoomInvitation(rooms matrix.Rooms) error {
	errs := wrappers.Errs{}
	for roomID, events := range rooms {
		go func() {
			for _, event := range events.Events {
				if matrix.RoomEventType(event.Type) == matrix.RoomMember && event.Content.Membership == "invite" {
					errs.Add(p.matrixClient.Join(roomID))
				}
			}
		}()
	}
	return errs.Err
}

func (p *processor) ProcessMessage(rooms matrix.Rooms) error {
	if p.userID == "" {
		return ErrUserIDNotSet
	}
	errs := wrappers.Errs{}
	for roomID, events := range rooms {
		go func() {
			for _, event := range events.Events {

				if event.Type == matrix.CaminoMsg && event.Sender == p.userID { // outbound messages = messages sent by own ext system
					if event.Content.Type.Category() == matrix.Request { // only request messages (received by are processed
						errs.Add(p.requester.Write(roomID, event.Content.TimelineEventContent)) // forward request message to matrix
					} else {
						errs.Add(ErrOnlyRequestMessagesAllowed) // ignore msg
					}
				} else if event.Type == matrix.CaminoMsg { // inbound messages = messages sent by other systems via messenger

					switch event.Content.Type.Category() {
					case matrix.Request:
						//TODO talk to legacy system and get response

						msg := matrix.TimelineEventContent{
							Type:                  "",
							MessageRequestContent: matrix.MessageRequestContent{},
						}
						errs.Add(p.requester.Write(roomID, msg))
					case matrix.Response:
						// TODO forward response to legacy system (grpc client)
						msg := matrix.TimelineEventContent{
							Type:                   "",
							MessageResponseContent: matrix.MessageResponseContent{},
						}
						errs.Add(p.responder.Write(roomID, msg))
					}
				}
			}
		}()
	}
	return errs.Err
}

package messaging

import (
	"camino-messenger-provider/internal/matrix"
	"camino-messenger-provider/internal/proto/pb"
	"camino-messenger-provider/internal/rpc/client"
	wrappers "camino-messenger-provider/internal/utils"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
)

var (
	_                             Processor = (*processor)(nil)
	ErrUserIDNotSet                         = errors.New("user id not set")
	ErrOnlyRequestMessagesAllowed           = errors.New("only request messages allowed")
)

type MsgProcessor interface {
	Request(roomID string, msg *matrix.TimelineEventContent) error
	Respond(extSystem *client.RPCClient, roomID string, event *matrix.RoomEvent) error
	Forward(extSystem *client.RPCClient, msg *matrix.TimelineEventContent) error
}
type Processor interface {
	MsgProcessor
	Start(ctx context.Context)
	RoomChannel() chan<- matrix.InviteRooms
	MsgChannel() chan<- matrix.JoinedRooms
	ProcessRoomInvitation(events matrix.InviteRooms) error
	ProcessMessage(message matrix.JoinedRooms) error
}

type processor struct {
	matrixClient matrix.Client
	rpcClient    *client.RPCClient
	userID       string
	logger       *zap.SugaredLogger
	roomChannel  chan matrix.InviteRooms
	msgChannel   chan matrix.JoinedRooms
}

func NewProcessor(matrixClient matrix.Client, rpcClient *client.RPCClient, userID string, logger *zap.SugaredLogger) Processor {
	return &processor{
		matrixClient: matrixClient,
		rpcClient:    rpcClient,
		userID:       userID,
		logger:       logger,
		roomChannel:  make(chan matrix.InviteRooms),
		msgChannel:   make(chan matrix.JoinedRooms),
	}
}

func (p *processor) Start(ctx context.Context) {
	//TODO start multiple routines??

	go func() {
		select {
		case roomEvent := <-p.roomChannel:
			p.logger.Debug("Received room event")
			err := p.ProcessRoomInvitation(roomEvent)
			if err != nil {
				p.logger.Error(err)
			}
		case msgEvent := <-p.msgChannel:
			p.logger.Debug("Received msg event")
			err := p.ProcessMessage(msgEvent)
			if err != nil {
				p.logger.Error(err)
			}
		case <-ctx.Done():
			p.logger.Info("Stopping processor...")
			return
		}
	}()

}

func (p *processor) RoomChannel() chan<- matrix.InviteRooms {
	return p.roomChannel
}

func (p *processor) MsgChannel() chan<- matrix.JoinedRooms {
	return p.msgChannel
}

func (p *processor) ProcessRoomInvitation(rooms matrix.InviteRooms) error {
	errs := wrappers.Errs{}
	for roomID, events := range rooms {
		go func() {
			for _, event := range events.Invite.Events {
				if event.Type == matrix.RoomMember && event.Content.Membership == "invite" {
					errs.Add(p.matrixClient.Join(roomID))
				}
			}
		}()
	}
	return errs.Err
}

func (p *processor) ProcessMessage(rooms matrix.JoinedRooms) error {
	if p.userID == "" {
		return ErrUserIDNotSet
	}
	errs := wrappers.Errs{}
	for roomID, events := range rooms {
		go func() {
			for _, event := range events.Timeline.Events {
				if event.Type == matrix.CaminoMsg && event.Sender == p.userID { // outbound messages = messages sent by own ext system
					if event.Content.Type.Category() == matrix.Request { // only request messages (received by are processed
						errs.Add(p.Request(roomID, &event.Content.TimelineEventContent)) // forward request message to matrix
					} else {
						p.logger.Debugf("Ignoring outgoing response message to matrix from sender: %s ", event.Sender)
						errs.Add(ErrOnlyRequestMessagesAllowed) // ignore msg
					}
				} else if event.Type == matrix.CaminoMsg { // inbound messages = messages sent by other systems via messenger
					switch event.Content.Type.Category() {
					case matrix.Request:
						errs.Add(p.Respond(p.rpcClient, roomID, &event))
					case matrix.Response:
						errs.Add(p.Forward(p.rpcClient, &event.Content.TimelineEventContent))
					case matrix.Unknown:
						p.logger.Debugf("Ignoring incoming  message of unknown category: %s ", event.Content.Type)
					}
				}
			}
		}()
	}
	return errs.Err
}

func (p *processor) Request(roomID string, msg *matrix.TimelineEventContent) error {
	p.logger.Debugf("Sending outgoing request message to room: %s ", roomID)
	// issue and attach cheques
	msg.Cheques = nil //TODO
	// forward msg to matrix
	return p.matrixClient.Send(roomID, *msg)
}

func (p *processor) Respond(extSystem *client.RPCClient, roomID string, event *matrix.RoomEvent) error {

	fmt.Println(extSystem)
	request := &pb.GreetingServiceRequest{Name: string(event.Content.Type)}
	resp, err := extSystem.Gsc.Greeting(context.Background(), request)
	if err != nil { //TODO retry mechanism?
		return err
	}
	//TODO talk to legacy system and get response
	msg := matrix.TimelineEventContent{
		Type:                  "",
		MessageRequestContent: matrix.MessageRequestContent{Body: resp.Message},
	}
	p.logger.Debugf("Responding to incoming request message with: %s ", msg.Body)
	return p.matrixClient.Send(roomID, msg)
}

func (p *processor) Forward(extSystem *client.RPCClient, msg *matrix.TimelineEventContent) error {
	// TODO forward response to legacy system (grpc client)
	request := &pb.GreetingServiceRequest{Name: string(msg.Type)}
	resp, err := extSystem.Gsc.Greeting(context.Background(), request)
	if err != nil {
		return err
	}
	p.logger.Debugf("Forwarding response message to ext system: %s ", resp.Message)
	return nil
}

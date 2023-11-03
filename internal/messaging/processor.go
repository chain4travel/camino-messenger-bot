package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	"camino-messenger-bot/internal/proto/pb"
	"camino-messenger-bot/internal/rpc/client"
	"go.uber.org/zap"
)

var (
	_                             Processor = (*processor)(nil)
	ErrUserIDNotSet                         = errors.New("user id not set")
	ErrUnknownMessageCategory               = errors.New("unknown message category")
	ErrOnlyRequestMessagesAllowed           = errors.New("only request messages allowed")
)

type MsgHandler interface {
	Request(msg Message) (Message, error)
	Respond(extSystem *client.RPCClient, msg Message) error
	Forward(msg Message)
}
type Processor interface {
	MsgHandler
	Start(ctx context.Context)
	ProcessInbound(message Message) error
	ProcessOutbound(message Message) (Message, error)
}

type processor struct {
	messenger       Messenger
	rpcClient       *client.RPCClient
	userID          string
	logger          *zap.SugaredLogger
	timeout         time.Duration // timeout after which a request is considered failed
	responseChannel chan Message
}

func NewProcessor(messenger Messenger, rpcClient *client.RPCClient, userID string, logger *zap.SugaredLogger, timeout time.Duration) Processor {
	return &processor{
		messenger:       messenger,
		rpcClient:       rpcClient,
		userID:          userID,
		logger:          logger,
		timeout:         timeout,            // for now applies to all request types
		responseChannel: make(chan Message), // channel where only response messages are routed
	}
}

func (p *processor) Start(ctx context.Context) {
	//TODO start multiple routines??

	go func() {
		select {
		case msgEvent := <-p.messenger.Inbound():
			p.logger.Debug("Received msg event")
			err := p.ProcessInbound(msgEvent)
			if err != nil {
				p.logger.Error(err)
			}
		case <-ctx.Done():
			p.logger.Info("Stopping processor...")
			return
		}
	}()

}

func (p *processor) ProcessInbound(msg Message) error {
	if p.userID == "" {
		return ErrUserIDNotSet
	}
	if msg.Metadata.Sender != p.userID { // outbound messages = messages sent by own ext system
		switch msg.Type.Category() {
		case Request:
			return p.Respond(p.rpcClient, msg)
		case Response:
			p.Forward(msg)
			return nil
		case Unknown:
			p.logger.Debugf("Ignoring incoming msg of unknown category: %s ", msg.Type)
			return ErrUnknownMessageCategory
		default:
			p.logger.Debugf("Ignoring incoming msg of category: %s ", msg.Type)
			return ErrOnlyRequestMessagesAllowed // ignore msg
		}
	} else {
		p.logger.Debug("Ignoring own outbound messages")
		return ErrOnlyRequestMessagesAllowed // ignore msg
	}
}

func (p *processor) ProcessOutbound(msg Message) (Message, error) {
	if msg.Metadata.Sender == p.userID && msg.Type.Category() == Request { // only request messages (received by are processed
		return p.Request(msg) // forward request msg to matrix
	} else {
		p.logger.Debugf("Ignoring any non-request message from sender other than: %s ", p.userID)
		return Message{}, ErrOnlyRequestMessagesAllowed // ignore msg
	}
}

func (p *processor) Request(msg Message) (Message, error) {
	p.logger.Debug("Sending outgoing request message")
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	msg.Metadata.Cheques = nil //TODO issue and attach cheques
	err := p.messenger.SendAsync(msg)
	if err != nil {
		return Message{}, err
	}

	for {
		select {
		case response := <-p.responseChannel:
			if response.RequestID == msg.RequestID {
				return response, nil
			}
		case <-ctx.Done():
			return Message{}, fmt.Errorf("response exceeded configured timeout of %v seconds", p.timeout)
		}
	}
}

func (p *processor) Respond(extSystem *client.RPCClient, msg Message) error {
	fmt.Println(extSystem)
	request := &pb.GreetingServiceRequest{Name: string(msg.Type)}
	resp, err := extSystem.Gsc.Greeting(context.Background(), request)
	if err != nil { //TODO retry mechanism?
		return err
	}
	//TODO talk to legacy system and get response
	// add metadata?
	responseMsg := Message{
		Type: "",
	}
	p.logger.Debugf("Responding to incoming request message with: %s ", resp.Message)
	return p.messenger.SendAsync(responseMsg)
}

func (p *processor) Forward(msg Message) {
	p.responseChannel <- msg
}

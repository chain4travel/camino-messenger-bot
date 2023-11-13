package messaging

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	"github.com/chain4travel/camino-messenger-bot/proto/pb/messages"
	"go.uber.org/zap"
	grpc_metadata "google.golang.org/grpc/metadata"
)

type InvalidMessageError struct {
	error
}

var (
	_ Processor = (*processor)(nil)

	ErrUserIDNotSet               = errors.New("user id not set")
	ErrUnknownMessageCategory     = errors.New("unknown message category")
	ErrOnlyRequestMessagesAllowed = errors.New("only request messages allowed")
	ErrUnsupportedRequestType     = errors.New("unsupported request type")
)

type MsgHandler interface {
	Request(ctx context.Context, msg Message) (Message, error)
	Respond(extSystem *client.RPCClient, msg Message) error
	Forward(msg Message)
}
type Processor interface {
	metadata.Checkpoint
	MsgHandler
	SetUserID(userID string)
	Start(ctx context.Context)
	ProcessInbound(message Message) error
	ProcessOutbound(ctx context.Context, message Message) (Message, error)
}

type processor struct {
	cfg       config.ProcessorConfig
	messenger Messenger
	rpcClient *client.RPCClient
	userID    string
	logger    *zap.SugaredLogger
	timeout   time.Duration // timeout after which a request is considered failed
	//responseChannel chan Message

	mu               sync.Mutex
	responseChannels map[string]chan Message
}

func (p *processor) SetUserID(userID string) {
	p.userID = userID
}

func (p *processor) Checkpoint() string {
	return "processor"
}

func NewProcessor(messenger Messenger, rpcClient *client.RPCClient, logger *zap.SugaredLogger, cfg config.ProcessorConfig) Processor {
	return &processor{
		cfg:              cfg,
		messenger:        messenger,
		rpcClient:        rpcClient,
		logger:           logger,
		timeout:          time.Duration(cfg.Timeout) * time.Millisecond, // for now applies to all request types
		responseChannels: make(map[string]chan Message),
	}
}

func (p *processor) Start(ctx context.Context) {
	//TODO start multiple routines??
	for {
		select {
		case msgEvent := <-p.messenger.Inbound():
			p.logger.Debug("Processing msg event of type: ", msgEvent.Type)
			go func() {
				err := p.ProcessInbound(msgEvent)
				if err != nil {
					p.logger.Warnf("could not process message: %v", err)
				}
			}()
		case <-ctx.Done():
			p.logger.Info("Stopping processor...")
			return
		}
	}
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
			return InvalidMessageError{ErrUnknownMessageCategory}
		default:
			return InvalidMessageError{ErrOnlyRequestMessagesAllowed} // ignore msg
		}
	} else {
		return nil // ignore own outbound messages
	}
}

func (p *processor) ProcessOutbound(ctx context.Context, msg Message) (Message, error) {
	if msg.Metadata.Sender == p.userID && msg.Type.Category() == Request { // only request messages (received by are processed
		return p.Request(ctx, msg) // forward request msg to matrix
	} else {
		p.logger.Debugf("Ignoring any non-request message from sender other than: %s ", p.userID)
		return Message{}, ErrOnlyRequestMessagesAllowed // ignore msg
	}
}

func (p *processor) Request(ctx context.Context, msg Message) (Message, error) {
	p.logger.Debug("Sending outbound request message")
	responseChan := make(chan Message)
	p.mu.Lock()
	p.responseChannels[msg.Metadata.RequestID] = responseChan
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.responseChannels, msg.Metadata.RequestID)
		p.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	msg.Metadata.Cheques = nil //TODO issue and attach cheques
	err := p.messenger.SendAsync(ctx, msg)
	if err != nil {
		return Message{}, err
	}

	for {
		select {
		case response := <-responseChan:
			if response.Metadata.RequestID == msg.Metadata.RequestID {
				return response, nil
			}
			//p.logger.Debugf("Ignoring response message with request id: %s, expecting: %s", response.Metadata.RequestID, msg.Metadata.RequestID)
		case <-ctx.Done():
			return Message{}, fmt.Errorf("response exceeded configured timeout of %v seconds for request: %s", p.timeout, msg.Metadata.RequestID)
		}
	}
}

func (p *processor) Respond(extSystem *client.RPCClient, msg Message) error {
	if !p.cfg.SupportedRequestTypes.Contains(string(msg.Type)) {
		return fmt.Errorf("%v: %s", ErrUnsupportedRequestType, msg.Type)
	}
	ctx := grpc_metadata.NewOutgoingContext(context.Background(), msg.Metadata.ToGrpcMD())

	md := msg.Metadata
	md.Sender = p.userID // overwrite sender with actual sender
	md.Stamp(fmt.Sprintf("%s-%s", extSystem.Checkpoint(), "request"))

	//TODO uncomment
	//request := &pb.GreetingServiceRequest{Name: string(msg.Type)}
	//resp, err := extSystem.Sc.Greeting(ctx, request)
	//if err != nil { //TODO retry mechanism?
	//	return err
	//}

	//TODO uncomment? do we need to pass metadata to legacy system?
	//err := md.ExtractMetadata(ctx)
	//if err != nil {
	//	return fmt.Errorf("error extracting metadata")
	//}
	md.Stamp(fmt.Sprintf("%s-%s", extSystem.Checkpoint(), "response"))
	//TODO talk to legacy system and get response
	// add metadata?
	responseMsg := Message{
		Type: FlightSearchResponse,
		Content: MessageContent{
			ResponseContent: ResponseContent{
				messages.FlightSearchResponse{
					Header:            nil,
					Context:           "Hello from mocked legacy system",
					Errors:            "",
					Warnings:          "",
					SupplierCode:      "",
					ExternalSessionId: "",
					SearchId:          msg.Metadata.RequestID,
					Options:           nil,
				},
			},
		},
		Metadata: md,
	}
	return p.messenger.SendAsync(ctx, responseMsg)
}

func (p *processor) Forward(msg Message) {
	p.logger.Debugf("Forwarding outbound response message: %s", msg.Metadata.RequestID)
	//p.responseChannel <- msg
	p.mu.Lock()
	responseChan, ok := p.responseChannels[msg.Metadata.RequestID]
	if ok {
		responseChan <- msg
		close(responseChan)
	}
	p.mu.Unlock()
}

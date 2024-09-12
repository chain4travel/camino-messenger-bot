package messaging

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/ethereum/go-ethereum/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	grpc_metadata "google.golang.org/grpc/metadata"
)

var (
	_ Processor = (*processor)(nil)

	ErrUserIDNotSet               = errors.New("user id not set")
	ErrUnknownMessageCategory     = errors.New("unknown message category")
	ErrOnlyRequestMessagesAllowed = errors.New("only request messages allowed")
	ErrUnsupportedRequestType     = errors.New("unsupported request type")
	ErrMissingRecipient           = errors.New("missing recipient")
	ErrMissingCMAccountRecipient  = errors.New("missing CM Account recipient")
	ErrForeignCMAccount           = errors.New("foreign or Invalid CM Account")
	ErrExceededResponseTimeout    = errors.New("response exceeded configured timeout")
)

type MsgHandler interface {
	Request(ctx context.Context, msg *Message) (*Message, error)
	Respond(msg *Message) error
	Forward(msg *Message)
}
type Processor interface {
	metadata.Checkpoint
	MsgHandler
	SetUserID(userID string)
	Start(ctx context.Context)
	ProcessInbound(message *Message) error
	ProcessOutbound(ctx context.Context, message *Message) (*Message, error)
}

type processor struct {
	cfg       config.ProcessorConfig
	messenger Messenger
	userID    string
	logger    *zap.SugaredLogger
	tracer    trace.Tracer
	timeout   time.Duration // timeout after which a request is considered failed

	mu                    sync.Mutex
	responseChannels      map[string]chan *Message
	serviceRegistry       ServiceRegistry
	responseHandler       ResponseHandler
	identificationHandler IdentificationHandler
}

func (p *processor) SetUserID(userID string) {
	p.userID = userID
}

func (*processor) Checkpoint() string {
	return "processor"
}

func NewProcessor(messenger Messenger, logger *zap.SugaredLogger, cfg config.ProcessorConfig, registry ServiceRegistry, responseHandler ResponseHandler, identificationHandler IdentificationHandler) Processor {
	return &processor{
		cfg:                   cfg,
		messenger:             messenger,
		logger:                logger,
		tracer:                otel.GetTracerProvider().Tracer(""),
		timeout:               time.Duration(cfg.Timeout) * time.Millisecond, // for now applies to all request types
		responseChannels:      make(map[string]chan *Message),
		serviceRegistry:       registry,
		responseHandler:       responseHandler,
		identificationHandler: identificationHandler,
	}
}

func (p *processor) Start(ctx context.Context) {
	for {
		select {
		case msgEvent := <-p.messenger.Inbound():
			p.logger.Debug("Processing msg event of type: ", msgEvent.Type)
			go func() {
				err := p.ProcessInbound(&msgEvent)
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

func (p *processor) ProcessInbound(msg *Message) error {
	if p.userID == "" {
		return ErrUserIDNotSet
	}
	if msg.Metadata.Sender != p.userID { // outbound messages = messages sent by own ext system
		switch msg.Type.Category() {
		case Request:
			return p.Respond(msg)
		case Response:
			p.Forward(msg)
			return nil
		default:
			return ErrUnknownMessageCategory
		}
	} else {
		return nil // ignore own outbound messages
	}
}

func (p *processor) ProcessOutbound(ctx context.Context, msg *Message) (*Message, error) {
	msg.Metadata.Sender = p.userID
	if msg.Type.Category() == Request { // only request messages (received by are processed
		return p.Request(ctx, msg) // forward request msg to matrix
	}
	p.logger.Debugf("Ignoring any non-request message from sender other than: %s ", p.userID)
	return nil, ErrOnlyRequestMessagesAllowed // ignore msg
}

func (p *processor) Request(ctx context.Context, msg *Message) (*Message, error) {
	p.logger.Debug("Sending outbound request message")
	responseChan := make(chan *Message)
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
	if msg.Metadata.Recipient == "" { // TODO: add address validation
		if msg.Metadata.RecipientCMAccount == "" {
			return nil, ErrMissingCMAccountRecipient
		}

		// lookup for CM Account -> bot
		botAddress := CMAccountBotMap[common.HexToAddress(msg.Metadata.RecipientCMAccount)]

		if botAddress == "" {
			// if not in mapping, fetch 1 bot from CM Account
			botAddress, err := p.identificationHandler.getSingleBotFromCMAccountAddress(common.HexToAddress(msg.Metadata.RecipientCMAccount))
			if err != nil {
				return nil, err
			}
			CMAccountBotMap[common.HexToAddress(msg.Metadata.RecipientCMAccount)] = botAddress
			msg.Metadata.Recipient = botAddress
			// TODO: @VjeraTurk forward the message to the correct bot?
		} else {
			msg.Metadata.Recipient = botAddress
			// TODO: @VjeraTurk forward the message to the correct bot?
		}
		// return nil, ErrMissingRecipient
	}
	msg.Metadata.Cheques = nil // TODO issue and attach cheques
	ctx, span := p.tracer.Start(ctx, "processor.Request", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer span.End()

	if err := p.responseHandler.HandleRequest(ctx, msg.Type, &msg.Content.RequestContent); err != nil {
		return nil, err
	}

	err := p.messenger.SendAsync(ctx, *msg)
	if err != nil {
		return nil, err
	}
	ctx, responseSpan := p.tracer.Start(ctx, "processor.AwaitResponse", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer responseSpan.End()
	for {
		select {
		case response := <-responseChan:
			if response.Metadata.RequestID == msg.Metadata.RequestID {
				p.responseHandler.HandleResponse(ctx, msg.Type, &msg.Content.RequestContent, &response.Content.ResponseContent)
				return response, nil
			}
		case <-ctx.Done():
			return nil, fmt.Errorf("%w of %v seconds for request: %s", ErrExceededResponseTimeout, p.timeout, msg.Metadata.RequestID)
		}
	}
}

func (p *processor) Respond(msg *Message) error {
	traceID, err := trace.TraceIDFromHex(msg.Metadata.RequestID)
	if err != nil {
		p.logger.Warnf("failed to parse traceID from hex [requestID:%s]: %v", msg.Metadata.RequestID, err)
	}
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID}))
	ctx, span := p.tracer.Start(ctx, "processor-response", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer span.End()
	var service Service
	var supported bool
	if service, supported = p.serviceRegistry.GetService(msg.Type); !supported {
		return fmt.Errorf("%w: %s", ErrUnsupportedRequestType, msg.Type)
	}

	md := &msg.Metadata
	// rewrite sender & recipient metadata
	md.Recipient = md.Sender
	md.Sender = p.userID
	md.Stamp(fmt.Sprintf("%s-%s", p.Checkpoint(), "request"))

	ctx = grpc_metadata.NewOutgoingContext(ctx, msg.Metadata.ToGrpcMD())
	var header grpc_metadata.MD
	ctx, cspan := p.tracer.Start(ctx, "service.Call", trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attribute.String("type", string(msg.Type))))
	response, msgType, err := service.Call(ctx, &msg.Content.RequestContent, grpc.Header(&header))
	cspan.End()
	if err != nil {
		return err // TODO handle error and return a response message
	}

	err = md.FromGrpcMD(header)
	if err != nil {
		p.logger.Infof("error extracting metadata for request: %s", md.RequestID)
	}

	p.responseHandler.HandleResponse(ctx, msgType, &msg.Content.RequestContent, response)
	responseMsg := Message{
		Type: msgType,
		Content: MessageContent{
			ResponseContent: *response,
		},
		Metadata: *md,
	}
	return p.messenger.SendAsync(ctx, responseMsg)
}

func (p *processor) Forward(msg *Message) {
	p.logger.Debugf("Forwarding outbound response message: %s", msg.Metadata.RequestID)
	p.mu.Lock()
	defer p.mu.Unlock()
	responseChan, ok := p.responseChannels[msg.Metadata.RequestID]
	if ok {
		responseChan <- msg
		close(responseChan)
		return
	}
	p.logger.Warnf("Failed to forward message: no response channel for request (%s)", msg.Metadata.RequestID)
}

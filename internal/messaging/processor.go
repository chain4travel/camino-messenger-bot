package messaging

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/clients"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/messages"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/ethereum/go-ethereum/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	grpc_metadata "google.golang.org/grpc/metadata"
	"maunium.net/go/mautrix/id"
)

const cashInTxIssueTimeout = 10 * time.Second

var (
	_ Processor = (*processor)(nil)

	ErrUserIDNotSet                 = errors.New("user id not set")
	ErrUnknownMessageCategory       = errors.New("unknown message category")
	ErrOnlyRequestMessagesAllowed   = errors.New("only request messages allowed")
	ErrUnsupportedService           = errors.New("unsupported service")
	ErrMissingRecipient             = errors.New("missing recipient")
	ErrForeignCMAccount             = errors.New("foreign or Invalid CM Account")
	ErrExceededResponseTimeout      = errors.New("response exceeded configured timeout")
	ErrMissingCheques               = errors.New("missing cheques in metadata")
	ErrBotNotInCMAccount            = errors.New("bot not in Cm Account")
	ErrCheckingCmAccount            = errors.New("problem calling contract")
	ErrBotMissingChequeOperatorRole = errors.New("bot missing permission")

	networkFee = big.NewInt(300000000000000) // 0.00003 CAM
)

type MsgHandler interface {
	Request(ctx context.Context, msg *messages.Message) (*messages.Message, error)
	Respond(msg *messages.Message) error
	Forward(msg *messages.Message)
}
type Processor interface {
	metadata.Checkpoint
	MsgHandler
	SetUserID(userID id.UserID)
	Start(ctx context.Context)
	ProcessInbound(message *messages.Message) error
	ProcessOutbound(ctx context.Context, message *messages.Message) (*messages.Message, error)
}

func NewProcessor(
	messenger Messenger,
	logger *zap.SugaredLogger,
	cfg config.ProcessorConfig,
	evmConfig config.EvmConfig,
	registry ServiceRegistry,
	responseHandler ResponseHandler,
	identificationHandler IdentificationHandler,
	chequeHandler ChequeHandler,
	compressor compression.Compressor[*messages.Message, [][]byte],
) Processor {
	return &processor{
		cfg:                   cfg,
		evmConfig:             evmConfig,
		messenger:             messenger,
		logger:                logger,
		tracer:                otel.GetTracerProvider().Tracer(""),
		timeout:               time.Duration(cfg.Timeout) * time.Millisecond, // for now applies to all request types
		responseChannels:      make(map[string]chan *messages.Message),
		serviceRegistry:       registry,
		responseHandler:       responseHandler,
		identificationHandler: identificationHandler,
		chequeHandler:         chequeHandler,
		compressor:            compressor,
	}
}

type processor struct {
	cfg       config.ProcessorConfig
	evmConfig config.EvmConfig
	messenger Messenger
	userID    id.UserID
	logger    *zap.SugaredLogger
	tracer    trace.Tracer
	timeout   time.Duration // timeout after which a request is considered failed

	mu                    sync.Mutex
	responseChannels      map[string]chan *messages.Message
	serviceRegistry       ServiceRegistry
	responseHandler       ResponseHandler
	identificationHandler IdentificationHandler
	chequeHandler         ChequeHandler
	myBotAddress          common.Address
	compressor            compression.Compressor[*messages.Message, [][]byte]
}

func (p *processor) SetUserID(userID id.UserID) {
	p.userID = userID
	p.myBotAddress = addressFromUserID(userID)
}

func (*processor) Checkpoint() string {
	return "processor"
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

func (p *processor) ProcessInbound(msg *messages.Message) error {
	if p.userID == "" {
		return ErrUserIDNotSet
	}
	if msg.Sender != p.userID { // outbound messages = messages sent by own ext system
		switch msg.Type.Category() {
		case messages.Request:
			return p.Respond(msg)
		case messages.Response:
			p.Forward(msg)
			return nil
		default:
			return ErrUnknownMessageCategory
		}
	} else {
		return nil // ignore own outbound messages
	}
}

func (p *processor) ProcessOutbound(ctx context.Context, msg *messages.Message) (*messages.Message, error) {
	msg.Sender = p.userID
	if msg.Type.Category() == messages.Request { // only request messages (received by are processed
		return p.Request(ctx, msg) // forward request msg to matrix
	}
	p.logger.Debugf("Ignoring any non-request message from sender other than: %s ", p.userID)
	return nil, ErrOnlyRequestMessagesAllowed // ignore msg
}

func (p *processor) Request(ctx context.Context, msg *messages.Message) (*messages.Message, error) {
	p.logger.Debug("Sending outbound request message")
	responseChan := make(chan *messages.Message)
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
		return nil, ErrMissingRecipient
	}

	p.logger.Infof("Distributor: received a request to propagate to CMAccount %s", msg.Metadata.Recipient)
	// lookup for CM Account -> bot
	recipientCMAccAddr := common.HexToAddress(msg.Metadata.Recipient)
	recipientBotUserID, err := p.identificationHandler.getFirstBotUserIDFromCMAccountAddress(recipientCMAccAddr)
	if err != nil {
		return nil, err
	}

	msg.Metadata.Cheques = []cheques.SignedCheque{}

	isBotAllowed, err := p.chequeHandler.IsBotAllowed(ctx, p.myBotAddress)
	if err != nil {
		return nil, err
	}
	if !isBotAllowed {
		return nil, ErrBotMissingChequeOperatorRole
	}

	// Compress and chunk message

	ctx, compressedContent, err := p.compress(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Cheque Issuing start
	numberOfChunks := big.NewInt(int64(len(compressedContent)))
	totalNetworkFee := new(big.Int).Mul(networkFee, numberOfChunks)

	networkFeeCheque, err := p.chequeHandler.IssueCheque(
		ctx,
		p.identificationHandler.getMyCMAccountAddress(),
		common.HexToAddress(p.evmConfig.NetworkFeeRecipientCMAccountAddress),
		common.HexToAddress(p.evmConfig.NetworkFeeRecipientBotAddress),
		totalNetworkFee,
	)
	if err != nil {
		p.logger.Errorf("failed to issue network fee cheque: %v", err)
		return nil, fmt.Errorf("failed to issue network fee cheque: %w", err)
	}

	serviceFee, err := p.chequeHandler.GetServiceFee(ctx, recipientCMAccAddr, msg.Type)
	if err != nil {
		return nil, err
	}
	serviceFeeCheque, err := p.chequeHandler.IssueCheque(
		ctx,
		p.identificationHandler.getMyCMAccountAddress(),
		recipientCMAccAddr,
		addressFromUserID(recipientBotUserID),
		serviceFee,
	)
	if err != nil {
		p.logger.Errorf("failed to issue service fee cheque: %v", err)
		return nil, fmt.Errorf("failed to issue service fee cheque: %w", err)
	}

	msg.Metadata.Cheques = append(msg.Metadata.Cheques, *networkFeeCheque, *serviceFeeCheque)
	// Cheque Issuing end

	ctx, span := p.tracer.Start(ctx, "processor.Request", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer span.End()

	if err := p.responseHandler.HandleRequest(ctx, msg.Type, msg.Content); err != nil {
		return nil, err
	}
	p.logger.Infof("Distributor: Bot %s is contacting bot %s of the CMaccount %s", msg.Sender, recipientBotUserID, msg.Metadata.Recipient)

	if err := p.messenger.SendAsync(ctx, *msg, compressedContent, recipientBotUserID); err != nil {
		return nil, err
	}
	ctx, responseSpan := p.tracer.Start(ctx, "processor.AwaitResponse", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer responseSpan.End()
	for {
		select {
		case response := <-responseChan:
			if response.Metadata.RequestID == msg.Metadata.RequestID {
				p.responseHandler.HandleResponse(ctx, msg.Type, msg.Content, response.Content)
				return response, nil
			}
		case <-ctx.Done():
			return nil, fmt.Errorf("%w of %v seconds for request: %s", ErrExceededResponseTimeout, p.timeout, msg.Metadata.RequestID)
		}
	}
}

func (p *processor) Respond(msg *messages.Message) error {
	traceID, err := trace.TraceIDFromHex(msg.Metadata.RequestID)
	if err != nil {
		p.logger.Warnf("failed to parse traceID from hex [requestID:%s]: %v", msg.Metadata.RequestID, err)
	}

	ctx := trace.ContextWithRemoteSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID}))
	ctx, responseSpan := p.tracer.Start(ctx, "processor-response", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer responseSpan.End()

	service, supported := p.serviceRegistry.GetClient(msg.Type)
	if !supported {
		return fmt.Errorf("%w: %s", ErrUnsupportedService, msg.Type)
	}

	cheque := p.getChequeForThisBot(msg.Metadata.Cheques)
	if cheque == nil {
		return ErrMissingCheques
	}

	serviceFee, err := p.chequeHandler.GetServiceFee(ctx, common.HexToAddress(msg.Metadata.Recipient), msg.Type)
	if err != nil {
		return err
	}

	if err := p.chequeHandler.VerifyCheque(ctx, cheque, addressFromUserID(msg.Sender), serviceFee); err != nil {
		return err
	}

	ctx, responseMsg, compressedContent := p.callPartnerPluginAndGetResponse(ctx, msg, cheque, service)

	return p.messenger.SendAsync(ctx, *responseMsg, compressedContent, msg.Sender)
}

func (p *processor) callPartnerPluginAndGetResponse(
	ctx context.Context,
	requestMsg *messages.Message,
	cheque *cheques.SignedCheque,
	service clients.Client,
) (context.Context, *messages.Message, [][]byte) {
	requestMsg.Metadata.Stamp(fmt.Sprintf("%s-%s", p.Checkpoint(), "request"))
	requestMsg.Metadata.Sender = cheque.FromCMAccount.Hex()

	responseMsg := &messages.Message{
		Metadata: requestMsg.Metadata,
	}

	ctx = grpc_metadata.NewOutgoingContext(ctx, requestMsg.Metadata.ToGrpcMD())
	header := &grpc_metadata.MD{}
	ctx, partnerPluginSpan := p.tracer.Start(ctx, "service.Call", trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attribute.String("type", string(requestMsg.Type))))
	response, msgType, err := service.Call(ctx, requestMsg.Content, grpc.Header(header))
	partnerPluginSpan.End()

	responseMsg.Type = msgType
	if response != nil {
		responseMsg.Content = response
	}

	if err != nil {
		errMessage := fmt.Sprintf("error calling partner plugin service: %v", err)
		p.logger.Errorf(errMessage)
		p.responseHandler.AddErrorToResponseHeader(msgType, responseMsg.Content, errMessage)
		return ctx, responseMsg, [][]byte{{}}
	}

	if err := responseMsg.Metadata.FromGrpcMD(*header); err != nil {
		p.logger.Infof("error extracting metadata for request: %s", responseMsg.Metadata.RequestID)
	}

	p.logger.Infof("Supplier: CMAccount %s is calling plugin of the CMAccount %s", responseMsg.Metadata.Sender, responseMsg.Metadata.Recipient)
	p.responseHandler.HandleResponse(ctx, msgType, requestMsg.Content, response)

	p.logger.Infof("Supplier: Bot %s responding to BOT %s", p.userID, requestMsg.Sender)

	ctx, compressedContent, err := p.compress(ctx, requestMsg)
	if err != nil {
		errMessage := fmt.Sprintf("error compressing/chunking response: %v", err)
		p.logger.Errorf(errMessage)
		p.responseHandler.AddErrorToResponseHeader(msgType, responseMsg.Content, errMessage)
	}

	return ctx, responseMsg, compressedContent
}

func (p *processor) Forward(msg *messages.Message) {
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

func (p *processor) getChequeForThisBot(cheques []cheques.SignedCheque) *cheques.SignedCheque {
	for _, cheque := range cheques {
		if cheque.ToBot == p.myBotAddress && cheque.ToCMAccount == p.identificationHandler.getMyCMAccountAddress() {
			return &cheque
		}
	}
	return nil
}

func (p *processor) compress(ctx context.Context, msg *messages.Message) (context.Context, [][]byte, error) {
	ctx, compressSpan := p.tracer.Start(ctx, "messenger.Compress", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer compressSpan.End()
	compressedContent, err := p.compressor.Compress(msg)
	if err != nil {
		return ctx, [][]byte{{}}, err
	}
	return ctx, compressedContent, nil
}

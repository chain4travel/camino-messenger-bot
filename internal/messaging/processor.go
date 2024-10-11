package messaging

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheque_handler"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	cmaccountscache "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts_cache"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	grpc_metadata "google.golang.org/grpc/metadata"
	"maunium.net/go/mautrix/id"
)

var (
	_ MessageProcessor = (*messageProcessor)(nil)

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

	networkFee         = big.NewInt(300000000000000) // 0.00003 CAM
	chequeOperatorRole = crypto.Keccak256Hash([]byte("CHEQUE_OPERATOR_ROLE"))
)

type MessageProcessor interface {
	metadata.Checkpoint

	Start(ctx context.Context)
	ProcessIncomingMessage(message *types.Message) error
	SendRequestMessage(ctx context.Context, message *types.Message) (*types.Message, error)
}

func NewMessageProcessor(
	messenger Messenger,
	logger *zap.SugaredLogger,
	responseTimeout time.Duration,
	botUserID id.UserID,
	cmAccountAddress common.Address,
	networkFeeRecipientBotAddress common.Address,
	networkFeeRecipientCMAccountAddress common.Address,
	registry ServiceRegistry,
	responseHandler ResponseHandler,
	chequeHandler cheque_handler.ChequeHandler,
	compressor compression.Compressor[*types.Message, [][]byte],
	cmAccounts cmaccountscache.CMAccountsCache,
	matrixHost string,
) MessageProcessor {
	return &messageProcessor{
		messenger:                           messenger,
		logger:                              logger,
		tracer:                              otel.GetTracerProvider().Tracer(""),
		responseTimeout:                     responseTimeout, // for now applies to all request types
		responseChannels:                    make(map[string]chan *types.Message),
		serviceRegistry:                     registry,
		responseHandler:                     responseHandler,
		chequeHandler:                       chequeHandler,
		compressor:                          compressor,
		cmAccounts:                          cmAccounts,
		myBotAddress:                        addressFromUserID(botUserID),
		botUserID:                           botUserID,
		cmAccountAddress:                    cmAccountAddress,
		networkFeeRecipientBotAddress:       networkFeeRecipientBotAddress,
		networkFeeRecipientCMAccountAddress: networkFeeRecipientCMAccountAddress,
		matrixHost:                          matrixHost,
	}
}

type messageProcessor struct {
	messenger                           Messenger
	logger                              *zap.SugaredLogger
	tracer                              trace.Tracer
	responseTimeout                     time.Duration // timeout after which a request is considered failed
	botUserID                           id.UserID
	myBotAddress                        common.Address
	cmAccountAddress                    common.Address
	networkFeeRecipientBotAddress       common.Address
	networkFeeRecipientCMAccountAddress common.Address

	responseChannelsLock sync.RWMutex
	responseChannels     map[string]chan *types.Message
	serviceRegistry      ServiceRegistry
	responseHandler      ResponseHandler
	chequeHandler        cheque_handler.ChequeHandler
	compressor           compression.Compressor[*types.Message, [][]byte]
	cmAccounts           cmaccountscache.CMAccountsCache
	matrixHost           string
}

func (*messageProcessor) Checkpoint() string {
	return "processor"
}

func (p *messageProcessor) Start(ctx context.Context) {
	for {
		select {
		case msgEvent := <-p.messenger.Inbound():
			p.logger.Debug("Processing msg event of type: ", msgEvent.Type)
			go func() {
				if err := p.ProcessIncomingMessage(&msgEvent); err != nil {
					p.logger.Warnf("could not process message: %v", err)
				}
			}()
		case <-ctx.Done():
			p.logger.Info("Stopping processor...")
			return
		}
	}
}

func (p *messageProcessor) ProcessIncomingMessage(msg *types.Message) error {
	switch msg.Type.Category() {
	case types.Request:
		return p.respond(msg)
	case types.Response:
		p.forward(msg)
		return nil
	default:
		return ErrUnknownMessageCategory
	}
}

func (p *messageProcessor) SendRequestMessage(ctx context.Context, requestMsg *types.Message) (*types.Message, error) {
	if requestMsg.Type.Category() != types.Request {
		return nil, ErrOnlyRequestMessagesAllowed
	}

	requestMsg.Sender = p.botUserID

	p.logger.Debug("Sending outbound request message")
	responseChan := make(chan *types.Message)
	p.setResponseChannel(requestMsg.Metadata.RequestID, responseChan)
	defer p.deleteResponseChannel(requestMsg.Metadata.RequestID)

	ctx, cancel := context.WithTimeout(ctx, p.responseTimeout)
	defer cancel()

	if requestMsg.Metadata.Recipient == "" { // TODO: add address validation
		return nil, ErrMissingRecipient
	}

	p.logger.Infof("Distributor: received a request to propagate to CMAccount %s", requestMsg.Metadata.Recipient)
	// lookup for CM Account -> bot
	recipientCMAccAddr := common.HexToAddress(requestMsg.Metadata.Recipient)
	recipientBotAddr, err := p.getFirstBotFromCMAccount(recipientCMAccAddr)
	if err != nil {
		return nil, err
	}

	// TODO@ do we want to check this every time? or just once at startup?
	// TODO@ we can also listen chain for bot permission changes and shut down bot if it loses
	// TODO@ or not shutdown, but set some bool that will block all incoming requests with noop
	isBotAllowed, err := p.chequeHandler.IsAllowedToIssueCheque(ctx, p.myBotAddress)
	if err != nil {
		return nil, err
	}
	if !isBotAllowed {
		return nil, ErrBotMissingChequeOperatorRole
	}

	serviceFee, err := p.getServiceFee(ctx, recipientCMAccAddr, requestMsg.Type.ToServiceName())
	if err != nil {
		// TODO @evlekht explicitly say if service is not supported and its not just some network error
		return nil, err
	}

	if err := p.responseHandler.PrepareRequest(requestMsg.Type, requestMsg.Content); err != nil {
		return nil, err
	}

	ctx, err = p.compressMessage(ctx, requestMsg)
	if err != nil {
		return nil, err
	}

	if err := p.issueCheques(ctx, requestMsg, serviceFee, recipientCMAccAddr, recipientBotAddr); err != nil {
		return nil, err
	}

	ctx, span := p.tracer.Start(ctx, "processor.Request", trace.WithAttributes(attribute.String("type", string(requestMsg.Type))))
	defer span.End()

	p.logger.Infof("Distributor: Bot %s is contacting bot %s of the CMaccount %s", requestMsg.Sender, recipientBotAddr, requestMsg.Metadata.Recipient)

	if err := p.messenger.SendAsync(ctx, requestMsg, UserIDFromAddress(recipientBotAddr, p.matrixHost)); err != nil {
		return nil, err
	}

	ctx, responseSpan := p.tracer.Start(ctx, "processor.AwaitResponse", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attribute.String("type", string(requestMsg.Type))))
	defer responseSpan.End()

	select {
	case responseMsg := <-responseChan:
		if responseMsg.Metadata.RequestID == requestMsg.Metadata.RequestID {
			// TODO@ do we still care about context timeout here? if not, context must be freed of timeout
			// TODO@ currently, timeout is described as its only for receiving response from matrix
			// TODO@ but maybe it will make more sense to use timeout for whole bot request-response cycle?
			// TODO@ like, its timeout meaningful for external requester
			p.responseHandler.ProcessResponseMessage(ctx, requestMsg, responseMsg)
			return responseMsg, nil
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("%w of %v seconds for request: %s", ErrExceededResponseTimeout, p.responseTimeout, requestMsg.Metadata.RequestID)
	}

	panic("unreachable") // will never get there, but compiler doesn't know that, so we need to satisfy return
}

func (p *messageProcessor) respond(msg *types.Message) error {
	traceID, err := trace.TraceIDFromHex(msg.Metadata.RequestID)
	if err != nil {
		p.logger.Warnf("failed to parse traceID from hex [requestID:%s]: %v", msg.Metadata.RequestID, err)
	}

	ctx := trace.ContextWithRemoteSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID}))
	ctx, responseSpan := p.tracer.Start(ctx, "processor-response", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer responseSpan.End()

	service, supported := p.serviceRegistry.GetService(msg.Type)
	if !supported {
		return fmt.Errorf("%w: %s", ErrUnsupportedService, msg.Type)
	}

	cheque, err := p.getChequeForThisBot(msg.Metadata.Cheques)
	if err != nil {
		return err
	}

	serviceFee, err := p.getServiceFee(ctx, p.cmAccountAddress, service.Name())
	if err != nil {
		return err
	}

	if err := p.chequeHandler.VerifyCheque(ctx, cheque, addressFromUserID(msg.Sender), serviceFee); err != nil {
		return err
	}

	ctx, responseMsg := p.callPartnerPluginAndGetResponse(ctx, msg, cheque, service)

	ctx, err = p.compressMessage(ctx, responseMsg)
	if err != nil {
		errMessage := fmt.Sprintf("error compressing/chunking response: %v", err)
		p.logger.Errorf(errMessage)
		p.responseHandler.AddErrorToResponseHeader(responseMsg.Content, errMessage)
	}

	return p.messenger.SendAsync(ctx, responseMsg, msg.Sender)
}

func (p *messageProcessor) callPartnerPluginAndGetResponse(
	ctx context.Context,
	requestMsg *types.Message,
	cheque *cheques.SignedCheque,
	service rpc.Client,
) (context.Context, *types.Message) {
	requestMsg.Metadata.Stamp(fmt.Sprintf("%s-%s", p.Checkpoint(), "request"))
	requestMsg.Metadata.Sender = cheque.FromCMAccount.Hex()

	responseMsg := &types.Message{
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
		p.responseHandler.AddErrorToResponseHeader(responseMsg.Content, errMessage)
		return ctx, responseMsg
	}

	if err := responseMsg.Metadata.FromGrpcMD(*header); err != nil {
		p.logger.Infof("error extracting metadata for request: %s", responseMsg.Metadata.RequestID)
	}

	p.logger.Infof("Supplier: CMAccount %s is calling plugin of the CMAccount %s", responseMsg.Metadata.Sender, responseMsg.Metadata.Recipient)
	p.responseHandler.PrepareResponseMessage(ctx, requestMsg, responseMsg)

	p.logger.Infof("Supplier: Bot %s responding to BOT %s", p.botUserID, requestMsg.Sender)

	return ctx, responseMsg
}

func (p *messageProcessor) forward(msg *types.Message) {
	p.logger.Debugf("Forwarding outbound response message: %s", msg.Metadata.RequestID)
	responseChan, ok := p.getResponseChannel(msg.Metadata.RequestID)
	if ok {
		responseChan <- msg
		close(responseChan)
		return
	}
	p.logger.Warnf("Failed to forward message: no response channel for request (%s)", msg.Metadata.RequestID)
}

func (p *messageProcessor) getChequeForThisBot(cheques []cheques.SignedCheque) (*cheques.SignedCheque, error) {
	for _, cheque := range cheques {
		if cheque.ToBot == p.myBotAddress && cheque.ToCMAccount == p.cmAccountAddress {
			return &cheque, nil
		}
	}
	return nil, ErrMissingCheques
}

func (p *messageProcessor) compressMessage(ctx context.Context, msg *types.Message) (context.Context, error) {
	ctx, compressSpan := p.tracer.Start(ctx, "messenger.Compress", trace.WithAttributes(attribute.String("type", string(msg.Type))))
	defer compressSpan.End()
	compressedContent, err := p.compressor.Compress(msg)
	if err != nil {
		return ctx, err
	}
	msg.CompressedContent = compressedContent
	return ctx, nil
}

func (p *messageProcessor) getServiceFee(
	ctx context.Context,
	supplierCmAccountAddress common.Address,
	serviceFullName string,
) (*big.Int, error) {
	supplierCmAccount, err := p.cmAccounts.Get(supplierCmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get supplier cmAccount: %w", err)
	}

	serviceFee, err := supplierCmAccount.GetServiceFee(
		&bind.CallOpts{Context: ctx},
		serviceFullName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get service fee: %w", err)
	}
	return serviceFee, nil
}

func (p *messageProcessor) issueCheques(
	ctx context.Context,
	msg *types.Message,
	serviceFee *big.Int,
	recipientCMAccAddr common.Address,
	recipientBotAddr common.Address,
) error {
	numberOfChunks := big.NewInt(int64(len(msg.CompressedContent)))
	totalNetworkFee := new(big.Int).Mul(networkFee, numberOfChunks)

	networkFeeCheque, err := p.chequeHandler.IssueCheque(
		ctx,
		p.cmAccountAddress,
		p.networkFeeRecipientCMAccountAddress,
		p.networkFeeRecipientBotAddress,
		totalNetworkFee,
	)
	if err != nil {
		err = fmt.Errorf("failed to issue network fee cheque: %w", err)
		p.logger.Error(err)
		return err
	}

	serviceFeeCheque, err := p.chequeHandler.IssueCheque(
		ctx,
		p.cmAccountAddress,
		recipientCMAccAddr,
		recipientBotAddr,
		serviceFee,
	)
	if err != nil {
		err = fmt.Errorf("failed to issue service fee cheque: %w", err)
		p.logger.Error(err)
		return err
	}

	msg.Metadata.Cheques = append(msg.Metadata.Cheques, *networkFeeCheque, *serviceFeeCheque)
	return nil
}

func (p *messageProcessor) getResponseChannel(requestID string) (chan *types.Message, bool) {
	p.responseChannelsLock.RLock()
	defer p.responseChannelsLock.RUnlock()
	ch, ok := p.responseChannels[requestID]
	return ch, ok
}

func (p *messageProcessor) setResponseChannel(requestID string, ch chan *types.Message) {
	p.responseChannelsLock.Lock()
	defer p.responseChannelsLock.Unlock()
	p.responseChannels[requestID] = ch
}

func (p *messageProcessor) deleteResponseChannel(requestID string) {
	p.responseChannelsLock.Lock()
	defer p.responseChannelsLock.Unlock()
	delete(p.responseChannels, requestID)
}

func (p *messageProcessor) getFirstBotFromCMAccount(cmAccountAddress common.Address) (common.Address, error) {
	bots, err := p.getAllBotAddressesFromCMAccount(cmAccountAddress)
	if err != nil {
		return common.Address{}, err
	}
	return bots[0], nil
}

func (p *messageProcessor) getAllBotAddressesFromCMAccount(cmAccountAddress common.Address) ([]common.Address, error) {
	cmAccount, err := p.cmAccounts.Get(cmAccountAddress)
	if err != nil {
		p.logger.Errorf("Failed to get cm Account: %v", err)
		return nil, err
	}

	countBig, err := cmAccount.GetRoleMemberCount(
		&bind.CallOpts{Context: context.TODO()},
		chequeOperatorRole,
	)
	if err != nil {
		p.logger.Errorf("Failed to call contract function: %v", err)
		return nil, err
	}

	count := countBig.Int64()
	botsAddresses := make([]common.Address, 0, count)
	for i := int64(0); i < count; i++ {
		address, err := cmAccount.GetRoleMember(
			&bind.CallOpts{Context: context.TODO()},
			chequeOperatorRole,
			big.NewInt(i),
		)
		if err != nil {
			p.logger.Errorf("Failed to call contract function: %v", err)
			continue
		}
		botsAddresses = append(botsAddresses, address)
	}

	return botsAddresses, nil
}

func UserIDFromAddress(address common.Address, host string) id.UserID {
	return id.NewUserID(strings.ToLower(address.Hex()), host)
}

func addressFromUserID(userID id.UserID) common.Address {
	return common.HexToAddress(userID.Localpart())
}

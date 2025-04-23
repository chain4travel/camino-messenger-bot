// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

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
	"github.com/chain4travel/camino-messenger-bot/internal/partnerplugin"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/pkg/chequehandler"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts"
	"github.com/ethereum/go-ethereum/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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

	networkFee = big.NewInt(300000000000000) // 0.00003 CAM
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
	partnerPlugin partnerplugin.PartnerPlugin,
	chequeHandler chequehandler.ChequeHandler,
	compressor compression.Compressor[*types.Message, [][]byte],
	cmAccounts cmaccounts.Service,
) MessageProcessor {
	return &messageProcessor{
		messenger:                           messenger,
		logger:                              logger,
		tracer:                              otel.GetTracerProvider().Tracer(""),
		responseTimeout:                     responseTimeout, // for now applies to all request types
		responseChannels:                    make(map[string]chan *types.Message),
		serviceRegistry:                     registry,
		responseHandler:                     responseHandler,
		partnerPlugin:                       partnerPlugin,
		chequeHandler:                       chequeHandler,
		compressor:                          compressor,
		cmAccounts:                          cmAccounts,
		matrixHost:                          botUserID.Homeserver(),
		myBotAddress:                        addressFromUserID(botUserID),
		botUserID:                           botUserID,
		cmAccountAddress:                    cmAccountAddress,
		networkFeeRecipientBotAddress:       networkFeeRecipientBotAddress,
		networkFeeRecipientCMAccountAddress: networkFeeRecipientCMAccountAddress,
	}
}

type messageProcessor struct {
	responseTimeout                     time.Duration // timeout after which a request is considered failed
	matrixHost                          string
	botUserID                           id.UserID
	myBotAddress                        common.Address
	cmAccountAddress                    common.Address
	networkFeeRecipientBotAddress       common.Address
	networkFeeRecipientCMAccountAddress common.Address

	messenger            Messenger
	logger               *zap.SugaredLogger
	tracer               trace.Tracer
	responseChannelsLock sync.RWMutex
	responseChannels     map[string]chan *types.Message
	serviceRegistry      ServiceRegistry
	responseHandler      ResponseHandler
	partnerPlugin        partnerplugin.PartnerPlugin
	chequeHandler        chequehandler.ChequeHandler
	compressor           compression.Compressor[*types.Message, [][]byte]
	cmAccounts           cmaccounts.Service
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

	requestMsg.SenderBotUserID = p.botUserID

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
	recipientBotAddr, err := p.cmAccounts.GetFirstChequeOperator(ctx, recipientCMAccAddr)
	if err != nil {
		return nil, err
	}

	requestMsg.Metadata.Cheques = []cheques.SignedCheque{}

	isBotAllowed, err := p.cmAccounts.IsBotAllowed(ctx, p.cmAccountAddress, p.myBotAddress)
	if err != nil {
		return nil, err
	}
	if !isBotAllowed {
		return nil, ErrBotMissingChequeOperatorRole
	}

	serviceFee, err := p.cmAccounts.GetServiceFee(ctx, recipientCMAccAddr, requestMsg.Type.ToServiceName())
	if err != nil {
		// TODO @evlekht explicitly say if service is not supported and its not just some network error
		return nil, err
	}

	if err := p.responseHandler.PrepareRequest(requestMsg.Content); err != nil {
		return nil, err
	}

	ctx, err = p.compressMessage(ctx, requestMsg)
	if err != nil {
		return nil, err
	}

	if err := p.issueNetworkCheque(ctx, requestMsg); err != nil {
		return nil, err
	}

	if err := p.issueServiceCheque(ctx, requestMsg, serviceFee, recipientCMAccAddr, recipientBotAddr); err != nil {
		return nil, err
	}

	ctx, span := p.tracer.Start(ctx, "processor.Request", trace.WithAttributes(attribute.String("type", string(requestMsg.Type))))
	defer span.End()

	p.logger.Infof("Distributor: Bot %s is contacting bot %s of the CMaccount %s", requestMsg.SenderBotUserID, recipientBotAddr, requestMsg.Metadata.Recipient)

	if err := p.messenger.SendAsync(
		ctx,
		requestMsg,
		UserIDFromAddress(recipientBotAddr, p.matrixHost),
	); err != nil {
		return nil, err
	}

	ctx, responseSpan := p.tracer.Start(ctx, "processor.AwaitResponse", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attribute.String("type", string(requestMsg.Type))))
	defer responseSpan.End()

	select {
	case responseMsg := <-responseChan:
		if responseMsg.Metadata.RequestID == requestMsg.Metadata.RequestID {
			p.responseHandler.ProcessResponseMessage(ctx, responseMsg)
			return responseMsg, nil
		} else {
			err := fmt.Errorf("unexpected response (%s) for request (%s)", responseMsg.Metadata.RequestID, requestMsg.Metadata.RequestID)
			p.logger.Error(err)
			return nil, err
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("%w of %v seconds for request: %s", ErrExceededResponseTimeout, p.responseTimeout, requestMsg.Metadata.RequestID)
	}
}

func (p *messageProcessor) respond(requestMsg *types.Message) error {
	traceID, err := trace.TraceIDFromHex(requestMsg.Metadata.RequestID)
	if err != nil {
		p.logger.Warnf("failed to parse traceID from hex [requestID:%s]: %v", requestMsg.Metadata.RequestID, err)
	}

	ctx := trace.ContextWithRemoteSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID}))
	ctx, responseSpan := p.tracer.Start(ctx, "processor-response", trace.WithAttributes(attribute.String("type", string(requestMsg.Type))))
	defer responseSpan.End()

	service, supported := p.serviceRegistry.GetService(requestMsg.Type)
	if !supported {
		return fmt.Errorf("%w: %s", ErrUnsupportedService, requestMsg.Type)
	}

	cheque, err := p.getChequeForThisBot(requestMsg.Metadata.Cheques)
	if err != nil {
		return err
	}

	serviceFee, err := p.cmAccounts.GetServiceFee(ctx, common.HexToAddress(requestMsg.Metadata.Recipient), service.Name())
	if err != nil {
		return err
	}

	if err := p.chequeHandler.VerifyCheque(ctx, cheque, addressFromUserID(requestMsg.SenderBotUserID), serviceFee); err != nil {
		return err
	}
	requestMsg.Metadata.Sender = cheque.FromCMAccount.Hex()

	p.logger.Infof("CMAccount %s is calling partner-plugin of the CMAccount %s", requestMsg.Metadata.Sender, requestMsg.Metadata.Recipient)

	ctx, responseMsg := p.callPartnerPluginAndGetResponse(ctx, requestMsg, service)

	ctx, err = p.compressMessage(ctx, responseMsg)
	if err != nil {
		errMessage := fmt.Sprintf("error compressing/chunking response: %v", err)
		p.logger.Errorf(errMessage)
		p.responseHandler.AddErrorToResponseHeader(responseMsg.Content, errMessage)
	}

	if err := p.issueNetworkCheque(ctx, responseMsg); err != nil {
		return err
	}

	return p.messenger.SendAsync(ctx, responseMsg, requestMsg.SenderBotUserID)
}

func (p *messageProcessor) callPartnerPluginAndGetResponse(
	ctx context.Context,
	requestMsg *types.Message,
	service rpc.Client,
) (context.Context, *types.Message) {
	requestMsg.Metadata.Stamp(fmt.Sprintf("%s-%s", p.Checkpoint(), "request"))

	ctx, responseMsg, err := p.partnerPlugin.DoServiceRequest(ctx, requestMsg, service)
	if err != nil {
		errMessage := fmt.Sprintf("error calling partner plugin service: %v", err)
		p.logger.Errorf(errMessage)
		p.responseHandler.AddErrorToResponseHeader(responseMsg.Content, errMessage)
		return ctx, responseMsg
	}

	p.responseHandler.PrepareResponseMessage(ctx, requestMsg, responseMsg)

	p.logger.Infof("Supplier: Bot %s responding to BOT %s", p.botUserID, requestMsg.SenderBotUserID)

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

func (p *messageProcessor) issueNetworkCheque(ctx context.Context, msg *types.Message) error {
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

	msg.Metadata.Cheques = append(msg.Metadata.Cheques, *networkFeeCheque)
	return nil
}

func (p *messageProcessor) issueServiceCheque(
	ctx context.Context,
	msg *types.Message,
	serviceFee *big.Int,
	recipientCMAccAddr common.Address,
	recipientBotAddr common.Address,
) error {
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

	msg.Metadata.Cheques = append(msg.Metadata.Cheques, *serviceFeeCheque)
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

func UserIDFromAddress(address common.Address, host string) id.UserID {
	return id.NewUserID(strings.ToLower(address.Hex()), host)
}

func addressFromUserID(userID id.UserID) common.Address {
	return common.HexToAddress(userID.Localpart())
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"buf.build/go/protovalidate"
	"github.com/chain4travel/camino-matrix-app-service/config"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/common"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/messaging/encryption"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/partnerplugin"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/chequehandler"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/cheques"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v12/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/metadata"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

var (
	_ MessageProcessor = (*messageProcessor)(nil)

	ErrUnknownMessageCategory       = errors.New("unknown message category")
	ErrUnsupportedService           = errors.New("unsupported service")
	ErrExceededResponseTimeout      = errors.New("response exceeded configured timeout")
	ErrBotMissingChequeOperatorRole = errors.New("bot missing permission")
)

type MessageProcessor interface {
	Start(ctx context.Context)

	SendRequestMessage(
		ctx context.Context,
		message *types.Message,
		recipientCMAccount ethCommon.Address,
	) (*types.Message, error)
}

type EncoderDecoder interface {
	EncodeMessage(
		ctx context.Context,
		msg *types.Message,
		serviceFeeCheque *cheques.SignedCheque,
		toBot ethCommon.Address,
		sharedKey encryption.Key,
	) (*EncodedSignedMessage, error)

	DecodeAndVerifyMessage(
		ctx context.Context,
		encodedMessage *EncodedSignedMessage,
		senderBotAddress ethCommon.Address,
	) (
		msg *types.Message,
		serviceFeeCheque *cheques.SignedCheque,
		sharedKey encryption.Key,
		err error,
	)
}

func NewMessageProcessor(
	messenger Messenger,
	logger *zap.SugaredLogger,
	responseTimeout time.Duration,
	botAddress ethCommon.Address,
	cmAccountAddress ethCommon.Address,
	networkFeeRecipientBotAddress ethCommon.Address,
	networkFeeRecipientCMAccountAddress ethCommon.Address,
	registry ServiceRegistry,
	responseHandler ResponseHandler,
	partnerPlugin partnerplugin.PartnerPlugin,
	chequeHandler chequehandler.ChequeHandler,
	cmAccounts cmaccounts.Service,
	responseHeaderHandler common.ResponseHeaderHandler,
	maxAllowedServiceFee *big.Int,
	messageEncoder EncoderDecoder,
) MessageProcessor {
	return &messageProcessor{
		messenger:                           messenger,
		logger:                              logger,
		responseTimeout:                     responseTimeout, // for now applies to all request types
		responseChannels:                    make(map[string]chan *types.Message),
		serviceRegistry:                     registry,
		responseHandler:                     responseHandler,
		partnerPlugin:                       partnerPlugin,
		chequeHandler:                       chequeHandler,
		cmAccounts:                          cmAccounts,
		botAddress:                          botAddress,
		cmAccountAddress:                    cmAccountAddress,
		networkFeeRecipientBotAddress:       networkFeeRecipientBotAddress,
		networkFeeRecipientCMAccountAddress: networkFeeRecipientCMAccountAddress,
		responseHeaderHandler:               responseHeaderHandler,
		maxAllowedServiceFee:                maxAllowedServiceFee,
		encoderDecoder:                      messageEncoder,
	}
}

type messageProcessor struct {
	responseTimeout                     time.Duration // timeout after which a request is considered failed
	botAddress                          ethCommon.Address
	cmAccountAddress                    ethCommon.Address
	networkFeeRecipientBotAddress       ethCommon.Address
	networkFeeRecipientCMAccountAddress ethCommon.Address
	maxAllowedServiceFee                *big.Int

	messenger             Messenger
	logger                *zap.SugaredLogger
	responseChannelsLock  sync.RWMutex
	responseChannels      map[string]chan *types.Message
	serviceRegistry       ServiceRegistry
	responseHandler       ResponseHandler
	partnerPlugin         partnerplugin.PartnerPlugin
	chequeHandler         chequehandler.ChequeHandler
	cmAccounts            cmaccounts.Service
	responseHeaderHandler common.ResponseHeaderHandler
	encoderDecoder        EncoderDecoder
}

func (p *messageProcessor) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case encodedMessage := <-p.messenger.ReceivedMessageChan():
				go func() {
					defer func() {
						if r := recover(); r != nil {
							p.logger.Errorf("Recovered from panic while processing message: %v", r)
						}
					}()
					p.logger.Debugf("Decoding message received from %s", encodedMessage.SenderBotAddress)

					if encodedMessage.SenderBotAddress == p.botAddress {
						// should never happen, if messenger server and p.messenger are configured and working correctly
						p.logger.Warnf("Received message from own bot %s, ignoring", p.botAddress.Hex())
						return
					}

					if encodedMessage.SenderCMAccountAddress == p.cmAccountAddress {
						// should never happen, if messenger server and p.messenger are configured and working correctly
						p.logger.Warnf("Received message from own CM Account %s, ignoring", p.cmAccountAddress.Hex())
						return
					}

					msg, serviceFeeCheque, sharedKey, err := p.encoderDecoder.DecodeAndVerifyMessage(ctx, &encodedMessage.Message, encodedMessage.SenderBotAddress)
					if err != nil {
						p.logger.Errorf("Failed to decode and verify message: %v", err)
						return
					}
					p.logger.Debugf("Decoded message (%s, %s), processing", msg.Type, msg.RequestID)

					if err := p.processIncomingMessage(msg, serviceFeeCheque, encodedMessage.SenderBotAddress, encodedMessage.SenderCMAccountAddress, sharedKey); err != nil {
						p.logger.Warnf("Could not process message: %v", err)
						return
					}
				}()
			case <-ctx.Done():
				// for consistency with how other components log their shutdown
				p.logger.Info("Stopping processor...")
				p.logger.Info("Processor stopped")
				return
			}
		}
	}()
}

func (p *messageProcessor) processIncomingMessage(
	msg *types.Message,
	serviceFeeCheque *cheques.SignedCheque,
	senderBotAddress ethCommon.Address,
	senderCMAccountAddress ethCommon.Address,
	sharedKey encryption.Key,
) error {
	if serviceFeeCheque != nil && serviceFeeCheque.FromCMAccount != senderCMAccountAddress {
		return fmt.Errorf("service fee cheque from %s does not match sender CM Account %s", serviceFeeCheque.FromCMAccount.Hex(), senderCMAccountAddress.Hex())
	}

	msgCategory := msg.Type.Category()
	if msgCategory == types.Request && serviceFeeCheque == nil {
		return fmt.Errorf("request message %s without service fee cheque", msg.Type)
	}

	switch msgCategory {
	case types.Request:
		msg.Timestamps.Stamp(metadata.CheckpointP2PRequestMessageReceivedFromServer)
		return p.respond(context.Background(), msg, serviceFeeCheque, senderBotAddress, sharedKey)
	case types.Response:
		msg.Timestamps.Stamp(metadata.CheckpointP2PResponseMessageReceivedFromServer)
		return p.forwardToHandler(msg)
	default:
		return ErrUnknownMessageCategory
	}
}

func (p *messageProcessor) SendRequestMessage(
	ctx context.Context,
	requestMsg *types.Message,
	recipientCMAccount ethCommon.Address,
) (*types.Message, error) {
	p.logger.Debugf("Sending request message %s (id %s) to CMAccount %s", requestMsg.Type, requestMsg.RequestID, recipientCMAccount.Hex())

	responseChan := make(chan *types.Message)
	p.setResponseChannel(requestMsg.RequestID, responseChan)
	defer p.deleteResponseChannel(requestMsg.RequestID)

	ctx, cancel := context.WithTimeout(ctx, p.responseTimeout)
	defer cancel()

	// lookup for CM Account -> bot
	recipientBotAddr, err := p.cmAccounts.GetFirstChequeOperator(ctx, recipientCMAccount)
	if err != nil {
		return nil, err
	}

	isBotAllowed, err := p.cmAccounts.IsBotAllowed(ctx, p.cmAccountAddress, p.botAddress)
	if err != nil {
		return nil, err
	}
	if !isBotAllowed {
		return nil, ErrBotMissingChequeOperatorRole
	}

	serviceFee, err := p.cmAccounts.GetServiceFee(ctx, recipientCMAccount, requestMsg.Type.ToServiceName())
	if err != nil {
		// TODO @evlekht explicitly say if service is not supported and its not just some network error
		return nil, err
	}

	if serviceFee.Cmp(p.maxAllowedServiceFee) > 0 {
		err = fmt.Errorf("%s service fee %s exceeds maximum allowed service fee %s", requestMsg.Type.ToServiceName(), serviceFee.String(), p.maxAllowedServiceFee.String())
		p.logger.Error(err)
		return nil, err
	}

	p.responseHandler.PrepareRequest(requestMsg.Content)

	serviceFeeCheque, err := p.chequeHandler.IssueCheque(
		ctx,
		recipientCMAccount,
		recipientBotAddr,
		serviceFee,
	)
	if err != nil {
		err = fmt.Errorf("failed to issue service fee cheque: %w", err)
		p.logger.Error(err)
		return nil, err
	}

	sharedKey, err := encryption.NewKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	requestMsg.Timestamps.Stamp(metadata.CheckpointP2PRequestMessageSentToServer)

	encodedRequestMessage, err := p.encoderDecoder.EncodeMessage(ctx, requestMsg, serviceFeeCheque, recipientBotAddr, sharedKey)
	if err != nil {
		return nil, err
	}

	networkFeeCheque, err := p.issueNetworkCheque(ctx, encodedRequestMessage)
	if err != nil {
		return nil, err
	}

	p.logger.Infof("Distributor: Bot %s is contacting bot %s of the CMaccount %s", p.botAddress, recipientBotAddr, recipientCMAccount.Hex())

	if err := p.messenger.SendMessage(
		ctx,
		encodedRequestMessage,
		recipientBotAddr,
		networkFeeCheque,
	); err != nil {
		return nil, err
	}

	select {
	case responseMsg := <-responseChan:
		if responseMsg.RequestID == requestMsg.RequestID {
			if err := protovalidate.Validate(responseMsg.Content); err != nil {
				return nil, fmt.Errorf("response validation failed: %w", err)
			}

			p.responseHandler.ProcessResponseMessage(ctx, requestMsg, responseMsg)
			return responseMsg, nil
		} else {
			err := fmt.Errorf("unexpected response (%s) for request (%s)", responseMsg.RequestID, requestMsg.RequestID)
			p.logger.Error(err)
			return nil, err
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("%w of %v seconds for request: %s", ErrExceededResponseTimeout, p.responseTimeout, requestMsg.RequestID)
	}
}

func (p *messageProcessor) respond(
	ctx context.Context,
	requestMsg *types.Message,
	serviceFeeCheque *cheques.SignedCheque,
	senderBotAddress ethCommon.Address,
	sharedKey encryption.Key,
) error {
	p.logger.Debugf("Responding to request message %s (id %s) from bot %s", requestMsg.Type, requestMsg.RequestID, senderBotAddress.Hex())

	service, supported := p.serviceRegistry.GetService(requestMsg.Type)
	if !supported {
		return fmt.Errorf("%w: %s", ErrUnsupportedService, requestMsg.Type)
	}

	serviceFee, err := p.cmAccounts.GetServiceFee(ctx, p.cmAccountAddress, service.Name())
	if err != nil {
		return err
	}

	if err := p.chequeHandler.VerifyAndStoreCheque(ctx, serviceFeeCheque, senderBotAddress, serviceFee); err != nil {
		return err
	}

	responseMsg := p.validateAndRespond(
		ctx,
		requestMsg,
		service,
		serviceFeeCheque.FromCMAccount,
		serviceFeeCheque.ToCMAccount,
	)

	p.logger.Infof("Supplier: Bot %s responding to BOT %s", p.botAddress, senderBotAddress)

	responseMsg.Timestamps.Stamp(metadata.CheckpointP2PResponseMessageSentToServer)

	encodedResponseMessage, err := p.encoderDecoder.EncodeMessage(ctx, responseMsg, nil, senderBotAddress, sharedKey)
	if err != nil {
		return err
	}

	networkFeeCheque, err := p.issueNetworkCheque(ctx, encodedResponseMessage)
	if err != nil {
		return err
	}

	return p.messenger.SendMessage(ctx, encodedResponseMessage, senderBotAddress, networkFeeCheque)
}

func (p *messageProcessor) validateAndRespond(
	ctx context.Context,
	requestMsg *types.Message,
	serviceClient rpc.Client,
	fromCMAccount ethCommon.Address,
	toCMAccount ethCommon.Address,
) *types.Message {
	responseMsg := &types.Message{
		RequestID:  requestMsg.RequestID,
		Timestamps: requestMsg.Timestamps,
	}

	err := protovalidate.Validate(requestMsg.Content)
	if err != nil {
		errMessage := fmt.Sprintf("request message validation failed: %v", err)
		responseMsg.Content, responseMsg.Type = serviceClient.ErrorResponseAndType(errMessage)
		p.logger.Errorf(errMessage)
		return responseMsg
	}

	p.logger.Infof("CMAccount %s is calling partner-plugin of the CMAccount %s", fromCMAccount, toCMAccount)

	requestMsg.Timestamps.Stamp(metadata.CheckpointP2PRequestMessageSentToPP)

	responseMsg.Content, responseMsg.Type, err = p.partnerPlugin.DoServiceRequest(
		ctx,
		requestMsg,
		serviceClient,
		fromCMAccount,
		toCMAccount,
	)
	if err != nil {
		errMessage := fmt.Sprintf("error calling partner plugin service: %v", err)
		p.logger.Errorf(errMessage)
		p.responseHeaderHandler.AddError(responseMsg.Content, errMessage)
	} else if err := protovalidate.Validate(responseMsg.Content); err != nil {
		errMessage := fmt.Sprintf("response message content validation failed: %v", err)
		p.logger.Errorf(errMessage)
		p.responseHeaderHandler.AddError(responseMsg.Content, errMessage)
	}

	requestMsg.Timestamps.Stamp(metadata.CheckpointP2PResponseMessageReceivedFromPP)

	// is is expected, that PrepareResponseMessage will correctly process failure responses
	p.responseHandler.PrepareResponseMessage(ctx, requestMsg, responseMsg)

	return responseMsg
}

func (p *messageProcessor) forwardToHandler(msg *types.Message) error {
	p.logger.Debugf("Forwarding incoming response message %s (id %s) to its handler", msg.Type, msg.RequestID)
	responseChan, ok := p.getResponseChannel(msg.RequestID)
	if ok {
		responseChan <- msg
		close(responseChan)
		return nil
	}
	err := fmt.Errorf("no response channel for request ID: %s", msg.RequestID)
	p.logger.Errorf("Failed to forward message: %v", err)
	return err
}

func (p *messageProcessor) issueNetworkCheque(ctx context.Context, msg *EncodedSignedMessage) (*cheques.SignedCheque, error) {
	numberOfChunks := big.NewInt(int64(len(msg.ChunkedEncodedMessage)))
	totalNetworkFee := new(big.Int).Mul(config.NetworkFee, numberOfChunks)

	networkFeeCheque, err := p.chequeHandler.IssueCheque(
		ctx,
		p.networkFeeRecipientCMAccountAddress,
		p.networkFeeRecipientBotAddress,
		totalNetworkFee,
	)
	if err != nil {
		err = fmt.Errorf("failed to issue network fee cheque: %w", err)
		p.logger.Error(err)
		return nil, err
	}

	return networkFeeCheque, nil
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

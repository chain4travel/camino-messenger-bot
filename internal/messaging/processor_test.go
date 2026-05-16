// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"

	"github.com/chain4travel/camino-matrix-app-service/config"
	"github.com/chain4travel/camino-messenger-bot/v13/internal/messaging/encryption"
	"github.com/chain4travel/camino-messenger-bot/v13/internal/messaging/message"
	"github.com/chain4travel/camino-messenger-bot/v13/internal/partnerplugin"
	"github.com/chain4travel/camino-messenger-bot/v13/internal/resolver"
	"github.com/chain4travel/camino-messenger-bot/v13/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/v13/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/chequehandler"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/cheques"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v13/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/metadata"
	m "github.com/chain4travel/camino-messenger-bot/v13/tests/matchers"
	ethCommon "github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

type messageProcessorArgs struct {
	messenger       *MockMessenger
	serviceRegistry *MockServiceRegistry
	responseHandler *MockResponseHandler
	partnerPlugin   *partnerplugin.MockPartnerPlugin
	chequeHandler   *chequehandler.MockChequeHandler
	cmAccounts      *cmaccounts.MockService
	encoderDecoder  *MockEncoderDecoder
	resolver        *resolver.MockResolver
}

func defaultMessageProcessorArgs(c *gomock.Controller) messageProcessorArgs {
	return messageProcessorArgs{
		messenger:       NewMockMessenger(c),
		serviceRegistry: NewMockServiceRegistry(c),
		responseHandler: NewMockResponseHandler(c),
		partnerPlugin:   partnerplugin.NewMockPartnerPlugin(c),
		chequeHandler:   chequehandler.NewMockChequeHandler(c),
		cmAccounts:      cmaccounts.NewMockService(c),
		encoderDecoder:  NewMockEncoderDecoder(c),
		resolver:        resolver.NewMockResolver(c),
	}
}

func TestProcessIncomingMessage(t *testing.T) {
	testErr := errors.New("test error")
	requestID := "requestID"
	testSharedKey := encryption.NopKey{SessionKey: []byte("test key")}

	senderBotAddress := ethCommon.Address{1}
	senderCMAccount := ethCommon.Address{2}

	ownBot := ethCommon.Address{3}
	ownCMAccount := ethCommon.Address{4}

	const serviceName = "dummy"
	serviceFee := big.NewInt(1)
	serviceFeeCheque := &cheques.SignedCheque{
		Cheque: cheques.Cheque{
			FromCMAccount: senderCMAccount,
			ToCMAccount:   ownCMAccount,
			ToBot:         ownBot,
		},
		Signature: []byte("signature"),
	}

	networkFeeBot := ethCommon.Address{5}
	networkFeeCMAccount := ethCommon.Address{6}

	responseMessage := &message.Message{
		Type:       generated.PingServiceV1Response,
		RequestID:  requestID,
		Timestamps: metadata.Timestamps{},
	}
	encodedRespMsg := &EncodedSignedMessage{
		ChunkedEncodedMessage: [][]byte{[]byte("chunk1"), []byte("chunk2")},
		Signature:             []byte("signature"),
	}
	numberOfChunks := big.NewInt(int64(len(encodedRespMsg.ChunkedEncodedMessage)))
	networkFee := new(big.Int).Mul(config.NetworkFee, numberOfChunks)
	respNetworkFeeCheque := &cheques.SignedCheque{Signature: []byte("network fee signature")}

	type args struct {
		requestMessage         *message.Message
		serviceFeeCheque       *cheques.SignedCheque
		senderBotAddress       ethCommon.Address
		senderCMAccountAddress ethCommon.Address
		sharedKey              encryption.Key
	}

	tests := map[string]struct {
		messageProcessorArgs func(*gomock.Controller, *messageProcessorArgs, args)
		responseChannels     map[string]chan *message.Message
		args                 args
		expectedErr          error
		require              func(*testing.T, *messageProcessor)
	}{
		"Invalid message type": {
			args: args{
				requestMessage:   &message.Message{Type: "invalid"},
				senderBotAddress: ethCommon.Address{},
			},
			expectedErr: ErrUnknownMessageCategory,
		},
		"Not supported service": {
			messageProcessorArgs: func(_ *gomock.Controller, pArgs *messageProcessorArgs, args args) {
				pArgs.serviceRegistry.EXPECT().GetService(args.requestMessage.Type).Return(nil, false)
			},
			args: args{
				requestMessage: &message.Message{
					Type:       generated.PingServiceV1Request,
					Timestamps: metadata.Timestamps{},
				},
				serviceFeeCheque:       serviceFeeCheque,
				senderBotAddress:       ethCommon.Address{},
				senderCMAccountAddress: serviceFeeCheque.FromCMAccount,
			},
			expectedErr: ErrUnsupportedService,
		},
		"Messenger failed to send message": {
			messageProcessorArgs: func(c *gomock.Controller, pArgs *messageProcessorArgs, a args) {
				rpcService := rpc.NewMockService(c)
				rpcService.EXPECT().Name().Return(serviceName)
				pArgs.serviceRegistry.EXPECT().GetService(a.requestMessage.Type).Return(rpcService, true)
				pArgs.cmAccounts.EXPECT().GetServiceFee(m.Context, ownCMAccount, serviceName).Return(serviceFee, nil)
				pArgs.chequeHandler.EXPECT().VerifyAndStoreCheque(m.Context, a.serviceFeeCheque, a.senderBotAddress, serviceFee).Return(nil)
				pArgs.partnerPlugin.EXPECT().DoServiceRequest(m.Context, a.requestMessage, rpcService, a.serviceFeeCheque.FromCMAccount, a.serviceFeeCheque.ToCMAccount).Return(responseMessage.Content, responseMessage.Type)
				pArgs.responseHandler.EXPECT().PrepareResponseMessage(m.Context, a.requestMessage, equalExceptTimestamps(responseMessage))
				pArgs.encoderDecoder.EXPECT().EncodeMessage(m.Context, equalExceptTimestamps(responseMessage), nil, a.senderBotAddress, a.sharedKey).Return(encodedRespMsg, nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, networkFeeCMAccount, networkFeeBot, networkFee).Return(respNetworkFeeCheque, nil)
				pArgs.messenger.EXPECT().SendMessage(m.Context, encodedRespMsg, senderBotAddress, respNetworkFeeCheque).Return(testErr)
			},
			args: args{
				requestMessage: &message.Message{
					RequestID:  requestID,
					Type:       generated.PingServiceV1Request,
					Timestamps: metadata.Timestamps{},
				},
				serviceFeeCheque:       serviceFeeCheque,
				senderBotAddress:       senderBotAddress,
				senderCMAccountAddress: serviceFeeCheque.FromCMAccount,
				sharedKey:              testSharedKey,
			},
			expectedErr: testErr,
		},
		"OK: process request message": {
			messageProcessorArgs: func(c *gomock.Controller, pArgs *messageProcessorArgs, a args) {
				rpcService := rpc.NewMockService(c)
				rpcService.EXPECT().Name().Return(serviceName)
				pArgs.serviceRegistry.EXPECT().GetService(a.requestMessage.Type).Return(rpcService, true)
				pArgs.cmAccounts.EXPECT().GetServiceFee(m.Context, ownCMAccount, serviceName).Return(serviceFee, nil)
				pArgs.chequeHandler.EXPECT().VerifyAndStoreCheque(m.Context, a.serviceFeeCheque, a.senderBotAddress, serviceFee).Return(nil)
				pArgs.partnerPlugin.EXPECT().DoServiceRequest(m.Context, a.requestMessage, rpcService, a.serviceFeeCheque.FromCMAccount, a.serviceFeeCheque.ToCMAccount).Return(responseMessage.Content, responseMessage.Type)
				pArgs.responseHandler.EXPECT().PrepareResponseMessage(m.Context, a.requestMessage, equalExceptTimestamps(responseMessage))
				pArgs.encoderDecoder.EXPECT().EncodeMessage(m.Context, equalExceptTimestamps(responseMessage), nil, a.senderBotAddress, a.sharedKey).Return(encodedRespMsg, nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, networkFeeCMAccount, networkFeeBot, networkFee).Return(respNetworkFeeCheque, nil)
				pArgs.messenger.EXPECT().SendMessage(m.Context, encodedRespMsg, senderBotAddress, respNetworkFeeCheque).Return(nil)
			},
			args: args{
				requestMessage: &message.Message{
					RequestID:  requestID,
					Type:       generated.PingServiceV1Request,
					Timestamps: metadata.Timestamps{},
				},
				serviceFeeCheque:       serviceFeeCheque,
				senderBotAddress:       senderBotAddress,
				senderCMAccountAddress: serviceFeeCheque.FromCMAccount,
				sharedKey:              testSharedKey,
			},
		},
		"OK: process response message": {
			responseChannels: map[string]chan *message.Message{
				requestID: make(chan *message.Message, 1),
			},
			args: args{
				requestMessage: responseMessage,
			},
			require: func(t *testing.T, p *messageProcessor) {
				responseChan, ok := p.getResponseChannel(requestID)
				require.True(t, ok)
				msgReceived := <-responseChan
				require.Equal(t, responseMessage, msgReceived)
			},
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			messageProcessorArgs := defaultMessageProcessorArgs(ctrl)
			if tt.messageProcessorArgs != nil {
				tt.messageProcessorArgs(ctrl, &messageProcessorArgs, tt.args)
			}

			p := NewMessageProcessor(
				messageProcessorArgs.messenger,
				zap.NewNop().Sugar(),
				time.Duration(0),
				ownBot,
				ownCMAccount,
				networkFeeBot,
				networkFeeCMAccount,
				messageProcessorArgs.serviceRegistry,
				messageProcessorArgs.responseHandler,
				messageProcessorArgs.partnerPlugin,
				messageProcessorArgs.chequeHandler,
				messageProcessorArgs.cmAccounts,
				big.NewInt(0),
				messageProcessorArgs.encoderDecoder,
				messageProcessorArgs.resolver,
			)
			messageProcessor := p.(*messageProcessor)
			for requestID, responseChan := range tt.responseChannels {
				messageProcessor.setResponseChannel(requestID, responseChan)
			}
			err := messageProcessor.processIncomingMessage(tt.args.requestMessage, tt.args.serviceFeeCheque, tt.args.senderBotAddress, tt.args.senderCMAccountAddress, tt.args.sharedKey)
			require.ErrorIs(t, err, tt.expectedErr)

			if tt.require != nil {
				tt.require(t, messageProcessor)
			}
		})
	}
}

func TestSendRequestMessage(t *testing.T) {
	testErr := errors.New("test error")
	requestID := "requestID"

	responseMessage := &message.Message{
		Type:      generated.PingServiceV1Response,
		RequestID: requestID,
	}

	serviceFee := big.NewInt(1)
	ownBot := ethCommon.Address{1}
	ownCMAccount := ethCommon.Address{2}
	recipientBot := ethCommon.Address{3}
	recipientCMAccount := ethCommon.Address{4}

	serviceFeeCheque := &cheques.SignedCheque{
		Cheque: cheques.Cheque{
			FromCMAccount: ownCMAccount,
			ToCMAccount:   recipientCMAccount,
			ToBot:         recipientBot,
		},
		Signature: []byte("signature"),
	}

	networkFeeBot := ethCommon.Address{5}
	networkFeeCMAccount := ethCommon.Address{6}

	encodedReqMsg := &EncodedSignedMessage{
		ChunkedEncodedMessage: [][]byte{[]byte("chunk1"), []byte("chunk2")},
		Signature:             []byte("signature"),
	}
	numberOfChunks := big.NewInt(int64(len(encodedReqMsg.ChunkedEncodedMessage)))
	networkFee := new(big.Int).Mul(config.NetworkFee, numberOfChunks)
	networkFeeCheque := &cheques.SignedCheque{Signature: []byte("network fee signature")}

	type args struct {
		msg                *message.Message
		recipientCMAccount ethCommon.Address
	}

	testSharedKey, err := encryption.NewKey()
	require.NoError(t, err)

	tests := map[string]struct {
		messageProcessorArgs    func(*messageProcessorArgs, args)
		args                    args
		expectedResponseMessage *message.Message
		expectedErr             error
		responses               func(*messageProcessor)
	}{
		"Messenger failed to send message": {
			messageProcessorArgs: func(pArgs *messageProcessorArgs, a args) {
				pArgs.resolver.EXPECT().GetBotAddress(m.Context, recipientCMAccount).Return(recipientBot, nil)
				pArgs.cmAccounts.EXPECT().GetServiceFee(m.Context, recipientCMAccount, a.msg.Type.ToServiceName()).Return(big.NewInt(1), nil)
				pArgs.responseHandler.EXPECT().PrepareRequest(a.msg.Content)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, recipientCMAccount, recipientBot, serviceFee).Return(serviceFeeCheque, nil)
				pArgs.encoderDecoder.EXPECT().EncodeMessage(m.Context, a.msg, serviceFeeCheque, recipientBot, gomock.AssignableToTypeOf(testSharedKey)).Return(encodedReqMsg, nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, networkFeeCMAccount, networkFeeBot, networkFee).Return(networkFeeCheque, nil)
				pArgs.messenger.EXPECT().SendMessage(m.Context, encodedReqMsg, recipientBot, networkFeeCheque).Return(testErr)
			},
			args: args{
				msg: &message.Message{
					Type:       generated.PingServiceV1Request,
					Timestamps: metadata.Timestamps{},
				},
				recipientCMAccount: recipientCMAccount,
			},
			expectedErr: testErr,
		},
		"Response timeout": {
			messageProcessorArgs: func(pArgs *messageProcessorArgs, a args) {
				pArgs.resolver.EXPECT().GetBotAddress(m.Context, recipientCMAccount).Return(recipientBot, nil)
				pArgs.cmAccounts.EXPECT().GetServiceFee(m.Context, recipientCMAccount, a.msg.Type.ToServiceName()).Return(big.NewInt(1), nil)
				pArgs.responseHandler.EXPECT().PrepareRequest(a.msg.Content)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, recipientCMAccount, recipientBot, serviceFee).Return(serviceFeeCheque, nil)
				pArgs.encoderDecoder.EXPECT().EncodeMessage(m.Context, a.msg, serviceFeeCheque, recipientBot, gomock.AssignableToTypeOf(testSharedKey)).Return(encodedReqMsg, nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, networkFeeCMAccount, networkFeeBot, networkFee).Return(networkFeeCheque, nil)
				pArgs.messenger.EXPECT().SendMessage(m.Context, encodedReqMsg, recipientBot, networkFeeCheque).Return(nil)
				pArgs.resolver.EXPECT().SetBotStatus(m.Context, recipientBot, resolver.BotStatusUnreachable).Return(nil)
			},
			args: args{
				msg: &message.Message{
					Type:       generated.PingServiceV1Request,
					Timestamps: metadata.Timestamps{},
				},
				recipientCMAccount: recipientCMAccount,
			},
			expectedErr: ErrExceededResponseTimeout,
		},
		"OK": {
			messageProcessorArgs: func(pArgs *messageProcessorArgs, a args) {
				pArgs.resolver.EXPECT().GetBotAddress(m.Context, recipientCMAccount).Return(recipientBot, nil)
				pArgs.cmAccounts.EXPECT().GetServiceFee(m.Context, recipientCMAccount, a.msg.Type.ToServiceName()).Return(big.NewInt(1), nil)
				pArgs.responseHandler.EXPECT().PrepareRequest(a.msg.Content)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, recipientCMAccount, recipientBot, serviceFee).Return(serviceFeeCheque, nil)
				pArgs.encoderDecoder.EXPECT().EncodeMessage(m.Context, a.msg, serviceFeeCheque, recipientBot, gomock.AssignableToTypeOf(testSharedKey)).Return(encodedReqMsg, nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, networkFeeCMAccount, networkFeeBot, networkFee).Return(networkFeeCheque, nil)
				pArgs.messenger.EXPECT().SendMessage(m.Context, encodedReqMsg, recipientBot, networkFeeCheque).Return(nil)
				pArgs.responseHandler.EXPECT().ProcessResponseMessage(m.Context, a.msg, responseMessage)
				pArgs.resolver.EXPECT().SetBotStatus(m.Context, recipientBot, resolver.BotStatusReachable).Return(nil)
			},
			args: args{
				msg: &message.Message{
					Type:       generated.PingServiceV1Request,
					RequestID:  requestID,
					Timestamps: metadata.Timestamps{},
				},
				recipientCMAccount: recipientCMAccount,
			},
			responses: func(p *messageProcessor) {
				for {
					responseChan, ok := p.getResponseChannel(requestID)
					if ok {
						responseChan <- responseMessage
						return
					}
					time.Sleep(50 * time.Millisecond)
				}
			},
			expectedResponseMessage: responseMessage,
		},
	}

	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			messageProcessorArgs := defaultMessageProcessorArgs(ctrl)
			if tt.messageProcessorArgs != nil {
				tt.messageProcessorArgs(&messageProcessorArgs, tt.args)
			}

			p := NewMessageProcessor(
				messageProcessorArgs.messenger,
				zap.NewNop().Sugar(),
				1*time.Second, // response timeout
				ownBot,
				ownCMAccount,
				networkFeeBot,
				networkFeeCMAccount,
				messageProcessorArgs.serviceRegistry,
				messageProcessorArgs.responseHandler,
				messageProcessorArgs.partnerPlugin,
				messageProcessorArgs.chequeHandler,
				messageProcessorArgs.cmAccounts,
				big.NewInt(1), // max allowed service fee
				messageProcessorArgs.encoderDecoder,
				messageProcessorArgs.resolver,
			)
			if tt.responses != nil {
				go tt.responses(p.(*messageProcessor))
			}
			responseMessage, err := p.SendRequestMessage(context.Background(), tt.args.msg, tt.args.recipientCMAccount)
			require.ErrorIs(t, err, tt.expectedErr)
			require.Equal(t, tt.expectedResponseMessage, responseMessage)
		})
	}
}

func TestStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := gomock.NewController(t)
	serviceRegistry := NewMockServiceRegistry(c)
	cmAccounts := cmaccounts.NewMockService(c)
	chequeHandler := chequehandler.NewMockChequeHandler(c)
	partnerPlugin := partnerplugin.NewMockPartnerPlugin(c)
	messenger := NewMockMessenger(c)
	encoderDecoder := NewMockEncoderDecoder(c)
	responseHandler := NewMockResponseHandler(c)
	resolver := resolver.NewMockResolver(c)

	senderBot := ethCommon.Address{1}
	senderCMAccount := ethCommon.Address{2}

	ownBot := ethCommon.Address{3}
	ownCMAccount := ethCommon.Address{4}

	networkFeeBot := ethCommon.Address{5}
	networkFeeCMAccount := ethCommon.Address{6}

	requestMsg := &message.Message{
		Type:       generated.PingServiceV1Request,
		RequestID:  "requestID",
		Timestamps: metadata.Timestamps{},
	}
	sharedKey := encryption.NopKey{SessionKey: []byte("test shared key")}

	incomingMessages := []EncodedSignedMessageWithSender{}

	// Received message from itself (bot)

	incomingMessages = append(incomingMessages, EncodedSignedMessageWithSender{
		Message:                EncodedSignedMessage{Signature: []byte("self-message (bot) signature")},
		SenderBotAddress:       ownBot,
		SenderCMAccountAddress: senderCMAccount,
	})

	// Received message from itself (cm account)

	encodedBadRequestMsg := EncodedSignedMessageWithSender{
		Message:                EncodedSignedMessage{Signature: []byte("self-message (cm-account) signature")},
		SenderBotAddress:       senderBot,
		SenderCMAccountAddress: ownCMAccount,
	}
	incomingMessages = append(incomingMessages, encodedBadRequestMsg)

	// OK message

	encodedRequestMsg := EncodedSignedMessageWithSender{
		Message:                EncodedSignedMessage{Signature: []byte("message signature")},
		SenderBotAddress:       senderBot,
		SenderCMAccountAddress: senderCMAccount,
	}
	serviceFeeCheque := &cheques.SignedCheque{Cheque: cheques.Cheque{
		FromCMAccount: senderCMAccount,
		ToCMAccount:   ownCMAccount,
		ToBot:         ownBot,
	}}
	incomingMessages = append(incomingMessages, encodedRequestMsg)

	const serviceName = "dummy"
	rpcService := rpc.NewMockService(c)
	rpcService.EXPECT().Name().Return(serviceName)

	serviceFee := big.NewInt(1)
	respNetworkFeeCheque := &cheques.SignedCheque{Signature: []byte("network fee signature")}
	responseMessage := &message.Message{
		Type:       generated.PingServiceV1Response,
		RequestID:  requestMsg.RequestID,
		Timestamps: metadata.Timestamps{},
	}
	encodedRespMsg := &EncodedSignedMessage{
		ChunkedEncodedMessage: [][]byte{[]byte("chunk1"), []byte("chunk2")},
		Signature:             []byte("response signature"),
	}
	numberOfChunks := big.NewInt(int64(len(encodedRespMsg.ChunkedEncodedMessage)))
	networkFee := new(big.Int).Mul(config.NetworkFee, numberOfChunks)

	encoderDecoder.EXPECT().DecodeAndVerifyMessage(ctx, &encodedRequestMsg.Message, encodedRequestMsg.SenderBotAddress).Return(requestMsg, serviceFeeCheque, sharedKey, nil)
	serviceRegistry.EXPECT().GetService(requestMsg.Type).Return(rpcService, true)
	cmAccounts.EXPECT().GetServiceFee(m.Context, ownCMAccount, serviceName).Return(serviceFee, nil)
	chequeHandler.EXPECT().VerifyAndStoreCheque(m.Context, serviceFeeCheque, senderBot, serviceFee).Return(nil)
	responseHandler.EXPECT().PrepareResponseMessage(m.Context, requestMsg, equalExceptTimestamps(responseMessage))
	partnerPlugin.EXPECT().DoServiceRequest(m.Context, requestMsg, rpcService, serviceFeeCheque.FromCMAccount, serviceFeeCheque.ToCMAccount).Return(responseMessage.Content, responseMessage.Type)
	encoderDecoder.EXPECT().EncodeMessage(m.Context, equalExceptTimestamps(responseMessage), nil, senderBot, sharedKey).Return(encodedRespMsg, nil)
	chequeHandler.EXPECT().IssueCheque(m.Context, networkFeeCMAccount, networkFeeBot, networkFee).Return(respNetworkFeeCheque, nil)
	messenger.EXPECT().SendMessage(m.Context, encodedRespMsg, senderBot, respNetworkFeeCheque).Return(nil)

	// set up incoming messages channel

	incomingMessagesChan := make(chan EncodedSignedMessageWithSender, len(incomingMessages))
	for _, msg := range incomingMessages {
		incomingMessagesChan <- msg
	}
	messenger.EXPECT().ReceivedMessageChan().Times(len(incomingMessages) + 1).Return(incomingMessagesChan)

	// set up and start messenger

	NewMessageProcessor(
		messenger,
		zap.NewNop().Sugar(),
		time.Duration(0),
		ownBot,
		ownCMAccount,
		networkFeeBot,
		networkFeeCMAccount,
		serviceRegistry,
		responseHandler,
		partnerPlugin,
		chequeHandler,
		cmAccounts,
		big.NewInt(1),
		encoderDecoder,
		resolver,
	).Start(ctx)

	time.Sleep(1 * time.Second)
}

func equalExceptTimestamps(expectedMessage *message.Message) gomock.Matcher {
	return gomock.Cond(func(actualMessage *message.Message) bool {
		return expectedMessage.RequestID == actualMessage.RequestID &&
			expectedMessage.Type == actualMessage.Type &&
			proto.Equal(expectedMessage.Content, actualMessage.Content)
	})
}

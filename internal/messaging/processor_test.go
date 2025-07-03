// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/chain4travel/camino-matrix-app-service/config"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/common"
	types "github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/partnerplugin"
	rpc "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/chequehandler"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/cheques"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v11/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/matrix"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	m "github.com/chain4travel/camino-messenger-bot/v11/tests/matchers"
	ethCommon "github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

type messageProcessorArgs struct {
	messenger             *MockMessenger
	serviceRegistry       *MockServiceRegistry
	responseHandler       ResponseHandler
	partnerPlugin         *partnerplugin.MockPartnerPlugin
	chequeHandler         *chequehandler.MockChequeHandler
	cmAccounts            *cmaccounts.MockService
	responseHeaderHandler *common.MockResponseHeaderHandler
}

func defaultMessageProcessorArgs(c *gomock.Controller) messageProcessorArgs {
	return messageProcessorArgs{
		messenger:             NewMockMessenger(c),
		serviceRegistry:       NewMockServiceRegistry(c),
		responseHandler:       NoopResponseHandler{},
		partnerPlugin:         partnerplugin.NewMockPartnerPlugin(c),
		chequeHandler:         chequehandler.NewMockChequeHandler(c),
		cmAccounts:            cmaccounts.NewMockService(c),
		responseHeaderHandler: common.NewMockResponseHeaderHandler(c),
	}
}

func TestProcessIncomingMessage(t *testing.T) {
	testErr := errors.New("test error")
	const requestID = "requestID"
	const matrixHomeServer = "localhost"

	senderBotAddress := ethCommon.Address{1}
	senderBotUserID := matrix.UserIDFromAddress(senderBotAddress, matrixHomeServer)
	senderCMAccount := ethCommon.Address{2}

	ownBot := ethCommon.Address{3}
	ownBotUserID := matrix.UserIDFromAddress(ownBot, matrixHomeServer)
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

	responseMessage := &types.Message{
		Type:             generated.PingServiceV1Response,
		RequestID:        requestID,
		NetworkFeeCheque: &cheques.SignedCheque{Signature: []byte("network fee signature")},
	}

	type args struct {
		msg *types.Message
	}

	tests := map[string]struct {
		messageProcessorArgs func(*gomock.Controller, *messageProcessorArgs, args)
		responseChannels     map[string]chan *types.Message
		args                 args
		expectedErr          error
		require              func(*testing.T, *messageProcessor)
	}{
		"Invalid message type": {
			args: args{
				msg: &types.Message{Type: "invalid"},
			},
			expectedErr: ErrUnknownMessageCategory,
		},
		"Not supported service": {
			messageProcessorArgs: func(_ *gomock.Controller, pArgs *messageProcessorArgs, args args) {
				pArgs.serviceRegistry.EXPECT().GetService(args.msg.Type).Return(nil, false)
			},
			args: args{
				msg: &types.Message{
					Type:             generated.PingServiceV1Request,
					Timestamps:       metadata.Timestamps{},
					ServiceFeeCheque: serviceFeeCheque,
					SenderBotUserID:  senderBotUserID,
				},
			},
			expectedErr: ErrUnsupportedService,
		},
		"Messenger failed to send message": {
			messageProcessorArgs: func(c *gomock.Controller, pArgs *messageProcessorArgs, a args) {
				rpcService := rpc.NewMockService(c)
				rpcService.EXPECT().Name().Return(serviceName)
				pArgs.serviceRegistry.EXPECT().GetService(a.msg.Type).Return(rpcService, true)
				pArgs.cmAccounts.EXPECT().GetServiceFee(m.Context, ownCMAccount, serviceName).Return(serviceFee, nil)
				pArgs.chequeHandler.EXPECT().VerifyCheque(m.Context, a.msg.ServiceFeeCheque, senderBotAddress, serviceFee).Return(nil)
				pArgs.partnerPlugin.EXPECT().DoServiceRequest(m.Context, a.msg, rpcService, a.msg.ServiceFeeCheque.FromCMAccount, a.msg.ServiceFeeCheque.ToCMAccount).Return(context.Background(), responseMessage, nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, networkFeeCMAccount, networkFeeBot, config.NetworkFee).Return(responseMessage.NetworkFeeCheque, nil)
				pArgs.messenger.EXPECT().SendMessage(m.Context, responseMessage, senderBotUserID).Return(testErr)
			},
			args: args{
				msg: &types.Message{
					Type:             generated.PingServiceV1Request,
					Timestamps:       metadata.Timestamps{},
					ServiceFeeCheque: serviceFeeCheque,
					SenderBotUserID:  senderBotUserID,
				},
			},
			expectedErr: testErr,
		},
		"OK: process request message": {
			messageProcessorArgs: func(c *gomock.Controller, pArgs *messageProcessorArgs, a args) {
				rpcService := rpc.NewMockService(c)
				rpcService.EXPECT().Name().Return(serviceName)
				pArgs.serviceRegistry.EXPECT().GetService(a.msg.Type).Return(rpcService, true)
				pArgs.cmAccounts.EXPECT().GetServiceFee(m.Context, ownCMAccount, serviceName).Return(serviceFee, nil)
				pArgs.chequeHandler.EXPECT().VerifyCheque(m.Context, a.msg.ServiceFeeCheque, senderBotAddress, serviceFee).Return(nil)
				pArgs.partnerPlugin.EXPECT().DoServiceRequest(m.Context, a.msg, rpcService, a.msg.ServiceFeeCheque.FromCMAccount, a.msg.ServiceFeeCheque.ToCMAccount).Return(context.Background(), responseMessage, nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, networkFeeCMAccount, networkFeeBot, config.NetworkFee).Return(responseMessage.NetworkFeeCheque, nil)
				pArgs.messenger.EXPECT().SendMessage(m.Context, responseMessage, senderBotUserID).Return(nil)
			},
			args: args{
				msg: &types.Message{
					Type:             generated.PingServiceV1Request,
					Timestamps:       metadata.Timestamps{},
					ServiceFeeCheque: serviceFeeCheque,
					SenderBotUserID:  senderBotUserID,
				},
			},
		},
		"OK: process response message": {
			responseChannels: map[string]chan *types.Message{
				requestID: make(chan *types.Message, 1),
			},
			args: args{
				msg: responseMessage,
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
				ownBotUserID,
				ownCMAccount,
				networkFeeBot,
				networkFeeCMAccount,
				messageProcessorArgs.serviceRegistry,
				messageProcessorArgs.responseHandler,
				messageProcessorArgs.partnerPlugin,
				messageProcessorArgs.chequeHandler,
				&noopCompressor{},
				messageProcessorArgs.cmAccounts,
				messageProcessorArgs.responseHeaderHandler,
				big.NewInt(0),
			)
			messageProcessor := p.(*messageProcessor)
			for requestID, responseChan := range tt.responseChannels {
				messageProcessor.setResponseChannel(requestID, responseChan)
			}
			err := messageProcessor.processIncomingMessage(tt.args.msg)
			require.ErrorIs(t, err, tt.expectedErr)

			if tt.require != nil {
				tt.require(t, messageProcessor)
			}
		})
	}
}

func TestSendRequestMessage(t *testing.T) {
	testErr := errors.New("test error")
	const requestID = "requestID"

	responseMessage := &types.Message{
		Type:      generated.PingServiceV1Response,
		RequestID: requestID,
	}

	const matrixHomeServer = "localhost"

	serviceFee := big.NewInt(1)
	ownBot := ethCommon.Address{1}
	ownBotUserID := matrix.UserIDFromAddress(ownBot, matrixHomeServer)
	ownCMAccount := ethCommon.Address{2}
	recipientBot := ethCommon.Address{3}
	recipientBotUserID := matrix.UserIDFromAddress(recipientBot, matrixHomeServer)
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

	networkFeeCheque := &cheques.SignedCheque{Signature: []byte("network fee signature")}

	type args struct {
		msg                *types.Message
		recipientCMAccount ethCommon.Address
	}

	tests := map[string]struct {
		messageProcessorArgs    func(*messageProcessorArgs, args)
		args                    args
		expectedResponseMessage *types.Message
		expectedErr             error
		responses               func(*messageProcessor)
	}{
		"Messenger failed to send message": {
			messageProcessorArgs: func(pArgs *messageProcessorArgs, a args) {
				pArgs.cmAccounts.EXPECT().GetFirstChequeOperator(m.Context, recipientCMAccount).Return(recipientBot, nil)
				pArgs.cmAccounts.EXPECT().IsBotAllowed(m.Context, ownCMAccount, ownBot).Return(true, nil)
				pArgs.cmAccounts.EXPECT().GetServiceFee(m.Context, recipientCMAccount, a.msg.Type.ToServiceName()).Return(big.NewInt(1), nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, recipientCMAccount, recipientBot, serviceFee).Return(serviceFeeCheque, nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, networkFeeCMAccount, networkFeeBot, config.NetworkFee).Return(networkFeeCheque, nil)
				pArgs.messenger.EXPECT().SendMessage(m.Context, a.msg, recipientBotUserID).Return(testErr)
			},
			args: args{
				msg: &types.Message{
					Type:      generated.PingServiceV1Request,
					RequestID: requestID,
				},
				recipientCMAccount: recipientCMAccount,
			},
			expectedErr: testErr,
		},
		"Response timeout": {
			messageProcessorArgs: func(pArgs *messageProcessorArgs, a args) {
				pArgs.cmAccounts.EXPECT().GetFirstChequeOperator(m.Context, recipientCMAccount).Return(recipientBot, nil)
				pArgs.cmAccounts.EXPECT().IsBotAllowed(m.Context, ownCMAccount, ownBot).Return(true, nil)
				pArgs.cmAccounts.EXPECT().GetServiceFee(m.Context, recipientCMAccount, a.msg.Type.ToServiceName()).Return(big.NewInt(1), nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, recipientCMAccount, recipientBot, serviceFee).Return(serviceFeeCheque, nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, networkFeeCMAccount, networkFeeBot, config.NetworkFee).Return(networkFeeCheque, nil)
				pArgs.messenger.EXPECT().SendMessage(m.Context, a.msg, recipientBotUserID).Return(nil)
			},
			args: args{
				msg: &types.Message{
					Type:      generated.PingServiceV1Request,
					RequestID: requestID,
				},
				recipientCMAccount: recipientCMAccount,
			},
			expectedErr: ErrExceededResponseTimeout,
		},
		"OK": {
			messageProcessorArgs: func(pArgs *messageProcessorArgs, a args) {
				pArgs.cmAccounts.EXPECT().GetFirstChequeOperator(m.Context, recipientCMAccount).Return(recipientBot, nil)
				pArgs.cmAccounts.EXPECT().IsBotAllowed(m.Context, ownCMAccount, ownBot).Return(true, nil)
				pArgs.cmAccounts.EXPECT().GetServiceFee(m.Context, recipientCMAccount, a.msg.Type.ToServiceName()).Return(big.NewInt(1), nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, recipientCMAccount, recipientBot, serviceFee).Return(serviceFeeCheque, nil)
				pArgs.chequeHandler.EXPECT().IssueCheque(m.Context, networkFeeCMAccount, networkFeeBot, config.NetworkFee).Return(networkFeeCheque, nil)
				pArgs.messenger.EXPECT().SendMessage(m.Context, a.msg, recipientBotUserID).Return(nil)
			},
			args: args{
				msg: &types.Message{
					Type:      generated.PingServiceV1Request,
					RequestID: requestID,
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
				ownBotUserID,
				ownCMAccount,
				networkFeeBot,
				networkFeeCMAccount,
				messageProcessorArgs.serviceRegistry,
				messageProcessorArgs.responseHandler,
				messageProcessorArgs.partnerPlugin,
				messageProcessorArgs.chequeHandler,
				&noopCompressor{},
				messageProcessorArgs.cmAccounts,
				messageProcessorArgs.responseHeaderHandler,
				big.NewInt(1), // max allowed service fee
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
	responseHeaderHandler := common.NewMockResponseHeaderHandler(c)

	const matrixHomeServer = "localhost"

	senderBot := ethCommon.Address{1}
	senderBotUserID := matrix.UserIDFromAddress(senderBot, matrixHomeServer)
	senderCMAccount := ethCommon.Address{2}

	ownBot := ethCommon.Address{3}
	ownBotUserID := matrix.UserIDFromAddress(ownBot, matrixHomeServer)
	ownCMAccount := ethCommon.Address{4}

	networkFeeBot := ethCommon.Address{5}
	networkFeeCMAccount := ethCommon.Address{6}

	incomingMessages := []types.Message{}

	// OK message

	requestMsg := &types.Message{
		Type:       generated.PingServiceV1Request,
		RequestID:  "requestID",
		Timestamps: metadata.Timestamps{},
		ServiceFeeCheque: &cheques.SignedCheque{Cheque: cheques.Cheque{
			FromCMAccount: senderCMAccount,
			ToCMAccount:   ownCMAccount,
			ToBot:         ownBot,
		}},
		SenderBotUserID: senderBotUserID,
	}
	incomingMessages = append(incomingMessages, *requestMsg)

	const serviceName = "dummy"
	rpcService := rpc.NewMockService(c)
	rpcService.EXPECT().Name().Return(serviceName)

	serviceFee := big.NewInt(1)
	respNetworkFeeCheque := &cheques.SignedCheque{Signature: []byte("network fee signature")}
	responseMessage := &types.Message{
		Type:      generated.PingServiceV1Response,
		RequestID: requestMsg.RequestID,
	}

	serviceRegistry.EXPECT().GetService(requestMsg.Type).Return(rpcService, true)
	cmAccounts.EXPECT().GetServiceFee(m.Context, ownCMAccount, serviceName).Return(serviceFee, nil)
	chequeHandler.EXPECT().VerifyCheque(m.Context, requestMsg.ServiceFeeCheque, senderBot, serviceFee).Return(nil)
	partnerPlugin.EXPECT().DoServiceRequest(m.Context, requestMsg, rpcService, requestMsg.ServiceFeeCheque.FromCMAccount, requestMsg.ServiceFeeCheque.ToCMAccount).Return(context.Background(), responseMessage, nil)
	chequeHandler.EXPECT().IssueCheque(m.Context, networkFeeCMAccount, networkFeeBot, config.NetworkFee).Return(respNetworkFeeCheque, nil)
	messenger.EXPECT().SendMessage(m.Context, responseMessage, senderBotUserID).Return(nil)

	// set up incoming messages channel

	incomingMessagesChan := make(chan types.Message, len(incomingMessages))
	for _, msg := range incomingMessages {
		incomingMessagesChan <- msg
	}
	messenger.EXPECT().ReceivedMessageChan().Times(len(incomingMessages) + 1).Return(incomingMessagesChan)

	// set up and start messenger

	NewMessageProcessor(
		messenger,
		zap.NewNop().Sugar(),
		time.Duration(0),
		ownBotUserID,
		ownCMAccount,
		networkFeeBot,
		networkFeeCMAccount,
		serviceRegistry,
		NoopResponseHandler{},
		partnerPlugin,
		chequeHandler,
		&noopCompressor{},
		cmAccounts,
		responseHeaderHandler,
		big.NewInt(1),
	).Start(ctx)

	time.Sleep(1 * time.Second)
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"maunium.net/go/mautrix/id"

	"go.uber.org/mock/gomock"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/common"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/partnerplugin"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/chequehandler"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/cheques"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v11/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	ethCommon "github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

var (
	userID       = id.UserID("0x4626cb544230e4d13fb72950501ff91740116a0a:localhost")
	requestID    = "requestID"
	errSomeError = errors.New("some error")
)

func TestProcessIncomingMessage(t *testing.T) {
	responseMessage := types.Message{
		Type:      generated.PingServiceV1Response,
		RequestID: requestID,
	}

	serviceFeeCheque := &cheques.SignedCheque{
		Cheque: cheques.Cheque{
			FromCMAccount: ethCommon.Address{1},
			ToCMAccount:   ethCommon.Address{2},
		},
	}

	mockCtrl := gomock.NewController(t)
	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockService := rpc.NewMockService(mockCtrl)
	mockMessenger := NewMockMessenger(mockCtrl)
	mockCMAccounts := cmaccounts.NewMockService(mockCtrl)
	mockChequeHandler := chequehandler.NewMockChequeHandler(mockCtrl)
	mockPartnerPlugin := partnerplugin.NewMockPartnerPlugin(mockCtrl)
	mockResponseHeaderHandler := common.NewMockResponseHeaderHandler(mockCtrl)

	type fields struct {
		messenger             Messenger
		serviceRegistry       ServiceRegistry
		responseHandler       ResponseHandler
		partnerPlugin         partnerplugin.PartnerPlugin
		chequeHandler         chequehandler.ChequeHandler
		compressor            compression.Compressor[*types.Message, [][]byte]
		cmAccounts            cmaccounts.Service
		responseHeaderHandler common.ResponseHeaderHandler
	}
	type args struct {
		msg *types.Message
	}
	tests := map[string]struct {
		fields  fields
		args    args
		prepare func(*messageProcessor)
		err     error
		assert  func(*testing.T, *messageProcessor)
	}{
		"err: invalid message type": {
			fields: fields{},
			args: args{
				msg: &types.Message{Type: "invalid"},
			},
			err: ErrUnknownMessageCategory,
		},
		"err: unsupported request message": {
			fields: fields{
				serviceRegistry: mockServiceRegistry,
			},
			prepare: func(*messageProcessor) {
				mockServiceRegistry.EXPECT().GetService(gomock.Any()).Return(nil, false)
			},
			args: args{
				msg: &types.Message{
					Type: generated.PingServiceV1Request,
				},
			},
			err: ErrUnsupportedService,
		},
		"err: process request message failed": {
			fields: fields{
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				partnerPlugin:         mockPartnerPlugin,
				chequeHandler:         mockChequeHandler,
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccounts:            mockCMAccounts,
				responseHeaderHandler: mockResponseHeaderHandler,
			},
			prepare: func(*messageProcessor) {
				mockService.EXPECT().Name().Return("dummy")
				mockServiceRegistry.EXPECT().GetService(gomock.Any()).Return(mockService, true)
				mockChequeHandler.EXPECT().VerifyCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockCMAccounts.EXPECT().GetServiceFee(gomock.Any(), gomock.Any(), gomock.Any()).Return(big.NewInt(1), nil)
				mockPartnerPlugin.EXPECT().DoServiceRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(context.Background(), &responseMessage, nil)
				mockChequeHandler.EXPECT().IssueCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&cheques.SignedCheque{}, nil)
				mockMessenger.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(errSomeError)
			},
			args: args{
				msg: &types.Message{
					Type:             generated.PingServiceV1Request,
					ServiceFeeCheque: serviceFeeCheque,
					Timestamps:       metadata.Timestamps{},
				},
			},
			err: errSomeError,
		},
		"success: process request message": {
			fields: fields{
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				partnerPlugin:         mockPartnerPlugin,
				chequeHandler:         mockChequeHandler,
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccounts:            mockCMAccounts,
				responseHeaderHandler: mockResponseHeaderHandler,
			},
			prepare: func(*messageProcessor) {
				mockService.EXPECT().Name().Return("dummy")
				mockServiceRegistry.EXPECT().GetService(gomock.Any()).Return(mockService, true)
				mockChequeHandler.EXPECT().VerifyCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockCMAccounts.EXPECT().GetServiceFee(gomock.Any(), gomock.Any(), gomock.Any()).Return(big.NewInt(1), nil)
				mockPartnerPlugin.EXPECT().DoServiceRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(context.Background(), &responseMessage, nil)
				mockChequeHandler.EXPECT().IssueCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&cheques.SignedCheque{}, nil)
				mockMessenger.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			args: args{
				msg: &types.Message{
					Type:             generated.PingServiceV1Request,
					ServiceFeeCheque: serviceFeeCheque,
					Timestamps:       metadata.Timestamps{},
				},
			},
		},
		"success: process response message": {
			fields: fields{
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				partnerPlugin:         mockPartnerPlugin,
				chequeHandler:         mockChequeHandler,
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccounts:            mockCMAccounts,
				responseHeaderHandler: mockResponseHeaderHandler,
			},
			prepare: func(p *messageProcessor) {
				p.responseChannels[requestID] = make(chan *types.Message, 1)
			},
			args: args{
				msg: &responseMessage,
			},
			assert: func(t *testing.T, p *messageProcessor) {
				responseChan, ok := p.getResponseChannel(requestID)
				require.True(t, ok)
				msgReceived := <-responseChan
				require.Equal(t, responseMessage, *msgReceived)
			},
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			p := NewMessageProcessor(
				tt.fields.messenger,
				zap.NewNop().Sugar(),
				time.Duration(0),
				userID,
				ethCommon.Address{},
				ethCommon.Address{},
				ethCommon.Address{},
				tt.fields.serviceRegistry,
				tt.fields.responseHandler,
				tt.fields.partnerPlugin,
				tt.fields.chequeHandler,
				tt.fields.compressor,
				tt.fields.cmAccounts,
				tt.fields.responseHeaderHandler,
				big.NewInt(0),
			)
			if tt.prepare != nil {
				tt.prepare(p.(*messageProcessor))
			}
			err := p.ProcessIncomingMessage(tt.args.msg)
			require.ErrorIs(t, err, tt.err)

			if tt.assert != nil {
				tt.assert(t, p.(*messageProcessor))
			}
		})
	}
}

func TestSendRequestMessage(t *testing.T) {
	productListResponse := &types.Message{
		Type:      generated.PingServiceV1Response,
		RequestID: requestID,
	}

	recipientCMAccount := ethCommon.Address{1}

	mockCtrl := gomock.NewController(t)
	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockMessenger := NewMockMessenger(mockCtrl)
	mockCMAccounts := cmaccounts.NewMockService(mockCtrl)
	mockChequeHandler := chequehandler.NewMockChequeHandler(mockCtrl)
	mockResponseHeaderHandler := common.NewMockResponseHeaderHandler(mockCtrl)

	type fields struct {
		responseTimeout       time.Duration
		messenger             Messenger
		serviceRegistry       ServiceRegistry
		responseHandler       ResponseHandler
		partnerPlugin         partnerplugin.PartnerPlugin
		chequeHandler         chequehandler.ChequeHandler
		compressor            compression.Compressor[*types.Message, [][]byte]
		cmAccounts            cmaccounts.Service
		responseHeaderHandler common.ResponseHeaderHandler
		maxAllowedServiceFee  *big.Int
	}
	type args struct {
		msg                *types.Message
		recipientCMAccount ethCommon.Address
	}
	tests := map[string]struct {
		fields                 fields
		args                   args
		want                   *types.Message
		err                    error
		prepare                func()
		writeResponseToChannel func(*messageProcessor)
	}{
		"err: non-request outbound message": {
			fields: fields{
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				chequeHandler:         mockChequeHandler,
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccounts:            mockCMAccounts,
				responseHeaderHandler: mockResponseHeaderHandler,
			},
			args: args{
				msg: &types.Message{
					RequestID:  requestID,
					Type:       generated.PingServiceV1Response,
					Timestamps: metadata.Timestamps{},
				},
				recipientCMAccount: recipientCMAccount,
			},
			err: ErrOnlyRequestMessagesAllowed,
		},
		"err: awaiting-response-timeout exceeded": {
			fields: fields{
				responseTimeout:       10 * time.Millisecond, // 10ms
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				chequeHandler:         mockChequeHandler,
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccounts:            mockCMAccounts,
				responseHeaderHandler: mockResponseHeaderHandler,
				maxAllowedServiceFee:  big.NewInt(1),
			},
			args: args{
				msg: &types.Message{
					RequestID:  requestID,
					Type:       generated.PingServiceV1Request,
					Timestamps: metadata.Timestamps{},
				},
				recipientCMAccount: recipientCMAccount,
			},
			prepare: func() {
				mockCMAccounts.EXPECT().GetFirstChequeOperator(gomock.Any(), gomock.Any()).Return(ethCommon.Address{}, nil)
				mockCMAccounts.EXPECT().GetServiceFee(gomock.Any(), gomock.Any(), gomock.Any()).Return(big.NewInt(1), nil)
				mockCMAccounts.EXPECT().IsBotAllowed(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				mockChequeHandler.EXPECT().IssueCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(&cheques.SignedCheque{}, nil)
				mockMessenger.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			err: ErrExceededResponseTimeout,
		},
		"err: while sending request": {
			fields: fields{
				responseTimeout:       100 * time.Millisecond, // 100ms
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				chequeHandler:         mockChequeHandler,
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccounts:            mockCMAccounts,
				responseHeaderHandler: mockResponseHeaderHandler,
				maxAllowedServiceFee:  big.NewInt(1),
			},
			args: args{
				msg: &types.Message{
					RequestID:  requestID,
					Type:       generated.PingServiceV1Request,
					Timestamps: metadata.Timestamps{},
				},
				recipientCMAccount: recipientCMAccount,
			},
			prepare: func() {
				mockCMAccounts.EXPECT().GetFirstChequeOperator(gomock.Any(), gomock.Any()).Return(ethCommon.Address{}, nil)
				mockCMAccounts.EXPECT().GetServiceFee(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(big.NewInt(1), nil)
				mockCMAccounts.EXPECT().IsBotAllowed(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				mockChequeHandler.EXPECT().IssueCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(&cheques.SignedCheque{}, nil)
				mockMessenger.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errSomeError)
			},
			err: errSomeError,
		},
		"success: response before timeout": {
			fields: fields{
				responseTimeout:       500 * time.Millisecond, // long enough timeout for response to be received
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				chequeHandler:         mockChequeHandler,
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccounts:            mockCMAccounts,
				responseHeaderHandler: mockResponseHeaderHandler,
				maxAllowedServiceFee:  big.NewInt(1),
			},
			args: args{
				msg: &types.Message{
					RequestID:  requestID,
					Type:       generated.PingServiceV1Request,
					Timestamps: metadata.Timestamps{},
				},
				recipientCMAccount: recipientCMAccount,
			},
			prepare: func() {
				mockCMAccounts.EXPECT().GetFirstChequeOperator(gomock.Any(), gomock.Any()).Return(ethCommon.Address{}, nil)
				mockCMAccounts.EXPECT().GetServiceFee(gomock.Any(), gomock.Any(), gomock.Any()).Return(big.NewInt(1), nil)
				mockCMAccounts.EXPECT().IsBotAllowed(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				mockMessenger.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockChequeHandler.EXPECT().IssueCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(&cheques.SignedCheque{}, nil)
			},
			writeResponseToChannel: func(p *messageProcessor) {
				done := func() bool {
					responseChan, ok := p.getResponseChannel(requestID)
					if ok {
						responseChan <- productListResponse
						return true
					}
					return false
				}
				for {
					// wait until the response channel is created
					if done() {
						break
					}
				}
			},
			want: productListResponse,
		},
	}

	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			p := NewMessageProcessor(
				tt.fields.messenger,
				zap.NewNop().Sugar(),
				tt.fields.responseTimeout,
				userID,
				ethCommon.Address{},
				ethCommon.Address{},
				ethCommon.Address{},
				tt.fields.serviceRegistry,
				tt.fields.responseHandler,
				tt.fields.partnerPlugin,
				tt.fields.chequeHandler,
				tt.fields.compressor,
				tt.fields.cmAccounts,
				tt.fields.responseHeaderHandler,
				big.NewInt(1),
			)
			if tt.prepare != nil {
				tt.prepare()
			}
			if tt.writeResponseToChannel != nil {
				go tt.writeResponseToChannel(p.(*messageProcessor))
			}
			got, err := p.SendRequestMessage(context.Background(), tt.args.msg, tt.args.recipientCMAccount)

			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestStart(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	serviceFeeCheque := &cheques.SignedCheque{
		Cheque: cheques.Cheque{
			FromCMAccount: ethCommon.Address{1},
			ToCMAccount:   ethCommon.Address{2},
		},
	}

	mockService := rpc.NewMockService(mockCtrl)
	mockService.EXPECT().Name().Return("dummy").Times(2)

	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockServiceRegistry.EXPECT().GetService(gomock.Any()).Return(mockService, true).AnyTimes()

	mockCMAccounts := cmaccounts.NewMockService(mockCtrl)
	mockCMAccounts.EXPECT().GetServiceFee(gomock.Any(), gomock.Any(), gomock.Any()).Return(big.NewInt(1), nil).Times(2)

	mockChequeHandler := chequehandler.NewMockChequeHandler(mockCtrl)
	mockChequeHandler.EXPECT().VerifyCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(2)
	mockChequeHandler.EXPECT().IssueCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&cheques.SignedCheque{}, nil).Times(2)

	mockPartnerPlugin := partnerplugin.NewMockPartnerPlugin(mockCtrl)
	mockPartnerPlugin.EXPECT().DoServiceRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(context.Background(), &types.Message{}, nil)
	mockMessenger := NewMockMessenger(mockCtrl)
	mockMessenger.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(2)

	mockResponseHeaderHandler := common.NewMockResponseHeaderHandler(mockCtrl)

	ch := make(chan types.Message, 5) // incoming messages

	// msg without sender
	ch <- types.Message{Timestamps: metadata.Timestamps{}} // should fail
	// msg with sender, but without valid msgType
	ch <- types.Message{Timestamps: metadata.Timestamps{}, ServiceFeeCheque: serviceFeeCheque} // should fail
	// msg with sender and valid msgType
	ch <- types.Message{
		Timestamps:       metadata.Timestamps{},
		Type:             generated.PingServiceV1Request,
		ServiceFeeCheque: serviceFeeCheque,
		SenderBotUserID:  userID,
	}
	// 2nd msg with sender == userID and valid msgType
	ch <- types.Message{
		Timestamps:       metadata.Timestamps{},
		Type:             generated.AccommodationProductInfoServiceV2Request,
		ServiceFeeCheque: serviceFeeCheque,
		SenderBotUserID:  userID,
	}

	// mocks
	mockMessenger.EXPECT().Inbound().AnyTimes().Return(ch)

	ctx, cancel := context.WithCancel(context.Background())
	p := NewMessageProcessor(
		mockMessenger,
		zap.NewNop().Sugar(),
		time.Duration(0),
		userID,
		ethCommon.Address{},
		ethCommon.Address{},
		ethCommon.Address{},
		mockServiceRegistry,
		NoopResponseHandler{},
		mockPartnerPlugin,
		mockChequeHandler,
		&noopCompressor{},
		mockCMAccounts,
		mockResponseHeaderHandler,
		big.NewInt(1),
	)

	go p.Start(ctx)

	time.Sleep(1 * time.Second)
	cancel()
}

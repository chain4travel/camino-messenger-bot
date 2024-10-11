/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"maunium.net/go/mautrix/id"

	"go.uber.org/mock/gomock"

	"github.com/chain4travel/camino-messenger-bot/internal/compression"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/pkg/chequehandler"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts"
	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

var (
	userID        = id.UserID("userID")
	anotherUserID = "anotherUserID"
	requestID     = "requestID"
	errSomeError  = errors.New("some error")

	dummyCheque = cheques.SignedCheque{
		Cheque: cheques.Cheque{
			FromCMAccount: zeroAddress,
			ToCMAccount:   zeroAddress,
			ToBot:         zeroAddress,
			Counter:       big.NewInt(0),
			Amount:        big.NewInt(0),
			CreatedAt:     big.NewInt(0),
			ExpiresAt:     big.NewInt(0),
		},
		Signature: []byte("signature"),
	}
)

func TestProcessInbound(t *testing.T) {
	responseMessage := types.Message{
		Type: generated.PingServiceV1Response,
		Metadata: metadata.Metadata{
			RequestID: requestID,
			Sender:    anotherUserID,
			Cheques:   []cheques.SignedCheque{},
		},
	}

	mockCtrl := gomock.NewController(t)
	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockService := rpc.NewMockService(mockCtrl)
	mockMessenger := NewMockMessenger(mockCtrl)
	mockCMAccounts := cmaccounts.NewMockService(mockCtrl)
	mockChequeHandler := chequehandler.NewMockChequeHandler(mockCtrl)

	type fields struct {
		messenger       Messenger
		serviceRegistry ServiceRegistry
		responseHandler ResponseHandler
		chequeHandler   chequehandler.ChequeHandler
		compressor      compression.Compressor[*types.Message, [][]byte]
		cmAccounts      cmaccounts.Service
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
				msg: &types.Message{Type: "invalid", Metadata: metadata.Metadata{Sender: anotherUserID, Cheques: []cheques.SignedCheque{}}},
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
					Metadata: metadata.Metadata{
						Sender:  anotherUserID,
						Cheques: []cheques.SignedCheque{},
					},
				},
			},
			err: ErrUnsupportedService,
		},
		"ignore own outbound messages": {
			fields: fields{},
			args: args{
				msg: &types.Message{Metadata: metadata.Metadata{}, Sender: userID},
			},
			err: nil, // no error, msg will be just ignored
		},
		"err: process request message failed": {
			fields: fields{
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				chequeHandler:   mockChequeHandler,
				messenger:       mockMessenger,
				compressor:      &noopCompressor{},
				cmAccounts:      mockCMAccounts,
			},
			prepare: func(*messageProcessor) {
				mockService.EXPECT().Name().Return("dummy")
				mockService.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, generated.PingServiceV1Response, nil)
				mockServiceRegistry.EXPECT().GetService(gomock.Any()).Return(mockService, true)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errSomeError)
				mockChequeHandler.EXPECT().VerifyCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockCMAccounts.EXPECT().GetServiceFee(gomock.Any(), gomock.Any(), gomock.Any()).Return(big.NewInt(1), nil)
			},
			args: args{
				msg: &types.Message{
					Type: generated.PingServiceV1Request,
					Metadata: metadata.Metadata{
						Sender:  anotherUserID,
						Cheques: []cheques.SignedCheque{dummyCheque},
					},
				},
			},
			err: errSomeError,
		},
		"success: process request message": {
			fields: fields{
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				chequeHandler:   mockChequeHandler,
				messenger:       mockMessenger,
				compressor:      &noopCompressor{},
				cmAccounts:      mockCMAccounts,
			},
			prepare: func(*messageProcessor) {
				mockService.EXPECT().Name().Return("dummy")
				mockService.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, generated.PingServiceV1Response, nil)
				mockServiceRegistry.EXPECT().GetService(gomock.Any()).Return(mockService, true)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockChequeHandler.EXPECT().VerifyCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockCMAccounts.EXPECT().GetServiceFee(gomock.Any(), gomock.Any(), gomock.Any()).Return(big.NewInt(1), nil)
			},
			args: args{
				msg: &types.Message{
					Type: generated.PingServiceV1Request,
					Metadata: metadata.Metadata{
						Sender:  anotherUserID,
						Cheques: []cheques.SignedCheque{dummyCheque},
					},
				},
			},
		},
		"success: process response message": {
			fields: fields{
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				chequeHandler:   mockChequeHandler,
				messenger:       mockMessenger,
				compressor:      &noopCompressor{},
				cmAccounts:      mockCMAccounts,
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
				common.Address{},
				common.Address{},
				common.Address{},
				tt.fields.serviceRegistry,
				tt.fields.responseHandler,
				tt.fields.chequeHandler,
				tt.fields.compressor,
				tt.fields.cmAccounts,
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
		Type:     generated.PingServiceV1Response,
		Metadata: metadata.Metadata{RequestID: requestID},
	}

	mockCtrl := gomock.NewController(t)
	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockMessenger := NewMockMessenger(mockCtrl)
	mockCMAccounts := cmaccounts.NewMockService(mockCtrl)
	mockChequeHandler := chequehandler.NewMockChequeHandler(mockCtrl)

	type fields struct {
		responseTimeout time.Duration
		messenger       Messenger
		serviceRegistry ServiceRegistry
		responseHandler ResponseHandler
		chequeHandler   chequehandler.ChequeHandler
		compressor      compression.Compressor[*types.Message, [][]byte]
		cmAccounts      cmaccounts.Service
	}
	type args struct {
		msg *types.Message
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
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				chequeHandler:   mockChequeHandler,
				messenger:       mockMessenger,
				compressor:      &noopCompressor{},
				cmAccounts:      mockCMAccounts,
			},
			args: args{
				msg: &types.Message{Type: generated.PingServiceV1Response},
			},
			err: ErrOnlyRequestMessagesAllowed,
		},
		"err: missing recipient": {
			fields: fields{
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				chequeHandler:   mockChequeHandler,
				messenger:       mockMessenger,
				compressor:      &noopCompressor{},
				cmAccounts:      mockCMAccounts,
			},
			args: args{
				msg: &types.Message{Type: generated.PingServiceV1Request},
			},
			err: ErrMissingRecipient,
		},
		"err: awaiting-response-timeout exceeded": {
			fields: fields{
				responseTimeout: 10 * time.Millisecond, // 10ms
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				chequeHandler:   mockChequeHandler,
				messenger:       mockMessenger,
				compressor:      &noopCompressor{},
				cmAccounts:      mockCMAccounts,
			},
			args: args{
				msg: &types.Message{
					Type:     generated.PingServiceV1Request,
					Metadata: metadata.Metadata{Recipient: anotherUserID},
				},
			},
			prepare: func() {
				mockCMAccounts.EXPECT().GetChequeOperators(gomock.Any(), gomock.Any()).Return([]common.Address{{}}, nil)
				mockCMAccounts.EXPECT().GetServiceFee(gomock.Any(), gomock.Any(), gomock.Any()).Return(big.NewInt(1), nil)
				mockCMAccounts.EXPECT().IsBotAllowed(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				mockChequeHandler.EXPECT().IssueCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(&cheques.SignedCheque{}, nil)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			err: ErrExceededResponseTimeout,
		},
		"err: while sending request": {
			fields: fields{
				responseTimeout: 100 * time.Millisecond, // 100ms
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				chequeHandler:   mockChequeHandler,
				messenger:       mockMessenger,
				compressor:      &noopCompressor{},
				cmAccounts:      mockCMAccounts,
			},
			args: args{
				msg: &types.Message{
					Type:     generated.PingServiceV1Request,
					Metadata: metadata.Metadata{Recipient: anotherUserID},
				},
			},
			prepare: func() {
				mockCMAccounts.EXPECT().GetChequeOperators(gomock.Any(), gomock.Any()).
					Return([]common.Address{{}}, nil)
				mockCMAccounts.EXPECT().GetServiceFee(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(big.NewInt(1), nil)
				mockCMAccounts.EXPECT().IsBotAllowed(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				mockChequeHandler.EXPECT().IssueCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(&cheques.SignedCheque{}, nil)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errSomeError)
			},
			err: errSomeError,
		},
		"success: response before timeout": {
			fields: fields{
				responseTimeout: 500 * time.Millisecond, // long enough timeout for response to be received
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				chequeHandler:   mockChequeHandler,
				messenger:       mockMessenger,
				compressor:      &noopCompressor{},
				cmAccounts:      mockCMAccounts,
			},
			args: args{
				msg: &types.Message{
					Type:     generated.PingServiceV1Request,
					Metadata: metadata.Metadata{Recipient: anotherUserID, RequestID: requestID},
				},
			},
			prepare: func() {
				mockCMAccounts.EXPECT().GetChequeOperators(gomock.Any(), gomock.Any()).Return([]common.Address{{}}, nil)
				mockCMAccounts.EXPECT().GetServiceFee(gomock.Any(), gomock.Any(), gomock.Any()).Return(big.NewInt(1), nil)
				mockCMAccounts.EXPECT().IsBotAllowed(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockChequeHandler.EXPECT().IssueCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(&cheques.SignedCheque{}, nil)
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
				common.Address{},
				common.Address{},
				common.Address{},
				tt.fields.serviceRegistry,
				tt.fields.responseHandler,
				tt.fields.chequeHandler,
				tt.fields.compressor,
				tt.fields.cmAccounts,
			)
			if tt.prepare != nil {
				tt.prepare()
			}
			if tt.writeResponseToChannel != nil {
				go tt.writeResponseToChannel(p.(*messageProcessor))
			}
			got, err := p.SendRequestMessage(context.Background(), tt.args.msg)

			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

var _ rpc.Service = (*dummyService)(nil)

type dummyService struct{}

func (d dummyService) Call(context.Context, protoreflect.ProtoMessage, ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	return nil, "", nil
}

func (d dummyService) Name() string {
	return "dummy"
}

func TestStart(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockMessenger := NewMockMessenger(mockCtrl)
	mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(nil)

	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockServiceRegistry.EXPECT().GetService(gomock.Any()).AnyTimes().Return(dummyService{}, true)

	mockCMAccounts := cmaccounts.NewMockService(mockCtrl)
	mockCMAccounts.EXPECT().GetServiceFee(gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(big.NewInt(1), nil)

	mockChequeHandler := chequehandler.NewMockChequeHandler(mockCtrl)
	mockChequeHandler.EXPECT().VerifyCheque(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(nil)

	t.Run("start processor and accept messages", func(*testing.T) {
		ch := make(chan types.Message, 5)
		// incoming messages
		{
			// msg without sender
			ch <- types.Message{Metadata: metadata.Metadata{}}
			// msg with sender == userID
			ch <- types.Message{Metadata: metadata.Metadata{}, Sender: userID}
			// msg with sender == userID but without valid msgType
			ch <- types.Message{Metadata: metadata.Metadata{Sender: anotherUserID, Cheques: []cheques.SignedCheque{dummyCheque}}}
			// msg with sender == userID and valid msgType
			ch <- types.Message{
				Type:     generated.PingServiceV1Request,
				Metadata: metadata.Metadata{Sender: anotherUserID, Cheques: []cheques.SignedCheque{dummyCheque}},
			}
			// 2nd msg with sender == userID and valid msgType
			ch <- types.Message{
				Type:     generated.AccommodationProductInfoServiceV2Request,
				Metadata: metadata.Metadata{Sender: anotherUserID, Cheques: []cheques.SignedCheque{dummyCheque}},
			}
		}
		// mocks
		mockMessenger.EXPECT().Inbound().AnyTimes().Return(ch)

		ctx, cancel := context.WithCancel(context.Background())
		p := NewMessageProcessor(
			mockMessenger,
			zap.NewNop().Sugar(),
			time.Duration(0),
			userID,
			common.Address{},
			common.Address{},
			common.Address{},
			mockServiceRegistry,
			NoopResponseHandler{},
			mockChequeHandler,
			&noopCompressor{},
			mockCMAccounts,
		)
		go p.Start(ctx)

		time.Sleep(1 * time.Second)
		cancel()
	})
}

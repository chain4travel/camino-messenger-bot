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
	cmaccountscache "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts_cache"
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
	defer mockCtrl.Finish()
	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockService := rpc.NewMockService(mockCtrl)
	mockMessenger := NewMockMessenger(mockCtrl)

	type fields struct {
		messenger             Messenger
		serviceRegistry       ServiceRegistry
		responseHandler       ResponseHandler
		identificationHandler IdentificationHandler
		chequeHandler         chequehandler.ChequeHandler
		compressor            compression.Compressor[*types.Message, [][]byte]
		cmAccountsCache       cmaccountscache.CMAccountsCache
	}
	type args struct {
		msg *types.Message
	}
	tests := map[string]struct {
		fields  fields
		args    args
		prepare func(*processor)
		err     error
		assert  func(*testing.T, *processor)
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
			prepare: func(*processor) {
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
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         chequehandler.NoopChequeHandler{},
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccountsCache:       &cmaccountscache.NoopCMAccountsCache{},
			},
			prepare: func(*processor) {
				mockService.EXPECT().Name().Return("dummy")
				mockService.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, generated.PingServiceV1Response, nil)
				mockServiceRegistry.EXPECT().GetService(gomock.Any()).Times(1).Return(mockService, true)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(errSomeError)
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
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         chequehandler.NoopChequeHandler{},
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccountsCache:       &cmaccountscache.NoopCMAccountsCache{},
			},
			prepare: func(*processor) {
				mockService.EXPECT().Name().Return("dummy")
				mockService.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, generated.PingServiceV1Response, nil)
				mockServiceRegistry.EXPECT().GetService(gomock.Any()).Times(1).Return(mockService, true)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
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
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         chequehandler.NoopChequeHandler{},
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccountsCache:       &cmaccountscache.NoopCMAccountsCache{},
			},
			prepare: func(p *processor) {
				p.responseChannels[requestID] = make(chan *types.Message, 1)
			},
			args: args{
				msg: &responseMessage,
			},
			assert: func(t *testing.T, p *processor) {
				msgReceived := <-p.responseChannels[requestID]
				require.Equal(t, responseMessage, *msgReceived)
			},
		},
	}
	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			p := NewProcessor(
				tt.fields.messenger,
				zap.NewNop().Sugar(),
				time.Duration(0),
				userID,
				common.Address{},
				common.Address{},
				common.Address{},
				tt.fields.serviceRegistry,
				tt.fields.responseHandler,
				tt.fields.identificationHandler,
				tt.fields.chequeHandler,
				tt.fields.compressor,
				tt.fields.cmAccountsCache,
			)
			if tt.prepare != nil {
				tt.prepare(p.(*processor))
			}
			err := p.ProcessInbound(tt.args.msg)
			require.ErrorIs(t, err, tt.err)

			if tt.assert != nil {
				tt.assert(t, p.(*processor))
			}
		})
	}
}

func TestProcessOutbound(t *testing.T) {
	productListResponse := &types.Message{
		Type:     generated.PingServiceV1Response,
		Metadata: metadata.Metadata{RequestID: requestID},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockMessenger := NewMockMessenger(mockCtrl)

	type fields struct {
		responseTimeout       time.Duration
		messenger             Messenger
		serviceRegistry       ServiceRegistry
		responseHandler       ResponseHandler
		identificationHandler IdentificationHandler
		chequeHandler         chequehandler.ChequeHandler
		compressor            compression.Compressor[*types.Message, [][]byte]
		cmAccountsCache       cmaccountscache.CMAccountsCache
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
		writeResponseToChannel func(*processor)
	}{
		"err: non-request outbound message": {
			fields: fields{
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         chequehandler.NoopChequeHandler{},
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccountsCache:       &cmaccountscache.NoopCMAccountsCache{},
			},
			args: args{
				msg: &types.Message{Type: generated.PingServiceV1Response},
			},
			err: ErrOnlyRequestMessagesAllowed,
		},
		"err: missing recipient": {
			fields: fields{
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         chequehandler.NoopChequeHandler{},
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccountsCache:       &cmaccountscache.NoopCMAccountsCache{},
			},
			args: args{
				msg: &types.Message{Type: generated.PingServiceV1Request},
			},
			err: ErrMissingRecipient,
		},
		"err: awaiting-response-timeout exceeded": {
			fields: fields{
				responseTimeout:       10 * time.Millisecond, // 10ms
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         chequehandler.NoopChequeHandler{},
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccountsCache:       &cmaccountscache.NoopCMAccountsCache{},
			},
			args: args{
				msg: &types.Message{
					Type:     generated.PingServiceV1Request,
					Metadata: metadata.Metadata{Recipient: anotherUserID},
				},
			},
			prepare: func() {
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
			},
			err: ErrExceededResponseTimeout,
		},
		"err: while sending request": {
			fields: fields{
				responseTimeout:       100 * time.Millisecond, // 100ms
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         chequehandler.NoopChequeHandler{},
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccountsCache:       &cmaccountscache.NoopCMAccountsCache{},
			},
			args: args{
				msg: &types.Message{
					Type:     generated.PingServiceV1Request,
					Metadata: metadata.Metadata{Recipient: anotherUserID},
				},
			},
			prepare: func() {
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(errSomeError)
			},
			err: errSomeError,
		},
		"success: response before timeout": {
			fields: fields{
				responseTimeout:       500 * time.Millisecond, // long enough timeout for response to be received
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         chequehandler.NoopChequeHandler{},
				messenger:             mockMessenger,
				compressor:            &noopCompressor{},
				cmAccountsCache:       &cmaccountscache.NoopCMAccountsCache{},
			},
			args: args{
				msg: &types.Message{
					Type:     generated.PingServiceV1Request,
					Metadata: metadata.Metadata{Recipient: anotherUserID, RequestID: requestID},
				},
			},
			prepare: func() {
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
			},
			writeResponseToChannel: func(p *processor) {
				done := func() bool {
					p.mu.Lock()
					defer p.mu.Unlock()
					if _, ok := p.responseChannels[requestID]; ok {
						p.responseChannels[requestID] <- productListResponse
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
			p := NewProcessor(
				tt.fields.messenger,
				zap.NewNop().Sugar(),
				tt.fields.responseTimeout,
				userID,
				common.Address{},
				common.Address{},
				common.Address{},
				tt.fields.serviceRegistry,
				tt.fields.responseHandler,
				tt.fields.identificationHandler,
				tt.fields.chequeHandler,
				tt.fields.compressor,
				tt.fields.cmAccountsCache,
			)
			if tt.prepare != nil {
				tt.prepare()
			}
			if tt.writeResponseToChannel != nil {
				go tt.writeResponseToChannel(p.(*processor))
			}
			got, err := p.ProcessOutbound(context.Background(), tt.args.msg)

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
	defer mockCtrl.Finish()
	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockMessenger := NewMockMessenger(mockCtrl)
	mockServiceRegistry.EXPECT().GetService(gomock.Any()).AnyTimes().Return(dummyService{}, true)
	mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(nil)

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
		p := NewProcessor(
			mockMessenger,
			zap.NewNop().Sugar(),
			time.Duration(0),
			userID,
			common.Address{},
			common.Address{},
			common.Address{},
			mockServiceRegistry,
			NoopResponseHandler{},
			NoopIdentification{},
			chequehandler.NoopChequeHandler{},
			&noopCompressor{},
			&cmaccountscache.NoopCMAccountsCache{},
		)
		go p.Start(ctx)

		time.Sleep(1 * time.Second)
		cancel()
	})
}

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
	"maunium.net/go/mautrix/id"

	"go.uber.org/mock/gomock"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"

	"github.com/stretchr/testify/require"

	"github.com/chain4travel/camino-messenger-bot/config"
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
	responseMessage := Message{Type: ActivityProductListResponse, Metadata: metadata.Metadata{RequestID: requestID, Sender: anotherUserID, Cheques: []cheques.SignedCheque{}}}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockActivityProductListServiceClient := NewMockActivityProductListServiceClient(mockCtrl)
	mockMessenger := NewMockMessenger(mockCtrl)

	type fields struct {
		cfg                   config.ProcessorConfig
		evmConfig             config.EvmConfig
		messenger             Messenger
		serviceRegistry       ServiceRegistry
		responseHandler       ResponseHandler
		identificationHandler IdentificationHandler
		chequeHandler         ChequeHandler
	}
	type args struct {
		msg *Message
	}
	tests := map[string]struct {
		fields  fields
		args    args
		prepare func(p *processor)
		err     error
		assert  func(t *testing.T, p *processor)
	}{
		"err: user id not set": {
			fields: fields{
				cfg: config.ProcessorConfig{},
			},
			err: ErrUserIDNotSet,
		},
		"err: invalid message type": {
			fields: fields{
				cfg: config.ProcessorConfig{},
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
			},
			args: args{
				msg: &Message{Type: "invalid", Metadata: metadata.Metadata{Sender: anotherUserID, Cheques: []cheques.SignedCheque{}}},
			},
			err: ErrUnknownMessageCategory,
		},
		"err: unsupported request message": {
			fields: fields{
				cfg:             config.ProcessorConfig{},
				serviceRegistry: mockServiceRegistry,
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
				mockServiceRegistry.EXPECT().GetService(gomock.Any()).Return(nil, false)
			},
			args: args{
				msg: &Message{Type: ActivitySearchRequest, Metadata: metadata.Metadata{Sender: anotherUserID, Cheques: []cheques.SignedCheque{}}},
			},
			err: ErrUnsupportedService,
		},
		"ignore own outbound messages": {
			fields: fields{
				cfg: config.ProcessorConfig{},
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
			},
			args: args{
				msg: &Message{Metadata: metadata.Metadata{}, Sender: userID},
			},
			err: nil, // no error, msg will be just ignored
		},
		"err: process request message failed": {
			fields: fields{
				cfg:                   config.ProcessorConfig{},
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         NoopChequeHandler{},
				messenger:             mockMessenger,
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
				mockActivityProductListServiceClient.EXPECT().ActivityProductList(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
				mockServiceRegistry.EXPECT().GetService(gomock.Any()).Times(1).Return(activityProductListService{client: mockActivityProductListServiceClient}, true)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(errSomeError)
			},
			args: args{
				msg: &Message{Type: ActivityProductListRequest, Metadata: metadata.Metadata{Sender: anotherUserID, Cheques: []cheques.SignedCheque{dummyCheque}}},
			},
			err: errSomeError,
		},
		"success: process request message": {
			fields: fields{
				cfg:                   config.ProcessorConfig{},
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				messenger:             mockMessenger,
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
				mockActivityProductListServiceClient.EXPECT().ActivityProductList(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
				mockServiceRegistry.EXPECT().GetService(gomock.Any()).Times(1).Return(activityProductListService{client: mockActivityProductListServiceClient}, true)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
			},
			args: args{
				msg: &Message{Type: ActivityProductListRequest, Metadata: metadata.Metadata{Sender: anotherUserID, Cheques: []cheques.SignedCheque{dummyCheque}}},
			},
		},
		"success: process response message": {
			fields: fields{
				cfg:                   config.ProcessorConfig{},
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				messenger:             mockMessenger,
			},
			prepare: func(p *processor) {
				p.responseChannels[requestID] = make(chan *Message, 1)
				p.SetUserID(userID)
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
				tt.fields.cfg,
				tt.fields.evmConfig,
				tt.fields.serviceRegistry,
				tt.fields.responseHandler,
				tt.fields.identificationHandler,
				tt.fields.chequeHandler,
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
	productListResponse := &Message{Type: ActivityProductListResponse, Metadata: metadata.Metadata{RequestID: requestID}}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockMessenger := NewMockMessenger(mockCtrl)

	type fields struct {
		cfg                   config.ProcessorConfig
		evmConfig             config.EvmConfig
		messenger             Messenger
		serviceRegistry       ServiceRegistry
		responseHandler       ResponseHandler
		identificationHandler IdentificationHandler
		chequeHandler         ChequeHandler
	}
	type args struct {
		msg *Message
	}
	tests := map[string]struct {
		fields                 fields
		args                   args
		want                   *Message
		err                    error
		prepare                func(p *processor)
		writeResponseToChannel func(p *processor)
	}{
		"err: non-request outbound message": {
			fields: fields{
				cfg:                   config.ProcessorConfig{},
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         NoopChequeHandler{},
				messenger:             mockMessenger,
			},
			args: args{
				msg: &Message{Type: ActivityProductListResponse},
			},
			err: ErrOnlyRequestMessagesAllowed,
		},
		"err: missing recipient": {
			fields: fields{
				cfg:                   config.ProcessorConfig{},
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         NoopChequeHandler{},
				messenger:             mockMessenger,
			},
			args: args{
				msg: &Message{Type: ActivityProductListRequest},
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
			},
			err: ErrMissingRecipient,
		},
		"err: awaiting-response-timeout exceeded": {
			fields: fields{
				cfg:                   config.ProcessorConfig{Timeout: 10}, // 10ms
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         NoopChequeHandler{},
				messenger:             mockMessenger,
			},
			args: args{
				msg: &Message{Type: ActivityProductListRequest, Metadata: metadata.Metadata{Recipient: anotherUserID}},
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
			},
			err: ErrExceededResponseTimeout,
		},
		"err: while sending request": {
			fields: fields{
				cfg:                   config.ProcessorConfig{Timeout: 100}, // 10ms
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         NoopChequeHandler{},
				messenger:             mockMessenger,
			},
			args: args{
				msg: &Message{Type: ActivityProductListRequest, Metadata: metadata.Metadata{Recipient: anotherUserID}},
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(errSomeError)
			},
			err: errSomeError,
		},
		"success: response before timeout": {
			fields: fields{
				cfg:                   config.ProcessorConfig{Timeout: 500}, // long enough timeout for response to be received
				serviceRegistry:       mockServiceRegistry,
				responseHandler:       NoopResponseHandler{},
				identificationHandler: NoopIdentification{},
				chequeHandler:         NoopChequeHandler{},
				messenger:             mockMessenger,
			},
			args: args{
				msg: &Message{Type: ActivityProductListRequest, Metadata: metadata.Metadata{Recipient: anotherUserID, RequestID: requestID}},
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
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
			p := NewProcessor(tt.fields.messenger, zap.NewNop().Sugar(), tt.fields.cfg, tt.fields.evmConfig, tt.fields.serviceRegistry, tt.fields.responseHandler, tt.fields.identificationHandler, tt.fields.chequeHandler)
			if tt.prepare != nil {
				tt.prepare(p.(*processor))
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

type dummyService struct{}

func (d dummyService) Call(context.Context, *RequestContent, ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	return &ResponseContent{}, "", nil
}

func TestStart(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockMessenger := NewMockMessenger(mockCtrl)
	mockServiceRegistry.EXPECT().GetService(gomock.Any()).AnyTimes().Return(dummyService{}, true)
	mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(nil)

	t.Run("start processor and accept messages", func(*testing.T) {
		cfg := config.ProcessorConfig{}
		evmConfig := config.EvmConfig{}
		serviceRegistry := mockServiceRegistry
		responseHandler := NoopResponseHandler{}
		identificationHandler := NoopIdentification{}
		chequeHandler := NoopChequeHandler{}
		messenger := mockMessenger

		ch := make(chan Message, 5)
		// incoming messages
		{
			// msg without sender
			ch <- Message{Metadata: metadata.Metadata{}}
			// msg with sender == userID
			ch <- Message{Metadata: metadata.Metadata{}, Sender: userID}
			// msg with sender == userID but without valid msgType
			ch <- Message{Metadata: metadata.Metadata{Sender: anotherUserID, Cheques: []cheques.SignedCheque{dummyCheque}}}
			// msg with sender == userID and valid msgType
			ch <- Message{
				Type:     ActivityProductListRequest,
				Metadata: metadata.Metadata{Sender: anotherUserID, Cheques: []cheques.SignedCheque{dummyCheque}},
			}
			// 2nd msg with sender == userID and valid msgType
			ch <- Message{
				Type:     ActivitySearchRequest,
				Metadata: metadata.Metadata{Sender: anotherUserID, Cheques: []cheques.SignedCheque{dummyCheque}},
			}
		}
		// mocks
		mockMessenger.EXPECT().Inbound().AnyTimes().Return(ch)

		ctx, cancel := context.WithCancel(context.Background())
		p := NewProcessor(messenger, zap.NewNop().Sugar(), cfg, evmConfig, serviceRegistry, responseHandler, identificationHandler, chequeHandler)
		p.SetUserID(userID)
		go p.Start(ctx)

		time.Sleep(1 * time.Second)
		cancel()
	})
}

/*
 * Copyright (C) 2024, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"

	"github.com/stretchr/testify/require"

	"github.com/chain4travel/camino-messenger-bot/config"
	"go.uber.org/zap"
)

func TestProcessInbound(t *testing.T) {
	userID := "userID"
	anotherUserID := "anotherUserID"
	someError := errors.New("some error")
	requestID := "requestID"
	responseMessage := Message{Type: ActivityProductListResponse, Metadata: metadata.Metadata{RequestID: requestID, Sender: anotherUserID}}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockActivityProductListServiceClient := NewMockActivityProductListServiceClient(mockCtrl)
	mockMessenger := NewMockMessenger(mockCtrl)

	type fields struct {
		cfg             config.ProcessorConfig
		messenger       Messenger
		serviceRegistry ServiceRegistry
		responseHandler ResponseHandler
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
				msg: &Message{Type: "invalid", Metadata: metadata.Metadata{Sender: anotherUserID}},
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
				msg: &Message{Type: ActivitySearchRequest, Metadata: metadata.Metadata{Sender: anotherUserID}},
			},
			err: ErrUnsupportedRequestType,
		},
		"ignore own outbound messages": {
			fields: fields{
				cfg: config.ProcessorConfig{},
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
			},
			args: args{
				msg: &Message{Metadata: metadata.Metadata{Sender: userID}},
			},
			err: nil, // no error, msg will be just ignored
		},
		"err: process request message failed": {
			fields: fields{
				cfg:             config.ProcessorConfig{},
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				messenger:       mockMessenger,
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
				mockActivityProductListServiceClient.EXPECT().ActivityProductList(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
				mockServiceRegistry.EXPECT().GetService(gomock.Any()).Times(1).Return(activityProductListService{client: mockActivityProductListServiceClient}, true)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any()).Times(1).Return(someError)
			},
			args: args{
				msg: &Message{Type: ActivityProductListRequest, Metadata: metadata.Metadata{Sender: anotherUserID}},
			},
			err: someError,
		},
		"success: process request message": {
			fields: fields{
				cfg:             config.ProcessorConfig{},
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				messenger:       mockMessenger,
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
				mockActivityProductListServiceClient.EXPECT().ActivityProductList(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
				mockServiceRegistry.EXPECT().GetService(gomock.Any()).Times(1).Return(activityProductListService{client: mockActivityProductListServiceClient}, true)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any()).Times(1).Return(nil)
			},
			args: args{
				msg: &Message{Type: ActivityProductListRequest, Metadata: metadata.Metadata{Sender: anotherUserID}},
			},
		},
		"success: process response message": {
			fields: fields{
				cfg:             config.ProcessorConfig{},
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				messenger:       mockMessenger,
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
			p := NewProcessor(tt.fields.messenger, zap.NewNop().Sugar(), tt.fields.cfg, tt.fields.serviceRegistry, tt.fields.responseHandler)
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
	requestID := "requestID"
	userID := "userID"
	anotherUserID := "anotherUserID"
	someError := errors.New("some error")
	productListResponse := &Message{Type: ActivityProductListResponse, Metadata: metadata.Metadata{RequestID: requestID}}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockServiceRegistry := NewMockServiceRegistry(mockCtrl)
	mockMessenger := NewMockMessenger(mockCtrl)

	type fields struct {
		cfg             config.ProcessorConfig
		messenger       Messenger
		serviceRegistry ServiceRegistry
		responseHandler ResponseHandler
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
				cfg:             config.ProcessorConfig{},
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				messenger:       mockMessenger,
			},
			args: args{
				msg: &Message{Type: ActivityProductListResponse},
			},
			err: ErrOnlyRequestMessagesAllowed,
		},
		"err: missing recipient": {
			fields: fields{
				cfg:             config.ProcessorConfig{},
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				messenger:       mockMessenger,
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
				cfg:             config.ProcessorConfig{Timeout: 10}, // 10ms
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				messenger:       mockMessenger,
			},
			args: args{
				msg: &Message{Type: ActivityProductListRequest, Metadata: metadata.Metadata{Recipient: anotherUserID}},
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any()).Times(1).Return(nil)
			},
			err: ErrExceededResponseTimeout,
		},
		"err: while sending request": {
			fields: fields{
				cfg:             config.ProcessorConfig{Timeout: 100}, // 10ms
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				messenger:       mockMessenger,
			},
			args: args{
				msg: &Message{Type: ActivityProductListRequest, Metadata: metadata.Metadata{Recipient: anotherUserID}},
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any()).Times(1).Return(someError)
			},
			err: someError,
		},
		"success: response before timeout": {
			fields: fields{
				cfg:             config.ProcessorConfig{Timeout: 500}, // long enough timeout for response to be received
				serviceRegistry: mockServiceRegistry,
				responseHandler: NoopResponseHandler{},
				messenger:       mockMessenger,
			},
			args: args{
				msg: &Message{Type: ActivityProductListRequest, Metadata: metadata.Metadata{Recipient: anotherUserID, RequestID: requestID}},
			},
			prepare: func(p *processor) {
				p.SetUserID(userID)
				mockMessenger.EXPECT().SendAsync(gomock.Any(), gomock.Any()).Times(1).Return(nil)
			},
			writeResponseToChannel: func(p *processor) {
				done := func() bool {
					// wait until the response channel is created
					p.mu.Lock()
					defer p.mu.Unlock()
					if _, ok := p.responseChannels[requestID]; ok {
						p.responseChannels[requestID] <- productListResponse
						return true
					}
					return false
				}
				for !done() {
				}
			},
			want: productListResponse,
		},
	}

	for tc, tt := range tests {
		t.Run(tc, func(t *testing.T) {
			p := NewProcessor(tt.fields.messenger, zap.NewNop().Sugar(), tt.fields.cfg, tt.fields.serviceRegistry, tt.fields.responseHandler)
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

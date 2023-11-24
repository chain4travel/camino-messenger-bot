/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1alpha1/pingv1alpha1grpc"
	"context"
	"errors"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha1/accommodationv1alpha1grpc"
	"google.golang.org/grpc"
)

var (
	_                     Service = (*accommodationService)(nil)
	ErrInvalidMessageType         = errors.New("invalid message type")
)

type Service interface {
	call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error)
}

type accommodationService struct {
	client *accommodationv1alpha1grpc.AccommodationSearchServiceClient
}

func (a accommodationService) call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
	if &request.AccommodationSearchRequest == nil {
		return ResponseContent{}, "", ErrInvalidMessageType
	}
	response, err := (*a.client).AccommodationSearch(ctx, &request.AccommodationSearchRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.AccommodationSearchResponse = *response // otherwise 	nil pointer dereference
	}
	return responseContent, AccommodationSearchResponse, err
}

type pingService struct {
	client *pingv1alpha1grpc.PingServiceClient
}

func (s pingService) call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
	if &request.PingRequest == nil {
		return ResponseContent{}, "", ErrInvalidMessageType
	}
	response, err := (*s.client).Ping(ctx, &request.PingRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.PingResponse = *response // otherwise 	nil pointer dereference
	}
	return responseContent, PingResponse, err
}

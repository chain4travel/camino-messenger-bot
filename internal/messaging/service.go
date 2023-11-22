/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"
	"errors"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha1/accommodationv1alpha1grpc"
	"google.golang.org/grpc"
)

var (
	_                 Service = (*accommodationService)(nil)
	ErrInvalidRequest         = errors.New("invalid request")
)

type Service interface {
	call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error)
}

type accommodationService struct {
	client *accommodationv1alpha1grpc.AccommodationSearchServiceClient
}

func (a accommodationService) call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
	if &request.AccommodationSearchRequest == nil {
		return ResponseContent{}, "", ErrInvalidRequest
	}
	response, err := (*a.client).AccommodationSearch(ctx, &request.AccommodationSearchRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.AccommodationSearchResponse = *response // otherwise 	nil pointer dereference
	}
	return responseContent, AccommodationSearchResponse, err
}

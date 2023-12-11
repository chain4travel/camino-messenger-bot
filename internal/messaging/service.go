/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1alpha1/activityv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/network/v1alpha1/networkv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v1alpha1/partnerv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1alpha1/pingv1alpha1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1alpha1/transportv1alpha1grpc"
	"context"
	"errors"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha1/accommodationv1alpha1grpc"
	"google.golang.org/grpc"
)

var (
	_                     Service = (*activityService)(nil)
	_                     Service = (*accommodationService)(nil)
	_                     Service = (*networkService)(nil)
	_                     Service = (*partnerService)(nil)
	_                     Service = (*pingService)(nil)
	_                     Service = (*transportService)(nil)
	ErrInvalidMessageType         = errors.New("invalid message type")
)

type Service interface {
	call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error)
}

type activityService struct {
	client *activityv1alpha1grpc.ActivitySearchServiceClient
}
type accommodationService struct {
	client *accommodationv1alpha1grpc.AccommodationSearchServiceClient
}
type networkService struct {
	client *networkv1alpha1grpc.GetNetworkFeeServiceClient
}
type partnerService struct {
	client *partnerv1alpha1grpc.GetPartnerConfigurationServiceClient
}
type pingService struct {
	client *pingv1alpha1grpc.PingServiceClient
}
type transportService struct {
	client *transportv1alpha1grpc.TransportSearchServiceClient
}

func (s activityService) call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
	if &request.ActivitySearchRequest == nil {
		return ResponseContent{}, "", ErrInvalidMessageType
	}
	response, err := (*s.client).ActivitySearch(ctx, &request.ActivitySearchRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.ActivitySearchResponse = *response // otherwise nil pointer dereference
	}
	return responseContent, ActivitySearchResponse, err
}

func (s accommodationService) call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
	if &request.AccommodationSearchRequest == nil {
		return ResponseContent{}, "", ErrInvalidMessageType
	}
	response, err := (*s.client).AccommodationSearch(ctx, &request.AccommodationSearchRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.AccommodationSearchResponse = *response // otherwise nil pointer dereference
	}
	return responseContent, AccommodationSearchResponse, err
}

func (s networkService) call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
	if &request.GetNetworkFeeRequest == nil {
		return ResponseContent{}, "", ErrInvalidMessageType
	}
	response, err := (*s.client).GetNetworkFee(ctx, &request.GetNetworkFeeRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.GetNetworkFeeResponse = *response // otherwise 	nil pointer dereference
	}
	return responseContent, GetNetworkFeeResponse, err
}

func (s partnerService) call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
	if &request.GetPartnerConfigurationRequest == nil {
		return ResponseContent{}, "", ErrInvalidMessageType
	}
	response, err := (*s.client).GetPartnerConfiguration(ctx, &request.GetPartnerConfigurationRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.GetPartnerConfigurationResponse = *response // otherwise 	nil pointer dereference
	}
	return responseContent, GetPartnerConfigurationResponse, err
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

func (s transportService) call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
	if &request.TransportSearchRequest == nil {
		return ResponseContent{}, "", ErrInvalidMessageType
	}
	response, err := (*s.client).TransportSearch(ctx, &request.TransportSearchRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.TransportSearchResponse = *response // otherwise 	nil pointer dereference
	}
	return responseContent, TransportSearchResponse, err
}

/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1alpha/activityv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1alpha/pingv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1alpha/transportv1alphagrpc"
	networkv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1alpha"
	partnerv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1alpha"
	"context"
	"errors"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha/accommodationv1alphagrpc"
	"google.golang.org/grpc"
)

var (
	_                     Service = (*activityService)(nil)
	_                     Service = (*accommodationProductInfoService)(nil)
	_                     Service = (*accommodationProductListService)(nil)
	_                     Service = (*accommodationService)(nil)
	_                     Service = (*networkService)(nil)
	_                     Service = (*partnerService)(nil)
	_                     Service = (*pingService)(nil)
	_                     Service = (*transportService)(nil)
	ErrInvalidMessageType         = errors.New("invalid message type")
)

type Service interface {
	Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error)
}

type activityService struct {
	client *activityv1alphagrpc.ActivitySearchServiceClient
}
type accommodationProductInfoService struct {
	client *accommodationv1alphagrpc.AccommodationProductInfoServiceClient
}
type accommodationProductListService struct {
	client *accommodationv1alphagrpc.AccommodationProductListServiceClient
}

type accommodationService struct {
	client *accommodationv1alphagrpc.AccommodationSearchServiceClient
}
type networkService struct {
}
type partnerService struct {
}
type pingService struct {
	client *pingv1alphagrpc.PingServiceClient
}
type transportService struct {
	client *transportv1alphagrpc.TransportSearchServiceClient
}

func (a accommodationProductInfoService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
	if &request.AccommodationProductInfoRequest == nil {
		return ResponseContent{}, "", ErrInvalidMessageType
	}
	response, err := (*a.client).AccommodationProductInfo(ctx, &request.AccommodationProductInfoRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.AccommodationProductInfoResponse = *response // otherwise nil pointer dereference
	}
	return responseContent, AccommodationProductInfoResponse, err
}
func (a accommodationProductListService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
	if &request.AccommodationProductListRequest == nil {
		return ResponseContent{}, "", ErrInvalidMessageType
	}
	response, err := (*a.client).AccommodationProductList(ctx, &request.AccommodationProductListRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.AccommodationProductListResponse = *response // otherwise nil pointer dereference
	}
	return responseContent, AccommodationProductListResponse, err
}
func (s activityService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
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

func (s accommodationService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
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

func (s networkService) Call(_ context.Context, request *RequestContent, _ ...grpc.CallOption) (ResponseContent, MessageType, error) {
	if &request.GetNetworkFeeRequest == nil {
		return ResponseContent{}, "", ErrInvalidMessageType
	}

	//TODO implement
	response, err := &networkv1alpha.GetNetworkFeeResponse{
		NetworkFee: &networkv1alpha.NetworkFee{Amount: 100000},
	}, (error)(nil)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.GetNetworkFeeResponse = *response // otherwise 	nil pointer dereference
	}
	return responseContent, GetNetworkFeeResponse, err
}

func (s partnerService) Call(_ context.Context, request *RequestContent, _ ...grpc.CallOption) (ResponseContent, MessageType, error) {
	if &request.GetPartnerConfigurationRequest == nil {
		return ResponseContent{}, "", ErrInvalidMessageType
	}

	//TODO implement
	response, err := &partnerv1alpha.GetPartnerConfigurationResponse{
		PartnerConfiguration: nil,
		CurrentBlockHeight:   0,
	}, (error)(nil)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.GetPartnerConfigurationResponse = *response // otherwise 	nil pointer dereference
	}
	return responseContent, GetPartnerConfigurationResponse, err
}

func (s pingService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
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

func (s transportService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (ResponseContent, MessageType, error) {
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

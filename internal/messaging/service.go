/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"
	"fmt"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v1/infov1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/insurance/v1/insurancev1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v1/seat_mapv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1/transportv1grpc"

	networkv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1"
	partnerv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	"google.golang.org/grpc"
)

var (
	_ Service = (*activityProductInfoService)(nil)
	_ Service = (*activityProductListService)(nil)
	_ Service = (*activityService)(nil)
	_ Service = (*accommodationProductInfoService)(nil)
	_ Service = (*accommodationProductListService)(nil)
	_ Service = (*accommodationSearchService)(nil)
	_ Service = (*mintService)(nil)
	_ Service = (*validationService)(nil)
	_ Service = (*networkService)(nil)
	_ Service = (*partnerService)(nil)
	_ Service = (*pingService)(nil)
	_ Service = (*transportService)(nil)
	_ Service = (*seatMapService)(nil)
	_ Service = (*seatMapAvailabilityService)(nil)
	_ Service = (*countryEntryRequirementsService)(nil)
	_ Service = (*insuranceProductInfoService)(nil)
	_ Service = (*insuranceProductListService)(nil)
	_ Service = (*insuranceSearchService)(nil)
)

type Service interface {
	Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error)
}
type activityProductInfoService struct {
	client *activityv1grpc.ActivityProductInfoServiceClient
}

func (s activityProductInfoService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*s.client).ActivityProductInfo(ctx, request.ActivityProductInfoRequest, opts...)
	ResponseContent := ResponseContent{}
	if err == nil {
		ResponseContent.ActivityProductInfoResponse = response
	}
	return &ResponseContent, ActivityProductInfoResponse, err
}

type activityProductListService struct {
	client activityv1grpc.ActivityProductListServiceClient
}

func (a activityProductListService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := a.client.ActivityProductList(ctx, request.ActivityProductListRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.ActivityProductListResponse = response // otherwise nil pointer dereference
	}
	return &responseContent, ActivityProductListResponse, err
}

type activityService struct {
	client *activityv1grpc.ActivitySearchServiceClient
}

func (s activityService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*s.client).ActivitySearch(ctx, request.ActivitySearchRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.ActivitySearchResponse = response // otherwise nil pointer dereference
	}
	return &responseContent, ActivitySearchResponse, err
}

type accommodationProductInfoService struct {
	client *accommodationv1grpc.AccommodationProductInfoServiceClient
}

func (a accommodationProductInfoService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*a.client).AccommodationProductInfo(ctx, request.AccommodationProductInfoRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.AccommodationProductInfoResponse = response // otherwise nil pointer dereference
	}
	return &responseContent, AccommodationProductInfoResponse, err
}

type accommodationProductListService struct {
	client *accommodationv1grpc.AccommodationProductListServiceClient
}

func (a accommodationProductListService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*a.client).AccommodationProductList(ctx, request.AccommodationProductListRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.AccommodationProductListResponse = response // otherwise nil pointer dereference
	}
	return &responseContent, AccommodationProductListResponse, err
}

type accommodationSearchService struct {
	client *accommodationv1grpc.AccommodationSearchServiceClient
}

func (s accommodationSearchService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*s.client).AccommodationSearch(ctx, request.AccommodationSearchRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.AccommodationSearchResponse = response // otherwise nil pointer dereference
	}
	return &responseContent, AccommodationSearchResponse, err
}

type mintService struct {
	client *bookv1grpc.MintServiceClient
}

func (m mintService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*m.client).Mint(ctx, request.MintRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.MintResponse = response // otherwise nil pointer dereference
	}
	return &responseContent, MintResponse, err
}

type validationService struct {
	client *bookv1grpc.ValidationServiceClient
}

func (v validationService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*v.client).Validation(ctx, request.ValidationRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.ValidationResponse = response // otherwise nil pointer dereference
	}
	return &responseContent, ValidationResponse, err
}

type networkService struct{}

func (s networkService) Call(_ context.Context, _ *RequestContent, _ ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	return &ResponseContent{
		GetNetworkFeeResponse: &networkv1.GetNetworkFeeResponse{
			NetworkFee: &networkv1.NetworkFee{Amount: 100000}, // TODO implement
		},
	}, GetNetworkFeeResponse, nil
}

type partnerService struct{}

func (s partnerService) Call(_ context.Context, _ *RequestContent, _ ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	return &ResponseContent{
		GetPartnerConfigurationResponse: &partnerv1.GetPartnerConfigurationResponse{
			PartnerConfiguration: nil, // TODO implement
			CurrentBlockHeight:   0,
		},
	}, GetPartnerConfigurationResponse, nil
}

type pingService struct{}

func (s pingService) Call(ctx context.Context, request *RequestContent, _ ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		return nil, PingResponse, err
	}
	return &ResponseContent{PingResponse: &pingv1.PingResponse{
		Header:      nil,
		PingMessage: fmt.Sprintf("Ping response to [%s] with request ID: %s", request.PingMessage, md.RequestID),
		Timestamp:   nil,
	}}, PingResponse, nil
}

type transportService struct {
	client *transportv1grpc.TransportSearchServiceClient
}

func (s transportService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*s.client).TransportSearch(ctx, request.TransportSearchRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.TransportSearchResponse = response // otherwise 	nil pointer dereference
	}
	return &responseContent, TransportSearchResponse, err
}

type seatMapService struct {
	client *seat_mapv1grpc.SeatMapServiceClient
}

func (s seatMapService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*s.client).SeatMap(ctx, request.SeatMapRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.SeatMapResponse = response
	}
	return &responseContent, SeatMapResponse, err
}

type seatMapAvailabilityService struct {
	client *seat_mapv1grpc.SeatMapAvailabilityServiceClient
}

func (s seatMapAvailabilityService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*s.client).SeatMapAvailability(ctx, request.SeatMapAvailabilityRequest, opts...)
	responseContent := ResponseContent{}
	if err == nil {
		responseContent.SeatMapAvailabilityResponse = response
	}

	return &responseContent, SeatMapAvailabilityResponse, err
}

type countryEntryRequirementsService struct {
	client *infov1grpc.CountryEntryRequirementsServiceClient
}

func (s countryEntryRequirementsService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*s.client).CountryEntryRequirements(ctx, request.CountryEntryRequirementsRequest, opts...)
	ResponseContent := ResponseContent{}
	if err == nil {
		ResponseContent.CountryEntryRequirementsResponse = response
	}

	return &ResponseContent, CountryEntryRequirementsResponse, err
}

type insuranceProductInfoService struct {
	client *insurancev1grpc.InsuranceProductInfoServiceClient
}

func (s insuranceProductInfoService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*s.client).InsuranceProductInfo(ctx, request.InsuranceProductInfoRequest, opts...)
	ResponseContent := ResponseContent{}
	if err == nil {
		ResponseContent.InsuranceProductInfoResponse = response
	}

	return &ResponseContent, InsuranceProductInfoResponse, err
}

type insuranceProductListService struct {
	client *insurancev1grpc.InsuranceProductListServiceClient
}

func (s insuranceProductListService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*s.client).InsuranceProductList(ctx, request.InsuranceProductListRequest, opts...)
	ResponseContent := ResponseContent{}
	if err == nil {
		ResponseContent.InsuranceProductListResponse = response
	}

	return &ResponseContent, InsuranceProductListResponse, err
}

type insuranceSearchService struct {
	client *insurancev1grpc.InsuranceSearchServiceClient
}

func (s insuranceSearchService) Call(ctx context.Context, request *RequestContent, opts ...grpc.CallOption) (*ResponseContent, MessageType, error) {
	response, err := (*s.client).InsuranceSearch(ctx, request.InsuranceSearchRequest, opts...)
	ResponseContent := ResponseContent{}
	if err == nil {
		ResponseContent.InsuranceSearchResponse = response
	}

	return &ResponseContent, InsuranceSearchResponse, err
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/insurance/v3/insurancev3grpc"
	insurancev3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/insurance/v3"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
)

var _ insurancev3grpc.InsuranceProductListServiceServer = (*insuranceProductListServiceV3Server)(nil)

type insuranceProductListServiceV3Server struct{}

func NewInsuranceProductListServiceServer() insurancev3grpc.InsuranceProductListServiceServer {
	return &insuranceProductListServiceV3Server{}
}

func (s *insuranceProductListServiceV3Server) InsuranceProductList(_ context.Context, req *insurancev3.InsuranceProductListRequest) (*insurancev3.InsuranceProductListResponse, error) {
	return &insurancev3.InsuranceProductListResponse{
		Response: &insurancev3.InsuranceProductListResponse_SuccessResponse{
			SuccessResponse: &insurancev3.InsuranceProductListSuccessResponse{
				Header: common.SuccessHeaderV4(),
			},
		},
	}, nil
}

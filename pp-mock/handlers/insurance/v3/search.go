// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/insurance/v3/insurancev3grpc"
	insurancev3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/insurance/v3"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
)

var _ insurancev3grpc.InsuranceSearchServiceServer = (*insuranceSearchServiceV3Server)(nil)

type insuranceSearchServiceV3Server struct{}

func NewInsuranceSearchServiceServer() insurancev3grpc.InsuranceSearchServiceServer {
	return &insuranceSearchServiceV3Server{}
}

func (s *insuranceSearchServiceV3Server) InsuranceSearch(_ context.Context, req *insurancev3.InsuranceSearchRequest) (*insurancev3.InsuranceSearchResponse, error) {
	return &insurancev3.InsuranceSearchResponse{
		Header: common.SuccessHeaderV4(),
	}, nil
}

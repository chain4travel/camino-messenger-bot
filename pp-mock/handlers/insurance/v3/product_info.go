// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/insurance/v3/insurancev3grpc"
	insurancev3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/insurance/v3"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
)

var _ insurancev3grpc.InsuranceProductInfoServiceServer = (*insuranceProductInfoServiceV3Server)(nil)

type insuranceProductInfoServiceV3Server struct{}

func NewInsuranceProductInfoServiceServer() insurancev3grpc.InsuranceProductInfoServiceServer {
	return &insuranceProductInfoServiceV3Server{}
}

func (s *insuranceProductInfoServiceV3Server) InsuranceProductInfo(_ context.Context, req *insurancev3.InsuranceProductInfoRequest) (*insurancev3.InsuranceProductInfoResponse, error) {
	return &insurancev3.InsuranceProductInfoResponse{
		Header: common.SuccessHeaderV4(),
	}, nil
}

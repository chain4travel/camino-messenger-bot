// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v3/cancellationv3grpc"
	cancellationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	typesv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v5"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
)

var _ cancellationv3grpc.CheckCancellationServiceServer = (*checkCancellationV3Server)(nil)

type checkCancellationV3Server struct{}

func NewCheckCancellationServer() cancellationv3grpc.CheckCancellationServiceServer {
	return &checkCancellationV3Server{}
}

func (s *checkCancellationV3Server) CheckCancellation(_ context.Context, _ *cancellationv3.CheckCancellationRequest) (*cancellationv3.CheckCancellationResponse, error) {
	return &cancellationv3.CheckCancellationResponse{
		Response: &cancellationv3.CheckCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv3.CheckCancellationSuccessResponse{
				Header: common.SuccessHeaderV4(),
				RefundAmount: &typesv5.Price{
					Value:    "100",
					Decimals: 0,
					Currency: &typesv4.Currency{
						Currency: &typesv4.Currency_NativeToken{},
					},
				},
			},
		},
	}, nil
}

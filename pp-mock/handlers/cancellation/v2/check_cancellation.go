// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v2

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v2/cancellationv2grpc"
	cancellationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v2"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ cancellationv2grpc.CheckCancellationServiceServer = (*checkCancellationV2Server)(nil)

type checkCancellationV2Server struct{}

func NewCheckCancellationServer() cancellationv2grpc.CheckCancellationServiceServer {
	return &checkCancellationV2Server{}
}

func (s *checkCancellationV2Server) CheckCancellation(_ context.Context, req *cancellationv2.CheckCancellationRequest) (*cancellationv2.CheckCancellationResponse, error) {
	return &cancellationv2.CheckCancellationResponse{
		Response: &cancellationv2.CheckCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv2.CheckCancellationSuccessResponse{
				Header:  common.SuccessHeaderV4(),
				TokenId: req.TokenId,
				RefundAmount: &typesv4.Price{
					Value:    common.BookingTokenPriceValue,
					Decimals: uint32(price.NativeTokenDecimals),
					Currency: &typesv4.Currency{
						Currency: &typesv4.Currency_NativeToken{},
					},
				},
				PolicyIdApplied: common.CancellationPolicyID,
				Status:          cancellationv2.CancellationCheckStatus_CANCELLATION_CHECK_STATUS_CONFIRM,
				Timestamp:       timestamppb.Now(),
			},
		},
	}, nil
}

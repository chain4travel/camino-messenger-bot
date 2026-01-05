// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v1/cancellationv1grpc"
	cancellationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ cancellationv1grpc.CheckCancellationServiceServer = (*checkCancellationV1Server)(nil)

type checkCancellationV1Server struct{}

func NewCheckCancellationServer() cancellationv1grpc.CheckCancellationServiceServer {
	return &checkCancellationV1Server{}
}

func (s *checkCancellationV1Server) CheckCancellation(_ context.Context, req *cancellationv1.CheckCancellationRequest) (*cancellationv1.CheckCancellationResponse, error) {
	return &cancellationv1.CheckCancellationResponse{
		Header:  common.SuccessHeaderV1(),
		TokenId: req.TokenId,
		RefundAmount: &typesv3.Price{
			Value:    common.BookingTokenPriceValue,
			Decimals: price.NativeTokenDecimals,
			Currency: &typesv3.Currency{
				Currency: &typesv3.Currency_NativeToken{},
			},
		},
		PolicyIdApplied: common.CancellationPolicyID,
		Status:          cancellationv1.CancellationCheckStatus_CANCELLATION_CHECK_STATUS_CONFIRM,
		Timestamp:       timestamppb.Now(),
	}, nil
}

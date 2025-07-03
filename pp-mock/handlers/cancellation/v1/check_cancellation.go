// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v1/cancellationv1grpc"
	cancellationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const PolicyID = "pp-mock-full-refund"

var _ cancellationv1grpc.CheckCancellationServiceServer = (*checkCancellationV1Server)(nil)

type checkCancellationV1Server struct {
	eventSender events.Sender
}

func NewCheckCancellationV1Server(eventSender events.Sender) cancellationv1grpc.CheckCancellationServiceServer {
	return &checkCancellationV1Server{eventSender: eventSender}
}

func (s *checkCancellationV1Server) CheckCancellation(ctx context.Context, req *cancellationv1.CheckCancellationRequest) (*cancellationv1.CheckCancellationResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request (CheckCancellation): %s", md.RequestID)

	response := &cancellationv1.CheckCancellationResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		TokenId: req.TokenId,
		RefundAmount: &typesv3.Price{
			Value:    common.BookingTokenPriceValue,
			Decimals: price.NativeTokenDecimals,
			Currency: &typesv3.Currency{
				Currency: &typesv3.Currency_NativeToken{},
			},
		},
		PolicyIdApplied: PolicyID,
		Status:          cancellationv1.CancellationCheckStatus_CANCELLATION_CHECK_STATUS_CONFIRM,
		Timestamp:       timestamppb.Now(),
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	return response, nil
}

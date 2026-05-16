// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v5

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v5/bookv5grpc"
	bookv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v5"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ bookv5grpc.MintServiceServer = (*mintV5Server)(nil)

type mintV5Server struct{}

func NewMintServiceServer() bookv5grpc.MintServiceServer {
	return &mintV5Server{}
}

func (s *mintV5Server) Mint(_ context.Context, req *bookv5.MintRequest) (*bookv5.MintResponse, error) {
	return &bookv5.MintResponse{
		Response: &bookv5.MintResponse_SuccessResponse{
			SuccessResponse: &bookv5.MintSuccessResponse{
				Header:            common.SuccessHeaderV4(),
				MintId:            &typesv4.UUID{Value: "mock-mint-id"},
				Price:             req.ExpectedPrice,
				BuyableUntil:      timestamppb.Now(),
				BookingTokenUri:   "mock-token-uri",
				Cancellable:       true,
				BookingTokenId:    1,
				MintTransactionId: &typesv4.EVMTransactionID{Hash: "0xmockminttxhash"},
			},
		},
	}, nil
}

// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v2

import (
	"context"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/config"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/handlers/state"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ bookv2grpc.MintServiceServer = (*mintServiceV2Server)(nil)

type mintServiceV2Server struct{}

func NewMintServiceServer() bookv2grpc.MintServiceServer {
	return &mintServiceV2Server{}
}

func (s *mintServiceV2Server) Mint(_ context.Context, req *bookv2.MintRequest) (*bookv2.MintResponse, error) {
	storedValidateData, ok := state.GetStore().GetValidationResult(req.ValidationId.Value)
	if !ok {
		return &bookv2.MintResponse{
			Header: common.ErrorHeaderV1("Validation not found in state"),
		}, nil
	}

	mintResponseInfoMessage := "Please note that the price given in this mint response does not reflect the verified total price of the product of '" + storedValidateData.Data.VerifiedPrice.Price + "'. The price is just a minimum value to be able to mint the product."

	response := bookv2.MintResponse{
		Header: common.SuccessHeaderWithInfoV1(mintResponseInfoMessage),
		MintId: &typesv1.UUID{Value: uuid.New().String()},
		BuyableUntil: &timestamppb.Timestamp{
			Seconds: time.Now().Add(config.BuyableUntilDefault).Unix(),
		},
		ValidationId: req.ValidationId,
		Price:        common.BookingTokenPriceV2,
	}

	state.GetStore().AddMintResult(response.MintId.Value, storedValidateData.Data.InitialSearchData.SeatMapID)

	return &response, nil
}

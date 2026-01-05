// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v4/bookv4grpc"
	bookv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/config"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/handlers/state"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ bookv4grpc.MintServiceServer = (*mintServiceV4Server)(nil)

type mintServiceV4Server struct{}

func NewMintServiceServer() bookv4grpc.MintServiceServer {
	return &mintServiceV4Server{}
}

func (s *mintServiceV4Server) Mint(_ context.Context, req *bookv4.MintRequest) (*bookv4.MintResponse, error) {
	storedValidateData, ok := state.GetStore().GetValidationResult(req.ValidationId.Value)
	if !ok {
		return &bookv4.MintResponse{
			Response: &bookv4.MintResponse_ErrorResponse{
				ErrorResponse: &bookv4.MintErrorResponse{
					Header: common.ErrorHeaderV4(typesv4.ErrorCode_ERROR_CODE_INVALID_IDENTIFIERS, "Validation not found in state"),
				},
			},
		}, nil
	}

	response := bookv4.MintResponse{
		Response: &bookv4.MintResponse_SuccessResponse{
			SuccessResponse: &bookv4.MintSuccessResponse{
				Header: common.SuccessHeaderV4(),
				MintId: &typesv4.UUID{Value: uuid.New().String()},
				BuyableUntil: &timestamppb.Timestamp{
					Seconds: time.Now().Add(config.BuyableUntilDefault).Unix(),
				},
				ValidationId:    req.ValidationId,
				Price:           common.BookingTokenPriceV4,
				Cancellable:     true,
				BookingTokenUri: "https://example.com/",
			},
		},
	}

	mintResponseInfoMessage := "Please note that the price given in this mint response does not reflect the verified total price of the product of '" + storedValidateData.Data.VerifiedPrice.Price + "'. The price is just a minimum value to be able to mint the product."
	common.AddHeaderAlertV4(response.GetSuccessResponse().Header, typesv4.AlertCode_ALERT_CODE_INFORMATIONAL, mintResponseInfoMessage)

	state.GetStore().AddMintResult(response.GetSuccessResponse().MintId.Value, storedValidateData.Data.InitialSearchData.SeatMapID)

	return &response, nil
}

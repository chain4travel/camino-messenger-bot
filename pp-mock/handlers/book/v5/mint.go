// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v5

import (
	"context"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v5/bookv5grpc"
	bookv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v5"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/config"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/state"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ bookv5grpc.MintServiceServer = (*mintV5Server)(nil)

type mintV5Server struct{}

func NewMintServiceServer() bookv5grpc.MintServiceServer {
	return &mintV5Server{}
}

func (s *mintV5Server) Mint(_ context.Context, req *bookv5.MintRequest) (*bookv5.MintResponse, error) {
	if req.ValidationId == nil || req.ValidationId.Value == "" {
		return &bookv5.MintResponse{
			Response: &bookv5.MintResponse_ErrorResponse{
				ErrorResponse: &bookv5.MintErrorResponse{
					Header: common.ErrorHeaderV4(typesv4.ErrorCode_ERROR_CODE_INVALID_IDENTIFIERS, "Validation ID is missing or invalid"),
				},
			},
		}, nil
	}

	storedValidateData, ok := state.GetStore().GetValidationResult(req.ValidationId.Value)
	if !ok {
		return &bookv5.MintResponse{
			Response: &bookv5.MintResponse_ErrorResponse{
				ErrorResponse: &bookv5.MintErrorResponse{
					Header: common.ErrorHeaderV4(typesv4.ErrorCode_ERROR_CODE_INVALID_IDENTIFIERS, "Validation not found in state"),
				},
			},
		}, nil
	}

	response := &bookv5.MintResponse{
		Response: &bookv5.MintResponse_SuccessResponse{
			SuccessResponse: &bookv5.MintSuccessResponse{
				Header:          common.SuccessHeaderV4(),
				MintId:          &typesv4.UUID{Value: uuid.New().String()},
				BuyableUntil:    timestamppb.New(time.Now().Add(config.BuyableUntilDefault)),
				ValidationId:    req.ValidationId,
				Price:           common.BookingTokenPriceV5,
				Cancellable:     true,
				BookingTokenUri: "https://example.com/",
			},
		},
	}

	mintResponseInfoMessage := "Please note that the price given in this mint response does not reflect the verified total price of the product of '" + storedValidateData.Data.VerifiedPrice.Price + "'. The price is just a minimum value to be able to mint the product."
	common.AddHeaderAlertV4(response.GetSuccessResponse().Header, typesv4.AlertCode_ALERT_CODE_INFORMATIONAL, mintResponseInfoMessage)

	state.GetStore().AddMintResult(response.GetSuccessResponse().MintId.Value, storedValidateData.Data.InitialSearchData.SeatMapID)

	return response, nil
}

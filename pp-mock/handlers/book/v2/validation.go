// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v2

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"

	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/handlers/state"
	"github.com/google/uuid"
)

var _ bookv2grpc.ValidationServiceServer = (*validationServiceV2Server)(nil)

type validationServiceV2Server struct{}

func NewValidationServiceServer() bookv2grpc.ValidationServiceServer {
	return &validationServiceV2Server{}
}

func (s *validationServiceV2Server) Validation(_ context.Context, req *bookv2.ValidationRequest) (*bookv2.ValidationResponse, error) {
	if req.ValidationObject == nil ||
		req.ValidationObject.SearchIdentifier == nil ||
		req.ValidationObject.SearchIdentifier.ResultId == 0 ||
		req.ValidationObject.SearchIdentifier.SearchId == nil {
		return &bookv2.ValidationResponse{
			Header: common.ErrorHeaderV1("Invalid validation request: missing validation object or search identifier"),
		}, nil
	}

	// Look-up the store if we actually have a search storedSearchData for the given search identifier
	// If we don't have a storedSearchData, return an error
	storedSearchData, found := state.GetStore().GetSearchResult(req.ValidationObject.SearchIdentifier.SearchId.Value)
	if !found {
		return &bookv2.ValidationResponse{
			Header: common.ErrorHeaderV1("Invalid validation request: searchId not found in state"),
		}, nil
	}

	resultIndex := int(req.ValidationObject.SearchIdentifier.ResultId - 1)
	if resultIndex < 0 || resultIndex >= len(storedSearchData.Data.Prices) {
		return &bookv2.ValidationResponse{
			Header: common.ErrorHeaderV1("Invalid validation request: resultId out of range"),
		}, nil
	}

	unifiedValidationPrice := storedSearchData.Data.Prices[resultIndex]
	validationPrice := unifiedValidationPrice.ToPriceV2()

	response := bookv2.ValidationResponse{
		Header:           common.SuccessHeaderV1(),
		ValidationId:     &typesv1.UUID{Value: uuid.New().String()},
		ValidationObject: req.ValidationObject,
		PriceDetail: &typesv2.PriceDetail{
			Price:       validationPrice,
			Description: "Validated total price",
		},
	}

	state.GetStore().AddValidationResult(response.ValidationId.Value, state.ValidationData{
		InitialSearchData: storedSearchData.Data,
		VerifiedPrice:     unifiedValidationPrice,
		JSONRequest:       req.String(),
		JSONResponse:      response.String(),
	})

	return &response, nil
}

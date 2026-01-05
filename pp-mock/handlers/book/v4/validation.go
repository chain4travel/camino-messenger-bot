// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v4/bookv4grpc"
	bookv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"

	"github.com/chain4travel/camino-messenger-bot/v12/pkg/conversion"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/handlers/state"
)

var _ bookv4grpc.ValidationServiceServer = (*validationServiceV4Server)(nil)

type validationServiceV4Server struct{}

func NewValidationServiceServer() bookv4grpc.ValidationServiceServer {
	return &validationServiceV4Server{}
}

func (s *validationServiceV4Server) Validation(_ context.Context, req *bookv4.ValidationRequest) (*bookv4.ValidationResponse, error) {
	// Look-up the store if we actually have a search storedSearchData for the given search identifier
	// If we don't have a storedSearchData, return an error
	storedSearchData, found := state.GetStore().GetSearchResult(req.ValidationObject.SearchResultIdentifier.SearchId.Value)
	if !found {
		return errValidationResp(typesv4.ErrorCode_ERROR_CODE_INVALID_IDENTIFIERS, "Invalid validation request: searchId not found in state"), nil
	}

	if req.ValidationObject.SearchResultIdentifier.ResultId >= conversion.MustIntToUInt32(len(storedSearchData.Data.Prices)) {
		return errValidationResp(typesv4.ErrorCode_ERROR_CODE_INVALID_IDENTIFIERS, "Invalid validation request: resultId out of range"), nil
	}

	unifiedValidationPrice := storedSearchData.Data.Prices[req.ValidationObject.SearchResultIdentifier.ResultId]

	resp := &bookv4.ValidationResponse{
		Response: &bookv4.ValidationResponse_SuccessResponse{
			SuccessResponse: &bookv4.ValidationSuccessResponse{
				Header:           common.SuccessHeaderV4(),
				ValidationId:     common.NewExpiringUUID(),
				ValidationObject: req.ValidationObject,
				TotalPrice: &typesv4.TotalPrice{
					Value: unifiedValidationPrice.ToPriceV4(),
				},
			},
		},
	}

	state.GetStore().AddValidationResult(resp.GetSuccessResponse().ValidationId.Id.Value, state.ValidationData{
		InitialSearchData: storedSearchData.Data,
		VerifiedPrice:     unifiedValidationPrice,
		JSONRequest:       req.String(),
		JSONResponse:      resp.String(),
	})

	return resp, nil
}

func errValidationResp(code typesv4.ErrorCode, message string) *bookv4.ValidationResponse {
	return &bookv4.ValidationResponse{
		Response: &bookv4.ValidationResponse_ErrorResponse{
			ErrorResponse: &bookv4.ValidationErrorResponse{
				Header: common.ErrorHeaderV4(code, message),
			},
		},
	}
}

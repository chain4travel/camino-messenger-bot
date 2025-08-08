// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v4/bookv4grpc"
	bookv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"

	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	"github.com/google/uuid"
)

var _ bookv4grpc.ValidationServiceServer = (*validationServiceV4Server)(nil)

type validationServiceV4Server struct{}

func NewValidationServiceServer() bookv4grpc.ValidationServiceServer {
	return &validationServiceV4Server{}
}

func (s *validationServiceV4Server) Validation(_ context.Context, req *bookv4.ValidationRequest) (*bookv4.ValidationResponse, error) {
	if req.ValidationObject == nil ||
		req.ValidationObject.SearchIdentifier == nil ||
		req.ValidationObject.SearchIdentifier.ResultId == 0 ||
		req.ValidationObject.SearchIdentifier.SearchId == nil {
		return &bookv4.ValidationResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Message: "Invalid validation request: missing validation object or search identifier",
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// Look-up the store if we actually have a search storedSearchData for the given search identifier
	// If we don't have a storedSearchData, return an error
	storedSearchData, found := state.GetStore().GetSearchResult(req.ValidationObject.SearchIdentifier.SearchId.Value)
	if !found {
		return &bookv4.ValidationResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Message: "Invalid validation request: searchId not found in state",
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	resultIndex := int(req.ValidationObject.SearchIdentifier.ResultId - 1)
	if resultIndex < 0 || resultIndex >= len(storedSearchData.Data.Prices) {
		return &bookv4.ValidationResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Message: "Invalid validation request: resultId out of range",
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	unifiedValidationPrice := storedSearchData.Data.Prices[resultIndex]
	validationPrice := unifiedValidationPrice.ToPriceV4()

	response := bookv4.ValidationResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
			Status:     typesv4.StatusType_STATUS_TYPE_SUCCESS,
		},
		ValidationId:     &typesv4.UUID{Value: uuid.New().String()},
		ValidationObject: req.ValidationObject,
		PriceDetail: &typesv4.PriceDetail{
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

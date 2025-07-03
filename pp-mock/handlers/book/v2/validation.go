// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v2

import (
	"context"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	"github.com/google/uuid"
)

// Ensure that ValidationServiceV1Server implements the ValidationServiceServer interface
var _ bookv2grpc.ValidationServiceServer = (*validationServiceV2Server)(nil)

// ValidationServiceV2Server is the server that provides Validation services.
type validationServiceV2Server struct {
	eventSender events.Sender
}

// NewValidationServiceV2Server creates a new ValidationServiceV2Server.
func NewValidationServiceV2Server(eventSender events.Sender) bookv2grpc.ValidationServiceServer {
	return &validationServiceV2Server{eventSender: eventSender}
}

// Validate handles ValidationRequest and returns a mock ValidationResponse.
func (s *validationServiceV2Server) Validation(ctx context.Context, req *bookv2.ValidationRequest) (*bookv2.ValidationResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request: %s (Validation)", md.RequestID)
	if req.ValidationObject == nil ||
		req.ValidationObject.SearchIdentifier == nil ||
		req.ValidationObject.SearchIdentifier.ResultId == 0 ||
		req.ValidationObject.SearchIdentifier.SearchId == nil {
		return &bookv2.ValidationResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Invalid validation request: missing validation object or search identifier",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// Look-up the store if we actually have a search storedSearchData for the given search identifier
	// If we don't have a storedSearchData, return an error
	storedSearchData, found := state.GetStore().GetSearchResult(req.ValidationObject.SearchIdentifier.SearchId.Value)
	if !found {
		return &bookv2.ValidationResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Invalid validation request: searchId not found in state",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	resultIndex := int(req.ValidationObject.SearchIdentifier.ResultId - 1)
	if resultIndex < 0 || resultIndex >= len(storedSearchData.Data.Prices) {
		return &bookv2.ValidationResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Invalid validation request: resultId out of range",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	unifiedValidationPrice := storedSearchData.Data.Prices[resultIndex]
	validationPrice := unifiedValidationPrice.ToPriceV2()

	response := bookv2.ValidationResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		ValidationId:     &typesv1.UUID{Value: uuid.New().String()},
		ValidationObject: req.ValidationObject,
		PriceDetail: &typesv2.PriceDetail{
			Price:       validationPrice,
			Description: "Validated total price",
		},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	state.GetStore().AddValidationResult(response.ValidationId.Value, state.ValidationData{
		InitialSearchData: storedSearchData.Data,
		VerifiedPrice:     unifiedValidationPrice,
		JSONRequest:       req.String(),
		JSONResponse:      response.String(),
	})

	return &response, nil
}

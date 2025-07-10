// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	"github.com/google/uuid"
)

// Ensure that ValidationServiceV1Server implements the ValidationServiceServer interface
var _ bookv1grpc.ValidationServiceServer = (*validationServiceV1Server)(nil)

// ValidationServiceV1Server is the server that provides Validation services.
type validationServiceV1Server struct {
	eventSender events.Sender
}

// NewValidationServiceV1Server creates a new ValidationServiceV1Server.
func NewValidationServiceV1Server(eventSender events.Sender) bookv1grpc.ValidationServiceServer {
	return &validationServiceV1Server{eventSender: eventSender}
}

// Validate handles ValidationRequest and returns a mock ValidationResponse.
func (s *validationServiceV1Server) Validation(ctx context.Context, req *bookv1.ValidationRequest) (*bookv1.ValidationResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request: %s (Validation)", md.RequestID)
	if req.ValidationObject == nil ||
		req.ValidationObject.SearchIdentifier == nil ||
		req.ValidationObject.SearchIdentifier.ResultId == 0 ||
		req.ValidationObject.SearchIdentifier.SearchId == nil {
		response := bookv1.ValidationResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Invalid validation request: missing validation object or search identifier",
					Type:    typesv1.AlertType_ALERT_TYPE_INFO,
				}},
			},
		}
		return &response, nil
	}

	response := bookv1.ValidationResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		ValidationId:     &typesv1.UUID{Value: uuid.New().String()},
		ValidationObject: req.ValidationObject,
		PriceDetail: &typesv1.PriceDetail{
			Price: &typesv1.Price{
				Value:    common.DefaultPricePerNightStr,
				Decimals: common.DefaultPricePerNightDecimals,

				Currency: &typesv1.Currency{
					Currency: &typesv1.Currency_NativeToken{},
				},
			},
			Description: "price per night",
		},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	return &response, nil
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v3/bookv3grpc"
	bookv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/config"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	common "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ bookv3grpc.MintServiceServer = (*mintServiceV3Server)(nil)

type mintServiceV3Server struct {
	eventSender events.Sender
}

func NewMintServiceV3Server(eventSender events.Sender) bookv3grpc.MintServiceServer {
	return &mintServiceV3Server{eventSender: eventSender}
}

func (s *mintServiceV3Server) Mint(ctx context.Context, req *bookv3.MintRequest) (*bookv3.MintResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request (MintV3): %s", md.RequestID)

	storedValidateData, ok := state.GetStore().GetValidationResult(req.ValidationId.Value)
	if !ok {
		return &bookv3.MintResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Validation not found in state",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	mintResponseInfoMessage := "Please note that the price given in this mint response does not reflect the verified total price of the product of '" + storedValidateData.Data.VerifiedPrice.Price + "'. The price is just a minimum value to be able to mint the product."

	response := bookv3.MintResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
			Alerts: []*typesv1.Alert{{
				Message: mintResponseInfoMessage,
				Type:    typesv1.AlertType_ALERT_TYPE_INFO,
			}},
		},
		MintId: &typesv1.UUID{Value: uuid.New().String()},
		BuyableUntil: &timestamppb.Timestamp{
			Seconds: time.Now().Add(config.BuyableUntilDefault).Unix(),
		},
		ValidationId: req.ValidationId,
		Price: &typesv3.Price{
			Value: common.BookingTokenPriceValue,
			Currency: &typesv3.Currency{
				Currency: &typesv3.Currency_NativeToken{},
			},
		},
		Cancellable: true,
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return &response, nil
}

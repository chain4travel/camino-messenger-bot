// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v3/accommodationv3grpc"
	accommodationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/pp-mock/events"
	mockdata "github.com/chain4travel/camino-messenger-bot/pp-mock/services/data"
	"google.golang.org/grpc"
)

var _ accommodationv3grpc.AccommodationProductInfoServiceServer = (*accommodationProductInfoV3Server)(nil)

type accommodationProductInfoV3Server struct {
	eventSender events.Sender
}

func NewAccommodationProductInfoV3Server(eventSender events.Sender) accommodationv3grpc.AccommodationProductInfoServiceServer {
	return &accommodationProductInfoV3Server{eventSender: eventSender}
}

func (s *accommodationProductInfoV3Server) AccommodationProductInfo(ctx context.Context, req *accommodationv3.AccommodationProductInfoRequest) (*accommodationv3.AccommodationProductInfoResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request (Accommodation Product Info): %s", md.RequestID)

	filteredProperties := filterExtendedPropertiesBySupplierCodes(mockdata.PropertiesV3, req.SupplierCodes)
	filteredProperties = filterExtendedPropertiesByLanguage(filteredProperties, req.Languages)

	response := &accommodationv3.AccommodationProductInfoResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Properties: filteredProperties,
	}

	if len(filteredProperties) == 0 {
		response.Header.Alerts = []*typesv1.Alert{{
			Message: "No properties found that match request",
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return response, nil
}

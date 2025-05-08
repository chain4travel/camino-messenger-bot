// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v3/transportv3grpc"
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/pp-mock/events"
	mockdata "github.com/chain4travel/camino-messenger-bot/pp-mock/services/data"
	"google.golang.org/grpc"
)

var _ transportv3grpc.TransportProductListServiceServer = (*transportProductListV3Server)(nil)

type transportProductListV3Server struct {
	eventSender events.Sender
}

func NewTransportProductListV3Server(eventSender events.Sender) transportv3grpc.TransportProductListServiceServer {
	return &transportProductListV3Server{eventSender: eventSender}
}

func (s *transportProductListV3Server) TransportProductList(ctx context.Context, req *transportv3.TransportProductListRequest) (*transportv3.TransportProductListResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (TransportProductList)", md.RequestID)

	filteredTrips := filterPropertiesByLastModified(mockdata.TripsBasicV3, req.GetModifiedAfter().AsTime())

	response := &transportv3.TransportProductListResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Trips: filteredTrips,
	}

	if len(filteredTrips) == 0 {
		response.Header.Alerts = []*typesv1.Alert{{
			Message: "No trips found that match request",
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return response, nil
}

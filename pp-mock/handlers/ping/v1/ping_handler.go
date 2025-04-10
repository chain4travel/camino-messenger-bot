// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/pp-mock/events"
)

var _ pingv1grpc.PingServiceServer = (*pingServiceV1Server)(nil)

type pingServiceV1Server struct {
	eventSender events.Sender
}

func NewPingServiceV1Server(eventSender events.Sender) pingv1grpc.PingServiceServer {
	return &pingServiceV1Server{eventSender: eventSender}
}

func (s *pingServiceV1Server) Ping(ctx context.Context, req *pingv1.PingRequest) (*pingv1.PingResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (Ping)", md.RequestID)

	return &pingv1.PingResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		PingMessage: fmt.Sprintf("Ping response to [%s] with request ID: %s", req.PingMessage, md.RequestID),
	}, nil
}

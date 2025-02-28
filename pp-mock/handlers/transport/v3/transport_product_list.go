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
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	mockdata "github.com/chain4travel/camino-messenger-bot/pp-mock/services/data"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ transportv3grpc.TransportProductListServiceServer = (*TransportProductListV3Server)(nil)

type TransportProductListV3Server struct{}

func (*TransportProductListV3Server) TransportProductList(ctx context.Context, req *transportv3.TransportProductListRequest) (*transportv3.TransportProductListResponse, error) {
	md := metadata.Metadata{}

	// check if req is nil
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request is nil")
	}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request: %s (TransportProductList)", md.RequestID)

	trips := make([]*transportv3.TripBasic, len(mockdata.TripsBasicV3))
	copy(trips, mockdata.TripsBasicV3)

	response := &transportv3.TransportProductListResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Trips: trips,
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return response, nil
}

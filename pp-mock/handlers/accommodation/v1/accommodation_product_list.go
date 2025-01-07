// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	mockdata "github.com/chain4travel/camino-messenger-bot/pp-mock/services/data"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ accommodationv1grpc.AccommodationProductListServiceServer = (*AccommodationProductListV1Server)(nil)

type AccommodationProductListV1Server struct{}

func (*AccommodationProductListV1Server) AccommodationProductList(ctx context.Context, req *accommodationv1.AccommodationProductListRequest) (*accommodationv1.AccommodationProductListResponse, error) {
	md := metadata.Metadata{}

	// check if req is nil
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request is nil")
	}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request (Accommodation Product List): %s", md.RequestID)

	// filter only property objects
	filteredProperties := []*accommodationv1.Property{}
	for _, property := range mockdata.PropertiesV1 {
		filteredProperties = append(filteredProperties, property.Property)
	}

	response := &accommodationv1.AccommodationProductListResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Properties: filteredProperties,
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return response, nil
}

// Copyright (C) 2022-2024, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ bookv2grpc.MintServiceServer = (*MintServiceV2Server)(nil)

type MintServiceV2Server struct{}

func (*MintServiceV2Server) Mint(ctx context.Context, mintRequest *bookv2.MintRequest) (*bookv2.MintResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request (MintV1): %s", md.RequestID)

	response := bookv2.MintResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		MintId: &typesv1.UUID{Value: uuid.New().String()},
		BuyableUntil: &timestamppb.Timestamp{
			Seconds: time.Now().Add(5 * time.Minute).Unix(),
		},
		ValidationId: mintRequest.ValidationId,
		Price: &typesv2.Price{
			Value: "1",
			Currency: &typesv2.Currency{
				Currency: &typesv2.Currency_NativeToken{},
			},
		},
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return &response, nil
}

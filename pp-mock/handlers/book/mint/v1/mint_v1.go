// Copyright (C) 2022-2024, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ bookv1grpc.MintServiceServer = (*MintServiceV1Server)(nil)

type MintServiceV1Server struct{}

func (*MintServiceV1Server) Mint(ctx context.Context, mintRequest *bookv1.MintRequest) (*bookv1.MintResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request (MintV1): %s", md.RequestID)

	response := bookv1.MintResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		MintId: &typesv1.UUID{Value: uuid.New().String()},
		BuyableUntil: &timestamppb.Timestamp{
			Seconds: time.Now().Add(5 * time.Minute).Unix(),
		},
		ValidationId: mintRequest.ValidationId,
		Price: &typesv1.Price{
			Value: "1",
			Currency: &typesv1.Currency{
				Currency: &typesv1.Currency_NativeToken{},
			},
		},
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return &response, nil
}

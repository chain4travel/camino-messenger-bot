package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/services/cache"
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
	md.RequestID = uuid.New().String()
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request (MintV1): %s", md.RequestID)

	cache := cache.NewValidationCache()
	priceDetail, found := cache.GetV1(mintRequest.ValidationId.Value)
	if !found {
		return nil, fmt.Errorf("no validation data found for validationId: %s", mintRequest.ValidationId.Value)
	}

	response := bookv1.MintResponse{
		MintId: &typesv1.UUID{Value: md.RequestID},
		BuyableUntil: &timestamppb.Timestamp{
			Seconds: time.Now().Add(5 * time.Minute).Unix(),
		},
		Price: priceDetail.Price, // change to Token or Offchain to test different scenarios
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return &response, nil
}

package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/services/cache"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ bookv2grpc.MintServiceServer = (*MintServiceV2Server)(nil)

type MintServiceV2Server struct{}

func (*MintServiceV2Server) Mint(ctx context.Context, req *bookv2.MintRequest) (*bookv2.MintResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	cache := cache.NewValidationCache()
	priceDetail, found := cache.GetV2(req.ValidationId.Value)
	if !found {
		return &bookv2.MintResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{
					{
						Message: fmt.Sprintf("no validation data found for validationId: %s", req.ValidationId.Value),
						Type:    typesv1.AlertType_ALERT_TYPE_INFO,
					},
				},
			},
		}, nil
	}

	log.Printf("Responding to request: %s (MintV2)", req.ValidationId.Value)

	response := bookv2.MintResponse{
		MintId: &typesv1.UUID{Value: uuid.New().String()},
		BuyableUntil: &timestamppb.Timestamp{
			Seconds: time.Now().Add(5 * time.Minute).Unix(),
		},
		Price:           priceDetail.Price, // change to Token or Offchain to test different scenarios
		ValidationId:    req.ValidationId,
		BookingTokenUri: "https://example.com/booking-token",
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return &response, nil
}

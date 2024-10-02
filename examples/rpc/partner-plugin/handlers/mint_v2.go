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
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ bookv2grpc.MintServiceServer = (*MintServiceV2Server)(nil)

type MintServiceV2Server struct{}

func (*MintServiceV2Server) Mint(ctx context.Context, _ *bookv2.MintRequest) (*bookv2.MintResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request: %s (MintV2)", md.RequestID)

	// On-chain payment of 1 nCAM value=1 decimals=9 this currency has denominator 18 on
	//
	//	Columbus and conclusively to mint the value of 1 nCam must be divided by 10^9 =
	//	0.000000001 CAM and minted in its smallest fraction by multiplying 0.000000001 *
	//	10^18 => 1000000000 aCAM
	response := bookv2.MintResponse{
		MintId: &typesv1.UUID{Value: md.RequestID},
		BuyableUntil: &timestamppb.Timestamp{
			Seconds: time.Now().Add(5 * time.Minute).Unix(),
		},
		// NATIVE TOKEN EXAMPLE:
		Price: &typesv2.Price{
			Value:    "12345",
			Decimals: 9,
			Currency: &typesv2.Currency{
				Currency: &typesv2.Currency_NativeToken{
					NativeToken: &emptypb.Empty{},
				},
			},
		},
		// ISO CURRENCY EXAMPLE:
		// Price: &typesv2.Price{
		// 	Value:    "10000",
		// 	Decimals: 2,
		// 	Currency: &typesv2.Currency{
		// 		Currency: &typesv2.Currency_IsoCurrency{
		// 			IsoCurrency: typesv2.IsoCurrency_ISO_CURRENCY_EUR,
		// 		},
		// 	},
		// },
		BookingTokenId:  uint64(123456),
		ValidationId:    &typesv1.UUID{Value: "123456"},
		BookingTokenUri: "https://example.com/booking-token",
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return &response, nil
}

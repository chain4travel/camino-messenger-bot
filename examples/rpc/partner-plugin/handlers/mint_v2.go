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

type mintServiceV2Payment struct {
	NativeToken *typesv2.Price
	Token       *typesv2.Price
	Offchain    *typesv2.Price
}

var mintServiceV2Config = mintServiceV2Payment{
	NativeToken: &typesv2.Price{
		Value:    "1",
		Decimals: 9,
		Currency: &typesv2.Currency{
			Currency: &typesv2.Currency_NativeToken{
				NativeToken: &emptypb.Empty{},
			},
		},
	},
	Offchain: &typesv2.Price{
		Value:    "1",
		Decimals: 9,
		Currency: &typesv2.Currency{
			Currency: &typesv2.Currency_IsoCurrency{
				IsoCurrency: typesv2.IsoCurrency_ISO_CURRENCY_EUR, // EUR
			},
		},
	},
	Token: &typesv2.Price{
		Value:    "100",
		Decimals: 2,
		Currency: &typesv2.Currency{
			Currency: &typesv2.Currency_TokenCurrency{
				TokenCurrency: &typesv2.TokenCurrency{
					ContractAddress: "0x87a131801978d1ffBa53a6D4180cBef3F8C9e760",
				},
			},
		},
	},
}

func (*MintServiceV2Server) Mint(ctx context.Context, _ *bookv2.MintRequest) (*bookv2.MintResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request: %s (MintV2)", md.RequestID)

	response := bookv2.MintResponse{
		MintId: &typesv1.UUID{Value: md.RequestID},
		BuyableUntil: &timestamppb.Timestamp{
			Seconds: time.Now().Add(5 * time.Minute).Unix(),
		},
		Price:           mintServiceV2Config.NativeToken, // change to Token or Offchain to test different scenarios
		BookingTokenId:  uint64(123456),
		ValidationId:    &typesv1.UUID{Value: "123456"},
		BookingTokenUri: "https://example.com/booking-token",
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return &response, nil
}

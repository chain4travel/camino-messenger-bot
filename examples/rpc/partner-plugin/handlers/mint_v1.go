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
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ bookv1grpc.MintServiceServer = (*MintServiceV1Server)(nil)

type MintServiceV1Server struct{}

type MintServiceV1Config struct {
	NativeToken *typesv1.Price
	Token       *typesv1.Price
	Offchain    *typesv1.Price
}

var mintServiceV1Config = MintServiceV1Config{
	NativeToken: &typesv1.Price{
		Value:    "1",
		Decimals: 9,
		Currency: &typesv1.Currency{
			Currency: &typesv1.Currency_NativeToken{
				NativeToken: &emptypb.Empty{},
			},
		},
	},
	Offchain: &typesv1.Price{
		Value:    "1",
		Decimals: 9,
		Currency: &typesv1.Currency{
			Currency: &typesv1.Currency_IsoCurrency{
				IsoCurrency: typesv1.IsoCurrency_ISO_CURRENCY_EUR, // EUR
			},
		},
	},
	Token: &typesv1.Price{
		Value:    "100",
		Decimals: 2,
		Currency: &typesv1.Currency{
			Currency: &typesv1.Currency_TokenCurrency{
				TokenCurrency: &typesv1.TokenCurrency{
					ContractAddress: "0x87a131801978d1ffBa53a6D4180cBef3F8C9e760",
				},
			},
		},
	},
}

func (*MintServiceV1Server) Mint(ctx context.Context, _ *bookv1.MintRequest) (*bookv1.MintResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request (MintV1): %s", md.RequestID)

	response := bookv1.MintResponse{
		MintId: &typesv1.UUID{Value: md.RequestID},
		BuyableUntil: &timestamppb.Timestamp{
			Seconds: time.Now().Add(5 * time.Minute).Unix(),
		},
		Price: mintServiceV1Config.Token, // change to Token or Offchain to test different scenarios
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return &response, nil
}

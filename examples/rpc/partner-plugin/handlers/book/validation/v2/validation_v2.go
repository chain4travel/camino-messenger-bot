package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	"google.golang.org/grpc"

	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/helpers"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
)

// Ensure that ValidationServiceV1Server implements the ValidationServiceServer interface
var _ bookv2grpc.ValidationServiceServer = (*ValidationServiceV2Server)(nil)

// ValidationServiceV1Server is the server that provides Validation services.
type ValidationServiceV2Server struct{}

// Validate handles ValidationRequest and returns a mock ValidationResponse.
func (*ValidationServiceV2Server) Validation(ctx context.Context, _ *bookv2.ValidationRequest) (*bookv2.ValidationResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (Validation)", md.RequestID)

	validation := helpers.LoadValidationMockData()

	response := bookv2.ValidationResponse{
		Header:           nil,
		ValidationId:     &typesv1.UUID{Value: md.RequestID},
		ValidationObject: nil,
		PriceDetail: &typesv2.PriceDetail{
			Price: &typesv2.Price{
				Value:    "100",
				Decimals: 0,
				Currency: &typesv2.Currency{
					Currency: &typesv2.Currency_NativeToken{},
				},
			},
		},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

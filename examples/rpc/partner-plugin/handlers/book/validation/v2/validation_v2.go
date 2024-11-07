package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	cache "github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/services/cache"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	// helpers "github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/services/data/v2"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
)

// Ensure that ValidationServiceV1Server implements the ValidationServiceServer interface
var _ bookv2grpc.ValidationServiceServer = (*ValidationServiceV2Server)(nil)

// ValidationServiceV1Server is the server that provides Validation services.
type ValidationServiceV2Server struct{}

// Validate handles ValidationRequest and returns a mock ValidationResponse.
func (*ValidationServiceV2Server) Validation(ctx context.Context, request *bookv2.ValidationRequest) (*bookv2.ValidationResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (Validation)", md.RequestID)

	searchId := request.ValidationObject.SearchIdentifier.SearchId
	resultId := request.ValidationObject.SearchIdentifier.ResultId

	accommodationSearchResponse, ok := cache.Cache.GetV2(searchId.Value) // Directly access using searchId and resultId
	if !ok {
		return nil, fmt.Errorf("no validation data found for searchId: %s", searchId)
	}
	var priceDetail *typesv2.PriceDetail
	for _, result := range accommodationSearchResponse {
		if result.ResultId == resultId {
			priceDetail = result.TotalPriceDetail
		}
	}

	var validationId = typesv1.UUID{Value: uuid.New().String()}
	cache.ValidationCache.SetValidationV2(validationId.Value)

	response := bookv2.ValidationResponse{
		Header:           nil,
		ValidationId:     &validationId,
		ValidationObject: request.ValidationObject,
		PriceDetail:      priceDetail,
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

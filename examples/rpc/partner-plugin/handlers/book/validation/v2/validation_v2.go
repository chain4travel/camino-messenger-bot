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

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
)

// Ensure that ValidationServiceV1Server implements the ValidationServiceServer interface
var _ bookv2grpc.ValidationServiceServer = (*ValidationServiceV2Server)(nil)

// ValidationServiceV1Server is the server that provides Validation services.
type ValidationServiceV2Server struct{}

// Validate handles ValidationRequest and returns a mock ValidationResponse.
func (*ValidationServiceV2Server) Validation(ctx context.Context, validationRequest *bookv2.ValidationRequest) (*bookv2.ValidationResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (Validation)", md.RequestID)

	if validationRequest.ValidationObject == nil ||
		validationRequest.ValidationObject.SearchIdentifier == nil ||
		validationRequest.ValidationObject.SearchIdentifier.ResultId == 0 ||
		validationRequest.ValidationObject.SearchIdentifier.SearchId == nil {
		response := &bookv2.ValidationResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{
					{
						Message: "Invalid validation request: missing validation object or search identifier",
						Type:    typesv1.AlertType_ALERT_TYPE_INFO,
					},
				},
			},
		}
		return response, nil
	}

	searchId := validationRequest.ValidationObject.SearchIdentifier.SearchId
	resultId := validationRequest.ValidationObject.SearchIdentifier.ResultId

	accommodationCache := cache.NewSearchCache()
	validationCache := cache.NewValidationCache()
	accommodationSearchResponse, found := accommodationCache.GetV2(searchId.Value) // Directly access using searchId and resultId
	if !found {
		return &bookv2.ValidationResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{
					{
						Message: fmt.Sprintf("no validation data found for searchId: %v", searchId),
						Type:    typesv1.AlertType_ALERT_TYPE_INFO,
					},
				},
			},
		}, nil
	}
	var priceDetail *typesv2.PriceDetail
	for _, result := range accommodationSearchResponse {
		if result.ResultId == resultId {
			priceDetail = result.TotalPriceDetail
		} else {
			return &bookv2.ValidationResponse{
				Header: &typesv1.ResponseHeader{
					Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
					Alerts: []*typesv1.Alert{
						{
							Message: fmt.Sprintf("no validation data found for resultId: %d", resultId),
							Type:    typesv1.AlertType_ALERT_TYPE_INFO,
						},
					},
				},
			}, nil
		}
	}

	var validationId = typesv1.UUID{Value: uuid.New().String()}
	validationCache.SetV2(validationId.Value, priceDetail)

	response := bookv2.ValidationResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		ValidationId:     &validationId,
		ValidationObject: validationRequest.ValidationObject,
		PriceDetail:      priceDetail,
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

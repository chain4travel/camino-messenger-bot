// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
)

var _ activityv1grpc.ActivitySearchServiceServer = (*activitySearchV1Server)(nil)

type activitySearchV1Server struct {
	eventSender events.Sender
}

func NewActivitySearchV1Server(eventSender events.Sender) activityv1grpc.ActivitySearchServiceServer {
	return &activitySearchV1Server{eventSender: eventSender}
}

func (s *activitySearchV1Server) ActivitySearch(ctx context.Context, req *activityv1.ActivitySearchRequest) (*activityv1.ActivitySearchResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	fmt.Printf("Search generic params: %+v\n", req.SearchParametersGeneric)

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request (Activity Search): %s", md.RequestID)

	// check if SearchParametersGeneric is nil or if Currency is nil
	if req.SearchParametersGeneric == nil || req.SearchParametersGeneric.Currency == nil {
		return &activityv1.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Mandatory field SearchParametersGeneric.Currency is missing",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// Validate travel period
	if req.TravelPeriod == nil {
		return &activityv1.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Mandatory field TravelPeriod is missing. A travel period is required to search for activities (with limits of start/end values of now() / now() + 60 days)",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	if !common.IsTravelPeriodAllowed(req.TravelPeriod) {
		return &activityv1.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Travel period is outside of the allowed constraints. The range is now() - now()+60 days. Additionally the start date must be before the end date.",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// Validate travellers
	if len(req.Travellers) == 0 {
		return &activityv1.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Mandatory field Travellers is missing. At least one traveller is required to search for activities.",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	searchResults := []*activityv1.ActivitySearchResult{}
	resultIDnum := int32(1)
	validationPrices := []*state.UnifiedPrice{}

	filteredActivities := filterSearchResultActivitiesByProductCodes(mockdata.ActivitySearchResultV1, req.SearchParametersActivity.ProductCodes)
	filteredActivities = filterSearchResultActivitiesByServiceCodes(filteredActivities, req.SearchParametersActivity.ServiceCodes)
	filteredActivities = filterSearchResultByCurrency(filteredActivities, req.SearchParametersGeneric.Currency)

	for i, activity := range filteredActivities {
		activity.ResultId = int32(i) + 1
		searchResults = append(searchResults, &activityv1.ActivitySearchResult{
			ResultId: resultIDnum,
			Info: &activityv1.Activity{
				Context:           activity.Info.Context,
				LastModified:      activity.Info.LastModified,
				ExternalSessionId: activity.Info.ExternalSessionId,
				ProductCode:       activity.Info.ProductCode,
				UnitCode:          activity.Info.UnitCode,
				ServiceCode:       activity.Info.ServiceCode,
				Bookability:       activity.Info.Bookability,
			},
			Schedule:        activity.Schedule,
			Location:        activity.Location,
			MinParticipants: activity.MinParticipants,
			MaxParticipants: activity.MaxParticipants,
			ChargeType:      activity.ChargeType,
		})

		validationPrice := state.PriceV1ToUnifiedPrice(activity.Price)
		validationPrices = append(validationPrices, validationPrice)

		resultIDnum++
	}

	response := &activityv1.ActivitySearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Results:    searchResults,
		Travellers: req.Travellers,
	}

	if len(searchResults) == 0 {
		response.Header.Alerts = []*typesv1.Alert{{
			Message: "No results found for search",
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
	} else {
		response.Metadata = &typesv1.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: uuid.New().String()},
		}
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	state.GetStore().AddSearchResult(response.Metadata.SearchId.Value, state.SearchData{
		NumResults:   len(searchResults),
		NumTravelers: len(req.Travellers),
		Prices:       validationPrices,
		JSONRequest:  req.String(),
		JSONResponse: response.String(),
	})

	return response, nil
}

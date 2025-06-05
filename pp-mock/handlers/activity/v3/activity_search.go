// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v3/activityv3grpc"
	activityv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

var _ activityv3grpc.ActivitySearchServiceServer = (*ActivitySearchV3Server)(nil)

type ActivitySearchV3Server struct{}

func (s *ActivitySearchV3Server) ActivitySearch(ctx context.Context, req *activityv3.ActivitySearchRequest) (*activityv3.ActivitySearchResponse, error) {
	log.Printf("ActivitySearch V3 received request: %+v", req)

	md := metadata.Metadata{}

	log.Printf("Activity Search V3 generic params (from req): %+v\n", req.SearchParametersGeneric)
	fmt.Printf("Activity Search V3 generic params: %+v\n", req.SearchParametersGeneric)

	log.Printf("Attempting to extract metadata from context")
	if err := md.ExtractMetadata(ctx); err != nil {
		log.Printf("ERROR extracting metadata: %v", err)
		return &activityv3.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Internal server error: failed to extract request metadata",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	log.Printf("Metadata extracted successfully. Metadata content (before stamp): %+v", md)
	log.Printf("Stamping metadata with Request ID: %s", md.RequestID)
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request (Activity Search V3): %s", md.RequestID)

	log.Printf("Checking if request metadata (req.Metadata) is nil. Value: %+v", req.Metadata)
	if req.Metadata == nil {
		log.Printf("Request metadata (req.Metadata) is missing.")
		return &activityv3.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Metadata is missing",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// Validate search parameters generic and currency
	log.Printf("Checking req.SearchParametersGeneric. Value: %+v", req.SearchParametersGeneric)
	if req.SearchParametersGeneric != nil {
		log.Printf("Checking req.SearchParametersGeneric.Currency. Value: %+v", req.SearchParametersGeneric.Currency)
	} else {
		log.Printf("req.SearchParametersGeneric is nil, currency check will fail.")
	}
	if req.SearchParametersGeneric == nil || req.SearchParametersGeneric.Currency == nil {
		log.Printf("Validation failed: SearchParametersGeneric or Currency is missing.")
		return &activityv3.ActivitySearchResponse{
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
	log.Printf("Checking req.TravelPeriod. Value: %+v", req.TravelPeriod)
	if req.TravelPeriod != nil {
		log.Printf("Checking req.TravelPeriod.StartDate. Value: %+v", req.TravelPeriod.StartDate)
	} else {
		log.Printf("req.TravelPeriod is nil, StartDate check will fail.")
	}
	if req.TravelPeriod == nil || req.TravelPeriod.StartDate == nil {
		log.Printf("Validation failed: TravelPeriod or StartDate is missing.")
		return &activityv3.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Mandatory field TravelPeriod StartDate is missing. A travel period is required to search for activities (with limits of start/end values of now() / now() + 60 days)",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// Validate travellers
	log.Printf("Checking number of travellers. Count: %d. Travellers list: %+v", len(req.Travellers), req.Travellers)
	if len(req.Travellers) == 0 {
		log.Printf("Validation failed: At least one traveller is required.")
		return &activityv3.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "At least one traveller is required to search for activities",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	log.Printf("Initializing outerSearchResults slice")
	outerSearchResults := []*activityv3.ActivitySearchResult{}
	log.Printf("Initializing resultIDnum to 1")
	resultIDnum := int32(1)
	log.Printf("Initializing validationPrices slice")
	validationPrices := []*state.UnifiedPrice{}

	log.Printf("Assigning filteredActivities using mockdata.ActivitySearchResultV3")
	filteredActivities := mockdata.ActivitySearchResultV3
	log.Printf("Number of activities to process (from mock data): %d", len(filteredActivities))

	// Generate search results
	log.Printf("Starting loop through filtered activities")
	for i, activity := range filteredActivities {
		log.Printf("Processing loop iteration %d. Current activity ProductCode: %s", i, activity.Info.ProductCode)
		log.Printf("Activity details: %+v", activity)

		log.Printf("Checking if activity.Price is nil. Activity Product Code: %s. Price value: %+v", activity.Info.ProductCode, activity.Price)
		if activity.Price == nil {
			log.Printf("Skipping activity %s due to missing price information", activity.Info.ProductCode)
			continue
		}

		log.Printf("Preparing to assign searchPrice. Using activity Price: %+v and request Currency: %+v", activity.Price, req.SearchParametersGeneric.Currency)
		searchPrice := &typesv3.Price{
			Value:    activity.Price.Value,
			Decimals: activity.Price.Decimals,
			Currency: common.CloneProto(req.SearchParametersGeneric.Currency),
		}
		log.Printf("Assigned searchPrice: %+v", searchPrice)

		log.Printf("Preparing to append to outerSearchResults. Current Result ID: %d. Activity Info: %+v, Price: %+v", resultIDnum, activity.Info, searchPrice)
		outerSearchResults = append(outerSearchResults, &activityv3.ActivitySearchResult{
			ResultId: resultIDnum,
			Info: &activityv3.Activity{
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
			Price:           searchPrice,
			ChargeType:      activity.ChargeType,
		})
		log.Printf("Appended to outerSearchResults. New length: %d", len(outerSearchResults))

		log.Printf("Converting searchPrice to UnifiedPrice. searchPrice: %+v", searchPrice)
		validationPrice := state.PriceV3ToUnifiedPrice(searchPrice)
		log.Printf("Created validationPrice: %+v", validationPrice)
		log.Printf("Preparing to append to validationPrices. Price: %+v", validationPrice)
		validationPrices = append(validationPrices, validationPrice)
		log.Printf("Appended to validationPrices. New length: %d", len(validationPrices))

		log.Printf("Incrementing resultIDnum from %d", resultIDnum)
		resultIDnum++
		log.Printf("resultIDnum is now %d", resultIDnum)
	}
	log.Printf("Finished loop through filtered activities.")

	// Generate search ID
	log.Printf("Generating new UUID for searchId")
	searchID := uuid.New().String()
	log.Printf("Generated searchId: %s", searchID)

	// Create response
	log.Printf("Creating final response object. Search ID: %s, Results Count: %d, Travellers Count: %d", searchID, len(outerSearchResults), len(req.Travellers))
	response := &activityv3.ActivitySearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Metadata: &typesv3.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: searchID},
		},
		Results:    outerSearchResults,
		Travellers: req.Travellers,
	}
	log.Printf("Created response object. Header Status: %s, Metadata SearchId: %s, Results count: %d, Travellers count: %d", response.Header.Status, response.Metadata.SearchId.GetValue(), len(response.Results), len(response.Travellers))

	// Add info alert if no results were found
	log.Printf("Checking if outerSearchResults is empty. Length: %d", len(outerSearchResults))
	if len(outerSearchResults) == 0 {
		log.Printf("No results found. Adding INFO alert to response header.")
		response.Header.Alerts = []*typesv1.Alert{{
			Message: "No results found for activity search",
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
		log.Printf("Response header updated with INFO alert: %+v", response.Header)
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	log.Printf("Attempting to set response header metadata using md: %+v", md)
	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("ERROR: Failed to set response header metadata: %v", err)
	} else {
		log.Printf("Successfully set response header metadata.")
	}

	// Store search result in state
	log.Printf("Preparing to store search result in state. Search ID: %s, NumResults: %d, NumTravelers: %d", searchID, len(outerSearchResults), len(req.Travellers))
	log.Printf("State data to be stored: NumResults=%d, NumTravelers=%d, Prices=%+v", len(outerSearchResults), len(req.Travellers), validationPrices)

	state.GetStore().AddSearchResult(searchID, state.SearchData{
		NumResults:   len(outerSearchResults),
		NumTravelers: len(req.Travellers),
		Prices:       validationPrices,
		JSONRequest:  req.String(),
		JSONResponse: response.String(),
	})
	log.Printf("Stored search result in state for Search ID: %s", searchID)

	log.Printf("Returning final response for Request ID: %s", md.RequestID)
	return response, nil
}

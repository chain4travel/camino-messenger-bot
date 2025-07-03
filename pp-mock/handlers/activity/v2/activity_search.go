// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v2

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v2/activityv2grpc"
	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

var _ activityv2grpc.ActivitySearchServiceServer = (*ActivitySearchV2Server)(nil)

type ActivitySearchV2Server struct{}

func (s *ActivitySearchV2Server) ActivitySearch(ctx context.Context, req *activityv2.ActivitySearchRequest) (*activityv2.ActivitySearchResponse, error) {
	// Log the entire incoming request at the beginning
	log.Printf("ActivitySearch received request: %+v", req)

	md := metadata.Metadata{}

	// Log generic search parameters from the request
	log.Printf("Activity Search generic params (from req): %+v\n", req.SearchParametersGeneric)
	// Keep original fmt.Printf if needed for specific console output distinct from logs
	fmt.Printf("Activity Search generic params: %+v\n", req.SearchParametersGeneric)

	log.Printf("Attempting to extract metadata from context")
	if err := md.ExtractMetadata(ctx); err != nil {
		// TODO Improve error handling for metadata extraction - handle consistently across all files. Must either return error or error response.
		log.Printf("ERROR extracting metadata: %v", err) // Log the actual error
		// Consider returning an error response here as well
		return &activityv2.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Internal server error: failed to extract request metadata",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil // Or return err if the framework handles it appropriately
	}
	// Log extracted metadata before stamping
	log.Printf("Metadata extracted successfully. Metadata content (before stamp): %+v", md)

	// Log the metadata request ID before stamping
	log.Printf("Stamping metadata with Request ID: %s", md.RequestID) // Log the ID being used for stamping context
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request (Activity Search): %s", md.RequestID) // Existing log is good

	// Log the request metadata before checking if it's nil
	log.Printf("Checking if request metadata (req.Metadata) is nil. Value: %+v", req.Metadata)
	if req.Metadata == nil {
		log.Printf("Request metadata (req.Metadata) is missing.")
		// return error if metadata is missing
		return &activityv2.ActivitySearchResponse{
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
	// Log the values before the check
	log.Printf("Checking req.SearchParametersGeneric. Value: %+v", req.SearchParametersGeneric)
	if req.SearchParametersGeneric != nil {
		log.Printf("Checking req.SearchParametersGeneric.Currency. Value: %+v", req.SearchParametersGeneric.Currency)
	} else {
		log.Printf("req.SearchParametersGeneric is nil, currency check will fail.")
	}
	if req.SearchParametersGeneric == nil || req.SearchParametersGeneric.Currency == nil {
		log.Printf("Validation failed: SearchParametersGeneric or Currency is missing.")
		return &activityv2.ActivitySearchResponse{
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
	// Log the values before the check
	log.Printf("Checking req.TravelPeriod. Value: %+v", req.TravelPeriod)
	if req.TravelPeriod != nil {
		log.Printf("Checking req.TravelPeriod.StartDate. Value: %+v", req.TravelPeriod.StartDate)
	} else {
		log.Printf("req.TravelPeriod is nil, StartDate check will fail.")
	}
	if req.TravelPeriod == nil || req.TravelPeriod.StartDate == nil {
		log.Printf("Validation failed: TravelPeriod or StartDate is missing.")
		return &activityv2.ActivitySearchResponse{
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
	// Log the value before the check
	log.Printf("Checking number of travellers. Count: %d. Travellers list: %+v", len(req.Travellers), req.Travellers)
	if len(req.Travellers) == 0 {
		log.Printf("Validation failed: At least one traveller is required.")
		return &activityv2.ActivitySearchResponse{
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
	outerSearchResults := []*activityv2.ActivitySearchResult{}
	log.Printf("Initializing resultIDnum to 1")
	resultIDnum := int32(1)
	log.Printf("Initializing validationPrices slice")
	validationPrices := []*state.UnifiedPrice{}

	// Filter activities based on search parameters
	// filteredActivities := filterActivitiesBySearchParameters(mockdata.ActivitySearchResultV2, req)
	log.Printf("Assigning filteredActivities using mockdata.ActivitySearchResultV2") // Log assignment source
	filteredActivities := mockdata.ActivitySearchResultV2
	filteredActivities = filterSearchResultActivitiesByProductCodes(filteredActivities, req.SearchParametersActivity.ProductCodes)
	filteredActivities = filterSearchResultActivitiesByServiceCodes(filteredActivities, req.SearchParametersActivity.ServiceCodes)
	log.Printf("Number of activities to process (from mock data): %d", len(filteredActivities))

	// Calculate duration only if both start and end date are present
	// (Logging related to duration calculation, even if commented out, could be useful if re-enabled)
	// log.Printf("Checking TravelPeriod StartDate: %+v and EndDate: %+v for duration calculation", req.TravelPeriod.GetStartDate(), req.TravelPeriod.GetEndDate())
	// var duration float64 = 1.0 // Default duration if end date is missing? Or handle differently?
	// if req.TravelPeriod.GetEndDate() != nil {
	//  log.Printf("Calculating duration from StartDate %v to EndDate %v", req.TravelPeriod.GetStartDate(), req.TravelPeriod.GetEndDate())
	//  duration = common.DateV1ToTime(req.TravelPeriod.GetEndDate()).Sub(common.DateV1ToTime(req.TravelPeriod.GetStartDate())).Hours() / 24
	//  duration = math.Ceil(duration)
	//  log.Printf("Calculated duration (days): %f", duration)
	// } else {
	//  log.Println("Travel period end date is missing, price calculation might be affected. Using default duration or alternative handling.")
	// }
	// NOTE: The original code calculated duration but didn't seem to use it in the price calculation that was active.
	// The commented-out price calculation used it. If price depends on duration, uncomment and adjust as needed.

	// Generate search results
	log.Printf("Starting loop through filtered activities")
	for i, activity := range filteredActivities { // Added index for clearer logging
		log.Printf("Processing loop iteration %d. Current activity ProductCode: %s", i, activity.Info.ProductCode)
		log.Printf("Activity details: %+v", activity) // Log the whole activity being processed

		// Assuming mockdata.ActivityV2 has a Price field of type *typesv2.Price
		// Log price before checking if nil
		log.Printf("Checking if activity.Price is nil. Activity Product Code: %s. Price value: %+v", activity.Info.ProductCode, activity.Price)
		if activity.Price == nil {
			log.Printf("Skipping activity %s due to missing price information", activity.Info.ProductCode)
			continue // Skip if price info is missing
		}

		// Log values used for searchPrice assignment
		log.Printf("Preparing to assign searchPrice. Using activity Price: %+v and request Currency: %+v", activity.Price, req.SearchParametersGeneric.Currency)
		searchPrice := &typesv2.Price{
			Value:    activity.Price.Value, // Use price from the mock activity data
			Decimals: activity.Price.Decimals,
			Currency: common.CloneProto(req.SearchParametersGeneric.Currency), // Use currency from the request
		}
		log.Printf("Assigned searchPrice: %+v", searchPrice)

		// Log values before appending to outerSearchResults
		log.Printf("Preparing to append to outerSearchResults. Current Result ID: %d. Activity Info: %+v, Price: %+v", resultIDnum, activity.Info, searchPrice)
		outerSearchResults = append(outerSearchResults, &activityv2.ActivitySearchResult{
			ResultId: resultIDnum,
			Info: &activityv2.Activity{
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

		// Log before converting price and appending to validationPrices
		log.Printf("Converting searchPrice to UnifiedPrice. searchPrice: %+v", searchPrice)
		validationPrice := state.PriceV2ToUnifiedPrice(searchPrice)
		log.Printf("Created validationPrice: %+v", validationPrice)
		log.Printf("Preparing to append to validationPrices. Price: %+v", validationPrice)
		validationPrices = append(validationPrices, validationPrice)
		log.Printf("Appended to validationPrices. New length: %d", len(validationPrices))

		// Log before incrementing resultIDnum
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
	response := &activityv2.ActivitySearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS, // Default to success
		},
		// Always include metadata with searchId
		Metadata: &typesv2.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: searchID},
		},
		Results:    outerSearchResults,
		Travellers: req.Travellers, // Return travellers from the request
	}
	// Log key parts of the created response
	log.Printf("Created response object. Header Status: %s, Metadata SearchId: %s, Results count: %d, Travellers count: %d", response.Header.Status, response.Metadata.SearchId.GetValue(), len(response.Results), len(response.Travellers))

	// Add info alert if no results were found
	// Log before the check
	log.Printf("Checking if outerSearchResults is empty. Length: %d", len(outerSearchResults))
	if len(outerSearchResults) == 0 {
		log.Printf("No results found. Adding INFO alert to response header.")
		response.Header.Alerts = []*typesv1.Alert{{
			Message: "No results found for activity search",
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
		// Log the updated header with the alert
		log.Printf("Response header updated with INFO alert: %+v", response.Header)
		// No need to set metadata again here, it's already set above.
	}

	// Log sender/recipient info
	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount) // Existing log is good

	// Log before setting gRPC header
	log.Printf("Attempting to set response header metadata using md: %+v", md)
	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		// Log the error but don't necessarily fail the whole request unless required
		log.Printf("ERROR: Failed to set response header metadata: %v", err)
	} else {
		log.Printf("Successfully set response header metadata.")
	}

	// Store search result in state
	// Log before storing state
	log.Printf("Preparing to store search result in state. Search ID: %s, NumResults: %d, NumTravelers: %d", searchID, len(outerSearchResults), len(req.Travellers))
	// Log the actual data being stored (JSON parts might be large, consider truncating in production)
	// Be cautious logging full request/response if they contain sensitive data.
	log.Printf("State data to be stored: NumResults=%d, NumTravelers=%d, Prices=%+v", len(outerSearchResults), len(req.Travellers), validationPrices)
	// Optionally log JSON strings if debugging requires it and size/sensitivity permits
	// log.Printf("State data JSONRequest (truncated): %s...", req.String()[0:min(len(req.String()), 200)]) // Example truncation
	// log.Printf("State data JSONResponse (truncated): %s...", response.String()[0:min(len(response.String()), 200)]) // Example truncation

	state.GetStore().AddSearchResult(searchID, state.SearchData{
		NumResults:   len(outerSearchResults),
		NumTravelers: len(req.Travellers),
		Prices:       validationPrices,
		JSONRequest:  req.String(),      // Be careful with logging sensitive data from requests
		JSONResponse: response.String(), // Be careful with logging sensitive data from responses
	})
	log.Printf("Stored search result in state for Search ID: %s", searchID)

	// Log before returning
	log.Printf("Returning final response for Request ID: %s", md.RequestID)
	return response, nil
}

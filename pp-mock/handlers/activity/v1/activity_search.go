// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ activityv1grpc.ActivitySearchServiceServer = (*ActivitySearchV1Server)(nil)

type ActivitySearchV1Server struct{}

func (s *ActivitySearchV1Server) ActivitySearch(ctx context.Context, req *activityv1.ActivitySearchRequest) (*activityv1.ActivitySearchResponse, error) {
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
		return nil, status.Error(codes.InvalidArgument, "failed to extract request metadata")
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
		return nil, status.Error(codes.InvalidArgument, "metadata is missing")
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
		return nil, status.Error(codes.InvalidArgument, "mandatory field SearchParametersGeneric.Currency is missing")
	}

	// Validate travel period
	log.Printf("Checking req.TravelPeriod. Value: %+v", req.TravelPeriod)
	if req.TravelPeriod == nil {
		log.Printf("Validation failed: TravelPeriod is missing.")
		return nil, status.Error(codes.InvalidArgument, "mandatory field TravelPeriod is missing")
	}

	// Validate travellers
	// Log the value before the check
	log.Printf("Checking number of travellers. Count: %d. Travellers list: %+v", len(req.Travellers), req.Travellers)
	if len(req.Travellers) == 0 {
		log.Printf("Validation failed: At least one traveller is required.")
		return nil, status.Error(codes.InvalidArgument, "at least one traveller is required to search for activities")
	}

	// FIX: Initialize outer slice, not inside the loop
	log.Printf("Initializing outerSearchResults slice")
	searchResults := []*activityv1.ActivitySearchResult{}
	log.Printf("Initializing resultIDnum to 1")
	resultIDnum := int32(1)
	log.Printf("Initializing validationPrices slice")
	validationPrices := []*state.UnifiedPrice{}

	log.Printf("Filtering activities by product codes: %+v", req.SearchParametersActivity.GetProductCodes())
	log.Printf("Filtering activities by service codes: %+v", req.SearchParametersActivity.GetServiceCodes())

	filteredActivities := mockdata.ActivityExtendedV1
	filteredActivities = filterExtendedActivitiesByProductCodes(filteredActivities, req.SearchParametersActivity.GetProductCodes())
	filteredActivities = filterExtendedActivitiesBySupplierCodes(filteredActivities, req.SearchParametersActivity.GetSupplierCodes())
	filteredActivities = filterExtendedActivitiesByServiceCodes(filteredActivities, req.SearchParametersActivity.GetServiceCodes())

	for _, activity := range filteredActivities {
		// mock price for each activity
		searchPrice := &typesv1.Price{
			Value:    strconv.Itoa(150000 + int(resultIDnum)),
			Decimals: 2,
			Currency: common.CloneProto(req.SearchParametersGeneric.Currency), // Use currency from the request
		}

		searchResult := &activityv1.ActivitySearchResult{
			Info: &activityv1.Activity{
				Context:           activity.Activity.Context,
				LastModified:      activity.Activity.LastModified,
				ExternalSessionId: activity.Activity.ExternalSessionId,
				ProductCode:       activity.Activity.ProductCode,
				UnitCode:          activity.Activity.UnitCode,
				ServiceCode:       activity.Activity.ServiceCode,
				Bookability:       activity.Activity.Bookability,
			},
			ResultId: resultIDnum,
			Schedule: getTotalScheduleFromUnits(activity.Units),
			Location: activity.Location,
			// hardcoding (mocking) participants and price per group
			MinParticipants: 3,
			MaxParticipants: 5,
			Price:           searchPrice,
			ChargeType:      typesv1.ChargeType_CHARGE_TYPE_PER_GROUP,
		}
		searchResults = append(searchResults, searchResult)

		// Log before converting price and appending to validationPrices
		log.Printf("Converting searchPrice to UnifiedPrice. searchPrice: %+v", searchPrice)
		validationPrice := state.PriceV1ToUnifiedPrice(searchPrice)
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
	log.Printf("Creating final response object. Search ID: %s, Results Count: %d, Travellers Count: %d", searchID, len(searchResults), len(req.Travellers))
	response := &activityv1.ActivitySearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS, // Default to success
		},
		// Always include metadata with searchId
		Metadata: &typesv1.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: searchID},
		},
		Results:    searchResults,
		Travellers: req.Travellers, // Return travellers from the request
	}
	// Log key parts of the created response
	log.Printf("Created response object. Header Status: %s, Metadata SearchId: %s, Results count: %d, Travellers count: %d", response.Header.Status, response.Metadata.SearchId.GetValue(), len(response.Results), len(response.Travellers))

	// Add info alert if no results were found
	// Log before the check
	log.Printf("Checking if searchResults is empty. Length: %d", len(searchResults))
	if len(searchResults) == 0 {
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
	log.Printf("Preparing to store search result in state. Search ID: %s, NumResults: %d, NumTravelers: %d", searchID, len(searchResults), len(req.Travellers))
	// Log the actual data being stored (JSON parts might be large, consider truncating in production)
	// Be cautious logging full request/response if they contain sensitive data.
	log.Printf("State data to be stored: NumResults=%d, NumTravelers=%d, Prices=%+v", len(searchResults), len(req.Travellers), validationPrices)
	// Optionally log JSON strings if debugging requires it and size/sensitivity permits
	// log.Printf("State data JSONRequest (truncated): %s...", req.String()[0:min(len(req.String()), 200)]) // Example truncation
	// log.Printf("State data JSONResponse (truncated): %s...", response.String()[0:min(len(response.String()), 200)]) // Example truncation

	state.GetStore().AddSearchResult(searchID, state.SearchData{
		NumResults:   len(searchResults),
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

func getTotalScheduleFromUnits(units []*activityv1.ActivityUnit) *typesv1.DateTimeRange {
	totalSchedule := &typesv1.DateTimeRange{}

	totalSchedule.StartDatetime = units[0].Schedule.StartDatetime
	totalSchedule.EndDatetime = units[0].Schedule.EndDatetime

	for _, unit := range units {
		if unit.Schedule.StartDatetime.AsTime().Before(totalSchedule.StartDatetime.AsTime()) {
			totalSchedule.StartDatetime = unit.Schedule.StartDatetime
		}
		if unit.Schedule.EndDatetime.AsTime().After(totalSchedule.EndDatetime.AsTime()) {
			totalSchedule.EndDatetime = unit.Schedule.EndDatetime
		}
	}
	return totalSchedule
}

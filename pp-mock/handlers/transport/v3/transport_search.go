// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v3/transportv3grpc"
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/pkg/price"
	common "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers"
	mockdata "github.com/chain4travel/camino-messenger-bot/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var _ transportv3grpc.TransportSearchServiceServer = (*TransportSearchV3Server)(nil)

type TransportSearchV3Server struct{}

func (*TransportSearchV3Server) TransportSearch(ctx context.Context, req *transportv3.TransportSearchRequest) (*transportv3.TransportSearchResponse, error) {
	md := metadata.Metadata{}

	// check if req is nil
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request is nil")
	}

	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (TransportSearch) v3", md.RequestID)

	// if there is no query, return no results
	if len(req.Queries) == 0 {
		return &transportv3.TransportSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "No queries provided",
					Type:    typesv1.AlertType_ALERT_TYPE_INFO,
				}},
			},
		}, nil
	}

	var resultIDnum int32 = 1

	searchResults := []*transportv3.TransportSearchResult{}

	for _, query := range req.Queries {
		filteredTrips := mockdata.TripsExtendedV3
		queryTrips := query.GetTrips()
		for _, queryTrip := range queryTrips {
			if queryTrip == nil {
				continue
			}
			searchParametersTransport := queryTrip.GetSearchParametersTransport()
			if searchParametersTransport == nil {
				continue
			}

			if queryTrip.Departure != nil && queryTrip.Arrival != nil && queryTrip.Departure.Date != nil && queryTrip.Arrival.Date != nil {
				departureDate := queryTrip.Departure.Date
				arrivalDate := queryTrip.Arrival.Date
				// Check if the travel period is valid
				if !common.AreTravelDatesValid(departureDate, arrivalDate) {
					return &transportv3.TransportSearchResponse{
						Header: &typesv1.ResponseHeader{
							Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
							Alerts: []*typesv1.Alert{{
								Message: "Invalid travel dates: departure date must be in the future and departure must be before arrival",
								Type:    typesv1.AlertType_ALERT_TYPE_INFO,
							},
							},
						},
					}, nil
				}
			}
			// This is just an example, not real business logic:
			filteredTrips = filterTripsByProductCodes(filteredTrips, searchParametersTransport.GetProductCodes())
			if searchParametersTransport.GetMaxSegments() != 0 {
				filteredTrips = filterTripsByMaxSegments(filteredTrips, searchParametersTransport.GetMaxSegments())
			}
			filteredTrips = filterTripsByMaxPrice(filteredTrips, searchParametersTransport.GetMaxPrice())
		}

		var travellerIDs []int32
		if query.Travellers != nil {
			travellerIDs = common.GetTravellerIDsV3(query.Travellers)
		}
		searchResults = append(searchResults, &transportv3.TransportSearchResult{
			ResultId:        resultIDnum,
			QueryId:         query.QueryId,
			TravellerIds:    travellerIDs,
			TravellingTrips: filteredTrips,
			TotalPrice: &typesv3.PriceDetail{
				Price: &typesv3.Price{
					Value: common.DefaultPrice,
				},
			},
		})
		resultIDnum++
	}

	if len(searchResults) == 0 {
		return &transportv3.TransportSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
				Alerts: []*typesv1.Alert{{
					Message: fmt.Sprintf("No results found for search %v", req.Queries),
					Type:    typesv1.AlertType_ALERT_TYPE_INFO,
				}},
			},
		}, nil
	}

	searchID := uuid.New().String()

	response := &transportv3.TransportSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Metadata: &typesv3.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: searchID},
		},
		Results: searchResults,
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return response, nil
}

func filterTripsByProductCodes(trips []*transportv3.TripExtended, productCodes []*typesv2.ProductCode) []*transportv3.TripExtended {
	if len(productCodes) == 0 {
		return trips
	}

	filtered := make([]*transportv3.TripExtended, 0)
	for _, trip := range trips {
		exists := false
		for _, segment := range trip.Segments {
			for _, code := range productCodes {
				if segment.GetInfo().GetProductCode().Code != "" && segment.GetInfo().GetProductCode().Code == code.GetCode() {
					filtered = append(filtered, trip)
					exists = true
					break
				}
			}
			if exists {
				break // Break out of segments loop after adding the trip
			}
		}
	}
	return filtered
}

func filterTripsByMaxSegments(trips []*transportv3.TripExtended, maxSegments int32) []*transportv3.TripExtended {
	filtered := make([]*transportv3.TripExtended, 0)
	for _, trip := range trips {
		if len(trip.Segments) <= int(maxSegments) {
			filtered = append(filtered, trip)
		}
	}
	return filtered
}

func filterTripsByMaxPrice(trips []*transportv3.TripExtended, maxPrice *typesv3.Price) []*transportv3.TripExtended {
	if maxPrice == nil {
		return trips
	}

	filtered := make([]*transportv3.TripExtended, 0)

	// Convert maxPrice to big.Int for comparison
	maxPriceBigInt, err := price.ToBigInt(maxPrice.Value, maxPrice.Decimals, maxPrice.Decimals)
	if err != nil {
		// Mock simplification: If the max price can't be converted, return all trips
		return trips
	}

	for _, trip := range trips {
		tripPrice := trip.GetPrice()

		if tripPrice == nil {
			continue
		}

		// TODO: @VjeraTurk this is a workaround for currency that is not parsed well from the .json
		if tripPrice.Currency == nil || tripPrice.Currency.Currency == nil {
			tripPrice.Currency = &typesv3.Currency{
				Currency: &typesv3.Currency_IsoCurrency{
					IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
				},
			}
		}

		if !proto.Equal(tripPrice.GetCurrency(), maxPrice.GetCurrency()) {
			continue
		}

		tripPriceBigInt, err := price.ToBigInt(tripPrice.Value, tripPrice.Decimals, maxPrice.Decimals)
		if err != nil {
			log.Printf("Failed to convert trip price: %v", err)
			continue
		}

		// Compare total price with max price
		if tripPriceBigInt.Cmp(maxPriceBigInt) <= 0 {
			filtered = append(filtered, trip)
		}
	}

	return filtered
}

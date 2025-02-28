// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v2/transportv2grpc"
	transportv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
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

var _ transportv2grpc.TransportSearchServiceServer = (*TransportSearchV2Server)(nil)

type TransportSearchV2Server struct{}

func (*TransportSearchV2Server) TransportSearch(ctx context.Context, req *transportv2.TransportSearchRequest) (*transportv2.TransportSearchResponse, error) {
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
	log.Printf("Responding to request: %s (TransportSearch) v2", md.RequestID)

	// if there is no query, return no results
	if len(req.Queries) == 0 {
		return &transportv2.TransportSearchResponse{
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

	searchResults := []*transportv2.TransportSearchResult{}

	for _, query := range req.Queries {
		filteredTrips := mockdata.TripsV2
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
					return &transportv2.TransportSearchResponse{
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

		searchResults = append(searchResults, &transportv2.TransportSearchResult{
			ResultId:        resultIDnum,
			QueryId:         query.QueryId,
			TravellerIds:    common.GetTravellerIDsV2(query.Travellers),
			TravellingTrips: filteredTrips,
			TotalPrice: &typesv2.PriceDetail{
				Price: &typesv2.Price{
					Value: common.DefaultPrice,
				},
			},
		})

		resultIDnum++
	}

	if len(searchResults) == 0 {
		return &transportv2.TransportSearchResponse{
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

	response := &transportv2.TransportSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Metadata: &typesv2.SearchResponseMetadata{
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

func filterTripsByProductCodes(trips []*transportv2.Trip, productCodes []*typesv2.ProductCode) []*transportv2.Trip {
	if len(productCodes) == 0 {
		return trips
	}
	filtered := make([]*transportv2.Trip, 0)
	for _, trip := range trips {
		exists := false
		for _, segment := range trip.Segments {
			for _, code := range productCodes {
				if segment.GetProductCode().Code != "" && segment.GetProductCode().Code == code.GetCode() {
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

func filterTripsByMaxSegments(trips []*transportv2.Trip, maxSegments int32) []*transportv2.Trip {
	filtered := make([]*transportv2.Trip, 0)
	for _, trip := range trips {
		if len(trip.Segments) <= int(maxSegments) {
			filtered = append(filtered, trip)
		}
	}
	return filtered
}

func filterTripsByMaxPrice(trips []*transportv2.Trip, maxPrice *typesv2.Price) []*transportv2.Trip {
	if maxPrice == nil {
		return trips
	}

	filtered := make([]*transportv2.Trip, 0)

	// Convert maxPrice to big.Int for comparison
	maxPriceBigInt, err := price.ToBigInt(maxPrice.Value, maxPrice.Decimals, maxPrice.Decimals)
	if err != nil {
		// Mock simplification: If the max price can't be converted, return all trips
		return trips
	}

	for _, trip := range trips {
		totalPrice := big.NewInt(0)
		validTrip := false // Flag to check if the trip has valid currency

		for _, segment := range trip.Segments {
			// Sum up the segment prices
			if segment.Price == nil {
				continue
			}

			segmentPrice := segment.GetPrice()
			// TODO: @VjeraTurk this is a workaround for currency that is not parsed well from the .json
			if segmentPrice.Currency == nil || segmentPrice.Currency.Currency == nil {
				segmentPrice.Currency = &typesv2.Currency{
					Currency: &typesv2.Currency_IsoCurrency{
						IsoCurrency: typesv2.IsoCurrency_ISO_CURRENCY_EUR,
					},
				}
			}

			if !proto.Equal(segmentPrice.GetCurrency(), maxPrice.GetCurrency()) {
				continue
			}

			segmentPriceBigInt, err := price.ToBigInt(segmentPrice.Value, segmentPrice.Decimals, maxPrice.Decimals)
			if err != nil {
				log.Printf("Failed to convert trip price: %v", err)
				continue
			}

			totalPrice = new(big.Int).Add(totalPrice, segmentPriceBigInt)
			validTrip = true
		}

		// Compare total price with max price only if the trip has valid currency
		if validTrip && totalPrice.Cmp(maxPriceBigInt) <= 0 {
			filtered = append(filtered, trip)
		}
	}

	return filtered
}

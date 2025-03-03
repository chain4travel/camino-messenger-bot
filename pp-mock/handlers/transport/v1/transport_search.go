// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1/transportv1grpc"
	transportv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/pkg/price"
	common "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers"
	mockdata "github.com/chain4travel/camino-messenger-bot/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

var _ transportv1grpc.TransportSearchServiceServer = (*TransportSearchV1Server)(nil)

type TransportSearchV1Server struct{}

func (*TransportSearchV1Server) TransportSearch(ctx context.Context, req *transportv1.TransportSearchRequest) (*transportv1.TransportSearchResponse, error) {
	md := metadata.Metadata{}

	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (TransportSearch) v1", md.RequestID)

	// if there is no query, return no results
	if len(req.Queries) == 0 {
		return &transportv1.TransportSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "No queries provided",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	if req.SearchParameters.GetCurrency() == nil {
		return &transportv1.TransportSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "SearchParameters.Currency is required",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	for queryIndex, query := range req.Queries {
		for queryTripIndex, queryTrip := range query.GetTrips() {
			if queryTrip == nil {
				return &transportv1.TransportSearchResponse{
					Header: &typesv1.ResponseHeader{
						Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv1.Alert{{
							Message: fmt.Sprintf("Invalid query[%d].QueryTrips[%d]: can't be nil", queryIndex, queryTripIndex),
							Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			}

			if queryTrip.Departure == nil || queryTrip.Arrival == nil ||
				queryTrip.Departure.Date == nil || queryTrip.Arrival.Date == nil ||
				queryTrip.Departure.LocationCode == nil || queryTrip.Arrival.LocationCode == nil {
				return &transportv1.TransportSearchResponse{
					Header: &typesv1.ResponseHeader{
						Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv1.Alert{{
							Message: "Invalid trip: departure and arrival must be provided",
							Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			}

			if !common.AreTravelDatesValid(queryTrip.Departure.Date, queryTrip.Arrival.Date) {
				return &transportv1.TransportSearchResponse{
					Header: &typesv1.ResponseHeader{
						Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv1.Alert{{
							Message: "Invalid travel dates: departure date must be in the future and departure must be before arrival",
							Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			}

			searchParametersTransport := queryTrip.GetSearchParametersTransport()
			if searchParametersTransport == nil {
				continue
			}

			// Mock simplification: we're limiting mock example to work only with currencies,
			// that have <= than 18 decimals in order to avoid adding blockchain interaction
			// to pp mock (it would be needed to get erc20 token decimals)
			if searchParametersTransport.GetMinPrice().GetDecimals() > price.NativeTokenDecimals { // 18 decimals
				return &transportv1.TransportSearchResponse{
					Header: &typesv1.ResponseHeader{
						Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv1.Alert{{
							Message: fmt.Sprintf("Invalid min price: decimals must be less than or equal to %d", price.NativeTokenDecimals),
							Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			}
		}
	}

	resultIDnum := int32(1)
	searchResults := []*transportv1.TransportSearchResult{}

	for _, query := range req.Queries {
		filteredTrips := mockdata.TripsV1
		for _, queryTrip := range query.GetTrips() {
			if queryTrip.SearchParametersTransport == nil { // its optional
				continue
			}

			// This is just an example, not real business logic:
			filteredTrips = filterTripsByProductCodes(filteredTrips, queryTrip.SearchParametersTransport.ProductCodes)
			if queryTrip.SearchParametersTransport.MaxSegments != 0 {
				filteredTrips = filterTripsByMaxSegments(filteredTrips, queryTrip.SearchParametersTransport.MaxSegments)
			}
		}

		totalPrice := big.NewInt(0)

		for _, trip := range filteredTrips {
			for _, segment := range trip.Segments {
				price, err := price.ToBigInt(
					segment.Price.Value,
					segment.Price.Decimals,
					price.NativeTokenDecimals, // max possible decimals
				)
				if err != nil {
					return &transportv1.TransportSearchResponse{
						Header: &typesv1.ResponseHeader{
							Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
							Alerts: []*typesv1.Alert{{
								Message: fmt.Sprintf("Failed to convert tripSegment price: %v", err),
								Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
							}},
						},
					}, nil
				}
				totalPrice = new(big.Int).Add(totalPrice, price)
			}
		}

		searchResults = append(searchResults, &transportv1.TransportSearchResult{
			ResultId:        resultIDnum,
			QueryId:         query.QueryId,
			TravellerIds:    common.GetTravellerIDsV1(query.Travellers),
			TravellingTrips: filteredTrips,
			TotalPrice: &typesv1.PriceDetail{
				Price: &typesv1.Price{
					Value:    totalPrice.String(),
					Decimals: price.NativeTokenDecimals,
					Currency: req.SearchParameters.Currency,
				},
			},
		})

		resultIDnum++
	}

	response := &transportv1.TransportSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Results: searchResults,
	}

	if len(searchResults) == 0 {
		response.Header.Alerts = []*typesv1.Alert{{
			Message: fmt.Sprintf("No results found for search %v", req.Queries),
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
	} else {
		response.Metadata = &typesv1.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: uuid.New().String()},
		}
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return response, nil
}

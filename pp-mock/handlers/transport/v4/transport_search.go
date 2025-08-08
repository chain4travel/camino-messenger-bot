// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"
	"fmt"
	"math/big"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v4/transportv4grpc"
	transportv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

var _ transportv4grpc.TransportSearchServiceServer = (*transportSearchV4Server)(nil)

type transportSearchV4Server struct{}

func NewTransportSearchServer() transportv4grpc.TransportSearchServiceServer {
	return &transportSearchV4Server{}
}

func (s *transportSearchV4Server) TransportSearch(_ context.Context, req *transportv4.TransportSearchRequest) (*transportv4.TransportSearchResponse, error) {
	// if there is no query, return no results
	if len(req.Queries) == 0 {
		return &transportv4.TransportSearchResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Message: "No queries provided",
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	if req.SearchParameters.GetCurrency() == nil {
		return &transportv4.TransportSearchResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Message: "SearchParameters.Currency is required",
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// edge-case prevention: check if the traveller definition is identical
	// in all queries. If not return an "unsupported" error.
	unsupportedResp := &transportv4.TransportSearchResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
			Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
			Alerts: []*typesv4.Alert{{
				Message: "Unsupported: Traveller definitions must be identical in all queries",
				Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
			}},
		},
	}
	for queryIndex, query := range req.Queries {
		for queryIndex2, query2 := range req.Queries {
			if queryIndex != queryIndex2 {
				travellersA := query.GetTravellers()
				travellersB := query2.GetTravellers()

				if len(travellersA) != len(travellersB) {
					return unsupportedResp, nil
				}

				for i, travellerA := range travellersA {
					travellerB := travellersB[i]
					if !proto.Equal(travellerA, travellerB) {
						return unsupportedResp, nil
					}
				}
			}
		}
	}

	for queryIndex, query := range req.Queries {
		for queryTripIndex, queryTrip := range query.GetTrips() {
			switch {
			case queryTrip == nil:
				return &transportv4.TransportSearchResponse{
					Header: &typesv4.ResponseHeader{
						BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
						Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv4.Alert{{
							Message: fmt.Sprintf("Invalid query[%d].QueryTrips[%d]: can't be nil", queryIndex, queryTripIndex),
							Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			case queryTrip.Departure == nil || queryTrip.Arrival == nil:
				return &transportv4.TransportSearchResponse{
					Header: &typesv4.ResponseHeader{
						BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
						Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv4.Alert{{
							Message: "Invalid trip filter: departure and arrival must be provided",
							Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			case queryTrip.Departure.Date == nil:
				return &transportv4.TransportSearchResponse{
					Header: &typesv4.ResponseHeader{
						BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
						Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv4.Alert{{
							Message: "Invalid trip filter: departure date must be provided",
							Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			case !queryTrip.Departure.Location.HasLocationCodes() || !queryTrip.Arrival.Location.HasLocationCodes():
				return &transportv4.TransportSearchResponse{
					Header: &typesv4.ResponseHeader{
						BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
						Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv4.Alert{{
							Message: "Unsupported trip filter: departure and arrival must provide location codes",
							Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			case queryTrip.Arrival != nil && queryTrip.Arrival.Date != nil && !common.AreTravelDatesValidV4(queryTrip.Departure.Date, queryTrip.Arrival.Date):
				return &transportv4.TransportSearchResponse{
					Header: &typesv4.ResponseHeader{
						BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
						Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv4.Alert{{
							Message: "Invalid travel dates: departure must be before arrival",
							Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			}
		}
	}

	resultIDnum := uint32(1)
	searchResults := []*transportv4.TransportSearchResult{}
	validationPrices := []*state.UnifiedPrice{}

	decimals := price.NativeTokenDecimals
	switch req.SearchParameters.Currency.Currency.(type) {
	case *typesv4.Currency_NativeToken:
	case *typesv4.Currency_IsoCurrency:
		decimals = price.ISODecimals
	default:
		return &transportv4.TransportSearchResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Message: "not supported currency type; only NativeToken and IsoCurrency are supported",
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				}},
			},
			Results: searchResults,
		}, nil
	}

	tripsFilteredByCurrency := filterTripsByCurrency(mockdata.TripsExtendedV4, req.SearchParameters.Currency)
	if len(tripsFilteredByCurrency) == 0 {
		return &transportv4.TransportSearchResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_SUCCESS,
				Alerts: []*typesv4.Alert{{
					Message: fmt.Sprintf("No trips found for currency %s", req.SearchParameters.Currency.String()),
					Type:    typesv4.AlertType_ALERT_TYPE_INFO,
				}},
			},
			Results: searchResults,
		}, nil
	}

	for _, query := range req.Queries {
		filteredTrips := tripsFilteredByCurrency
		for _, queryTrip := range query.GetTrips() {
			filteredTrips = filterTripsByDates(filteredTrips, queryTrip)
			filteredTrips = filterTripsByLocations(filteredTrips, queryTrip)

			if queryTrip.SearchParametersTransport == nil { // its optional
				continue
			}

			filteredTrips = filterTripsByProductCodes(filteredTrips, queryTrip.SearchParametersTransport.ProductCodes)
			if queryTrip.SearchParametersTransport.MaxSegments != 0 {
				filteredTrips = filterTripsByMaxSegments(filteredTrips, queryTrip.SearchParametersTransport.MaxSegments)
			}
		}

		if len(filteredTrips) == 0 {
			// Nothing left after filtering - just skip ahead to the next query
			continue
		}

		totalPrice := big.NewInt(0)

		for _, trip := range filteredTrips {
			priceBig, err := price.ToBigInt(
				trip.Price.Value,
				int32(trip.Price.Decimals),
				decimals,
			)
			if err != nil {
				return &transportv4.TransportSearchResponse{
					Header: &typesv4.ResponseHeader{
						BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
						Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv4.Alert{{
							Message: fmt.Sprintf("Failed to convert tripSegment price: %v", err),
							Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			}
			totalPrice = new(big.Int).Add(totalPrice, priceBig)
		}

		searchPrice := &typesv4.Price{
			Value:    totalPrice.String(),
			Currency: req.SearchParameters.Currency,
			Decimals: uint32(decimals),
		}

		searchResults = append(searchResults, &transportv4.TransportSearchResult{
			ResultId:        resultIDnum,
			QueryId:         query.QueryId,
			TravellerIds:    common.GetTravellerIDsV4(query.Travellers),
			TravellingTrips: filteredTrips,
			TotalPrice: &typesv4.TotalPrice{
				Value: searchPrice,
			},
		})
		resultIDnum++

		validationPrice := state.PriceV4ToUnifiedPrice(searchPrice)
		validationPrices = append(validationPrices, validationPrice)
	}

	response := &transportv4.TransportSearchResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
			Status:     typesv4.StatusType_STATUS_TYPE_SUCCESS,
		},
		Results: searchResults,
	}

	if len(searchResults) == 0 {
		response.Header.Alerts = []*typesv4.Alert{{
			Message: fmt.Sprintf("No results found for search %v", req.Queries),
			Type:    typesv4.AlertType_ALERT_TYPE_INFO,
		}}
	} else {
		response.Metadata = &typesv4.SearchResponseMetadata{
			SearchId: &typesv4.UUID{Value: uuid.New().String()},
		}
		state.GetStore().AddSearchResult(response.Metadata.SearchId.Value, state.SearchData{
			NumResults:   len(searchResults),
			NumTravelers: len(req.Queries[0].Travellers),
			Prices:       validationPrices,
			JSONRequest:  req.String(),
			JSONResponse: response.String(),
			SeatMapIndex: mockdata.SeatMapTransportIndex,
		})
	}

	return response, nil
}

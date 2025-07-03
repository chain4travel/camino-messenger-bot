// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v3/transportv3grpc"
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

var _ transportv3grpc.TransportSearchServiceServer = (*transportSearchV3Server)(nil)

type transportSearchV3Server struct {
	eventSender events.Sender
}

func NewTransportSearchV3Server(eventSender events.Sender) transportv3grpc.TransportSearchServiceServer {
	return &transportSearchV3Server{eventSender: eventSender}
}

func (s *transportSearchV3Server) TransportSearch(ctx context.Context, req *transportv3.TransportSearchRequest) (*transportv3.TransportSearchResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request: %s (TransportSearch) v3", md.RequestID)

	// if there is no query, return no results
	if len(req.Queries) == 0 {
		return &transportv3.TransportSearchResponse{
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
		return &transportv3.TransportSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "SearchParameters.Currency is required",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// edge-case prevention: check if the traveller definition is identical
	// in all queries. If not return an "unsupported" error.
	unsupportedResp := &transportv3.TransportSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
			Alerts: []*typesv1.Alert{{
				Message: "Unsupported: Traveller definitions must be identical in all queries",
				Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
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
				return &transportv3.TransportSearchResponse{
					Header: &typesv1.ResponseHeader{
						Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv1.Alert{{
							Message: fmt.Sprintf("Invalid query[%d].QueryTrips[%d]: can't be nil", queryIndex, queryTripIndex),
							Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			case queryTrip.Departure == nil || queryTrip.Arrival == nil:
				return &transportv3.TransportSearchResponse{
					Header: &typesv1.ResponseHeader{
						Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv1.Alert{{
							Message: "Invalid trip filter: departure and arrival must be provided",
							Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			case queryTrip.Departure.Date == nil:
				return &transportv3.TransportSearchResponse{
					Header: &typesv1.ResponseHeader{
						Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv1.Alert{{
							Message: "Invalid trip filter: departure date must be provided",
							Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			case !queryTrip.Departure.Location.HasLocationCodes() || !queryTrip.Arrival.Location.HasLocationCodes():
				return &transportv3.TransportSearchResponse{
					Header: &typesv1.ResponseHeader{
						Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv1.Alert{{
							Message: "Unsupported trip filter: departure and arrival must provide location codes",
							Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			case queryTrip.Arrival != nil && queryTrip.Arrival.Date != nil && !common.AreTravelDatesValid(queryTrip.Departure.Date, queryTrip.Arrival.Date):
				return &transportv3.TransportSearchResponse{
					Header: &typesv1.ResponseHeader{
						Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv1.Alert{{
							Message: "Invalid travel dates: departure must be before arrival",
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
				return &transportv3.TransportSearchResponse{
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
	searchResults := []*transportv3.TransportSearchResult{}
	validationPrices := []*state.UnifiedPrice{}

	for _, query := range req.Queries {
		filteredTrips := mockdata.TripsExtendedV3
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
			price, err := price.ToBigInt(
				trip.Price.Value,
				trip.Price.Decimals,
				price.NativeTokenDecimals, // max possible decimals
			)
			if err != nil {
				return &transportv3.TransportSearchResponse{
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

		searchPrice := &typesv3.Price{
			Value:    totalPrice.String(),
			Decimals: price.NativeTokenDecimals,
			Currency: req.SearchParameters.Currency,
		}
		searchResults = append(searchResults, &transportv3.TransportSearchResult{
			ResultId:        resultIDnum,
			QueryId:         query.QueryId,
			TravellerIds:    common.GetTravellerIDsV3(query.Travellers),
			TravellingTrips: filteredTrips,
			TotalPrice: &typesv3.PriceDetail{
				Price: searchPrice,
			},
		})
		resultIDnum++

		validationPrice := state.PriceV3ToUnifiedPrice(searchPrice)
		validationPrices = append(validationPrices, validationPrice)
	}

	response := &transportv3.TransportSearchResponse{
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
		response.Metadata = &typesv3.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: uuid.New().String()},
		}
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	if len(searchResults) > 0 {
		state.GetStore().AddSearchResult(response.Metadata.SearchId.Value, state.SearchData{
			NumResults:   len(searchResults),
			NumTravelers: len(req.Queries[0].Travellers),
			Prices:       validationPrices,
			JSONRequest:  req.String(),
			JSONResponse: response.String(),
		})
	}

	return response, nil
}

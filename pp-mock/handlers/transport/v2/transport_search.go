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
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	common "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

var _ transportv2grpc.TransportSearchServiceServer = (*transportSearchV2Server)(nil)

type transportSearchV2Server struct {
	eventSender events.Sender
}

func NewTransportSearchV2Server(eventSender events.Sender) transportv2grpc.TransportSearchServiceServer {
	return &transportSearchV2Server{eventSender: eventSender}
}

func (s *transportSearchV2Server) TransportSearch(ctx context.Context, req *transportv2.TransportSearchRequest) (*transportv2.TransportSearchResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.Metadata{}

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
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	if req.SearchParameters.GetCurrency() == nil {
		return &transportv2.TransportSearchResponse{
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
				return &transportv2.TransportSearchResponse{
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
				queryTrip.Departure.Date == nil ||
				queryTrip.Departure.LocationCode == nil ||
				queryTrip.Arrival.LocationCode == nil {
				return &transportv2.TransportSearchResponse{
					Header: &typesv1.ResponseHeader{
						Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
						Alerts: []*typesv1.Alert{{
							Message: "Invalid trip filter: departure and arrival must be provided",
							Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
						}},
					},
				}, nil
			}

			if queryTrip.Arrival != nil && queryTrip.Arrival.Date != nil && !common.AreTravelDatesValid(queryTrip.Departure.Date, queryTrip.Arrival.Date) {
				return &transportv2.TransportSearchResponse{
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
				return &transportv2.TransportSearchResponse{
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
	searchResults := []*transportv2.TransportSearchResult{}

	for _, query := range req.Queries {
		filteredTrips := mockdata.TripsV2
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
					return &transportv2.TransportSearchResponse{
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

		searchResults = append(searchResults, &transportv2.TransportSearchResult{
			ResultId:        resultIDnum,
			QueryId:         query.QueryId,
			TravellerIds:    common.GetTravellerIDsV2(query.Travellers),
			TravellingTrips: filteredTrips,
			TotalPrice: &typesv2.PriceDetail{
				Price: &typesv2.Price{
					Value:    totalPrice.String(),
					Decimals: price.NativeTokenDecimals,
					Currency: req.SearchParameters.Currency,
				},
			},
		})

		resultIDnum++
	}

	response := &transportv2.TransportSearchResponse{
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
		response.Metadata = &typesv2.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: uuid.New().String()},
		}
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return response, nil
}

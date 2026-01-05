// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"context"
	"fmt"
	"math/big"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v3/transportv3grpc"
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"

	"github.com/chain4travel/camino-messenger-bot/v12/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

var _ transportv3grpc.TransportSearchServiceServer = (*transportSearchV3Server)(nil)

type transportSearchV3Server struct{}

func NewTransportSearchServer() transportv3grpc.TransportSearchServiceServer {
	return &transportSearchV3Server{}
}

func (s *transportSearchV3Server) TransportSearch(_ context.Context, req *transportv3.TransportSearchRequest) (*transportv3.TransportSearchResponse, error) {
	// if there is no query, return no results
	if len(req.Queries) == 0 {
		return &transportv3.TransportSearchResponse{
			Header: common.ErrorHeaderV1("No queries provided"),
		}, nil
	}

	if req.SearchParameters.GetCurrency() == nil {
		return &transportv3.TransportSearchResponse{
			Header: common.ErrorHeaderV1("SearchParameters.Currency is required"),
		}, nil
	}

	// edge-case prevention: check if the traveller definition is identical
	// in all queries. If not return an "unsupported" error.
	unsupportedResp := &transportv3.TransportSearchResponse{
		Header: common.ErrorHeaderV1("Unsupported: Traveller definitions must be identical in all queries"),
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
					Header: common.ErrorHeaderV1(fmt.Sprintf("Invalid query[%d].QueryTrips[%d]: can't be nil", queryIndex, queryTripIndex)),
				}, nil
			case queryTrip.Departure == nil || queryTrip.Arrival == nil:
				return &transportv3.TransportSearchResponse{
					Header: common.ErrorHeaderV1("Invalid trip filter: departure and arrival must be provided"),
				}, nil
			case queryTrip.Departure.Date == nil:
				return &transportv3.TransportSearchResponse{
					Header: common.ErrorHeaderV1("Invalid trip filter: departure date must be provided"),
				}, nil
			case !queryTrip.Departure.Location.HasLocationCodes() || !queryTrip.Arrival.Location.HasLocationCodes():
				return &transportv3.TransportSearchResponse{
					Header: common.ErrorHeaderV1("Unsupported trip filter: departure and arrival must provide location codes"),
				}, nil
			case queryTrip.Arrival != nil && queryTrip.Arrival.Date != nil && !common.AreTravelDatesValidV1(queryTrip.Departure.Date, queryTrip.Arrival.Date):
				return &transportv3.TransportSearchResponse{
					Header: common.ErrorHeaderV1("Invalid travel dates: departure must be before arrival"),
				}, nil
			}
		}
	}

	resultIDnum := int32(1)
	searchResults := []*transportv3.TransportSearchResult{}
	validationPrices := []*state.UnifiedPrice{}

	decimals := price.NativeTokenDecimals
	switch req.SearchParameters.Currency.Currency.(type) {
	case *typesv3.Currency_NativeToken:
	case *typesv3.Currency_IsoCurrency:
		decimals = price.ISODecimals
	default:
		return &transportv3.TransportSearchResponse{
			Header:  common.ErrorHeaderV1("not supported currency type; only NativeToken and IsoCurrency are supported"),
			Results: searchResults,
		}, nil
	}

	tripsFilteredByCurrency := filterTripsByCurrency(mockdata.TripsExtendedV3, req.SearchParameters.Currency)
	if len(tripsFilteredByCurrency) == 0 {
		return &transportv3.TransportSearchResponse{
			Header:  common.SuccessHeaderWithInfoV1(fmt.Sprintf("No trips found for currency %s", req.SearchParameters.Currency.String())),
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
				trip.Price.Decimals,
				decimals,
			)
			if err != nil {
				return &transportv3.TransportSearchResponse{
					Header: common.ErrorHeaderV1(fmt.Sprintf("Failed to convert tripSegment price: %v", err)),
				}, nil
			}
			totalPrice = new(big.Int).Add(totalPrice, priceBig)
		}

		searchPrice := &typesv3.Price{
			Value:    totalPrice.String(),
			Currency: req.SearchParameters.Currency,
			Decimals: decimals,
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
		Header:  common.SuccessHeaderV1(),
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

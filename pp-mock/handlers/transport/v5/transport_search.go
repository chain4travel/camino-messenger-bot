// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v5

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v5/transportv5grpc"
	transportv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v5"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	typesv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v5"

	"github.com/chain4travel/camino-messenger-bot/v13/pkg/conversion"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/services/data"
)

var _ transportv5grpc.TransportSearchServiceServer = (*transportSearchV5Server)(nil)

type transportSearchV5Server struct{}

func NewTransportSearchServer() transportv5grpc.TransportSearchServiceServer {
	return &transportSearchV5Server{}
}

func (s *transportSearchV5Server) TransportSearch(_ context.Context, req *transportv5.TransportSearchRequest) (*transportv5.TransportSearchResponse, error) {
	// edge-case prevention: check if the traveller definition is identical
	// in all queries. If not return an "unsupported" error.
	for i := 0; i < len(req.Queries); i++ {
		travellersI := req.Queries[i].GetTravellers()
		for j := i + 1; j < len(req.Queries); j++ {
			if !common.ProtoSlicesEqual(travellersI, req.Queries[j].GetTravellers()) {
				return errSearchResp(typesv4.ErrorCode_ERROR_CODE_UNIMPLEMENTED, "Unsupported: Traveller definitions must be identical in all queries"), nil
			}
		}
	}

	// check that all travel dates are valid (departure before arrival) and query IDs are unique
	uniqueQueryIDs := make(map[uint32]struct{})
	for _, query := range req.Queries {
		if _, exists := uniqueQueryIDs[query.QueryId]; exists {
			return errSearchResp(typesv4.ErrorCode_ERROR_CODE_BUSINESS_PROCESS_ERROR, "Unsupported: Duplicate QueryId found in queries"), nil
		}
		uniqueQueryIDs[query.QueryId] = struct{}{}

		for _, queryTrip := range query.Trips {
			if !common.AreTravelDatesValidV4(queryTrip.Departure.Date, queryTrip.Arrival.Date) {
				return errSearchResp(typesv4.ErrorCode_ERROR_CODE_BUSINESS_PROCESS_ERROR, "Invalid travel dates: departure must be before arrival"), nil
			}
		}
	}

	currencyDecimals := price.NativeTokenDecimals
	switch req.SearchParameters.Currency.Currency.(type) {
	case *typesv4.Currency_NativeToken:
	case *typesv4.Currency_IsoCurrency:
		currencyDecimals = price.ISODecimals
	default:
		return errSearchResp(typesv4.ErrorCode_ERROR_CODE_INVALID_CURRENCY, "Not supported currency type; only NativeToken and ISOCurrency are supported"), nil
	}

	resultID := uint32(0)
	searchResults := []*transportv5.TransportSearchResult{}
	validationPrices := []*state.UnifiedPrice{}

	tripsFilteredByCurrency := filterTripsByCurrency(mockdata.TripsV4, req.SearchParameters.Currency)

	for _, query := range req.Queries {
		filteredTrips := tripsFilteredByCurrency
		if len(filteredTrips) == 0 {
			continue
		}

		trip := filteredTrips[0]
		totalPriceBig, err := price.ToBigInt(
			trip.Extended.Price.Value,
			conversion.MustUInt32ToInt32(trip.Extended.Price.Decimals),
			currencyDecimals,
		)
		if err != nil {
			return errSearchResp(typesv4.ErrorCode_ERROR_CODE_INTERNAL, "Failed to convert tripSegment price to big int"), nil
		}

		searchPrice := &typesv5.Price{
			Value:    totalPriceBig.String(),
			Decimals: conversion.MustInt32ToUInt32(currencyDecimals),
			Currency: common.CloneProto(req.SearchParameters.Currency),
		}

		travellingTrips := make([]*transportv5.TripExtended, 1)
		travellingTrips[0] = &transportv5.TripExtended{
			Price:    searchPrice,
			Segments: make([]*transportv5.SegmentExtended, 0), // Placeholder
		}

		searchResults = append(searchResults, &transportv5.TransportSearchResult{
			ResultId:        resultID,
			QueryId:         query.QueryId,
			TravellingTrips: travellingTrips,
			TotalPrice: &typesv5.TotalPrice{
				Value: searchPrice,
			},
			Bookability: &typesv4.Bookability{
				Type: typesv4.BookabilityType_BOOKABILITY_TYPE_AVAILABLE,
			},
		})
		resultID++

		validationPrice := state.PriceV5ToUnifiedPrice(searchPrice)
		validationPrices = append(validationPrices, validationPrice)
	}

	resp := &transportv5.TransportSearchResponse{
		Response: &transportv5.TransportSearchResponse_SuccessResponse{
			SuccessResponse: &transportv5.TransportSearchSuccessResponse{
				Header:   common.SuccessHeaderV4(),
				SearchId: common.NewExpiringUUID(),
				Results:  searchResults,
			},
		},
	}

	if len(searchResults) == 0 {
		common.AddHeaderAlertV4(resp.GetSuccessResponse().Header, typesv4.AlertCode_ALERT_CODE_NO_CONTENT, "No results found")
	} else {
		state.GetStore().AddSearchResult(resp.GetSuccessResponse().SearchId.Id.Value, state.SearchData{
			NumResults:   len(searchResults),
			NumTravelers: len(req.Queries[0].Travellers),
			Prices:       validationPrices,
			JSONRequest:  req.String(),
			JSONResponse: resp.String(),
		})
	}

	return resp, nil
}

func errSearchResp(code typesv4.ErrorCode, message string) *transportv5.TransportSearchResponse {
	return &transportv5.TransportSearchResponse{
		Response: &transportv5.TransportSearchResponse_ErrorResponse{
			ErrorResponse: &transportv5.TransportSearchErrorResponse{
				Header: common.ErrorHeaderV4(code, message),
			},
		},
	}
}

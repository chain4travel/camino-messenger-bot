// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v5

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v5/activityv5grpc"
	activityv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v5"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/services/data"
)

var _ activityv5grpc.ActivitySearchServiceServer = (*activitySearchV5Server)(nil)

type activitySearchV5Server struct{}

func NewActivitySearchServer() activityv5grpc.ActivitySearchServiceServer {
	return &activitySearchV5Server{}
}

func (s *activitySearchV5Server) ActivitySearch(_ context.Context, req *activityv5.ActivitySearchRequest) (*activityv5.ActivitySearchResponse, error) {
	if req.GetSearchParameters() == nil || req.SearchParameters.GetCurrency() == nil {
		return &activityv5.ActivitySearchResponse{
			Response: &activityv5.ActivitySearchResponse_ErrorResponse{
				ErrorResponse: &activityv5.ActivitySearchErrorResponse{
					Header: common.ErrorHeaderV4(typesv4.ErrorCode_ERROR_CODE_BUSINESS_PROCESS_ERROR, "search_parameters.currency is required"),
				},
			},
		}, nil
	}

	if req.GetTravelPeriod() == nil || req.TravelPeriod.GetStartDate() == nil || req.TravelPeriod.GetEndDate() == nil {
		return &activityv5.ActivitySearchResponse{
			Response: &activityv5.ActivitySearchResponse_ErrorResponse{
				ErrorResponse: &activityv5.ActivitySearchErrorResponse{
					Header: common.ErrorHeaderV4(typesv4.ErrorCode_ERROR_CODE_BUSINESS_PROCESS_ERROR, "travel_period (start_date and end_date) is required"),
				},
			},
		}, nil
	}

	if !common.IsTravelPeriodAllowedV4(req.TravelPeriod) {
		return &activityv5.ActivitySearchResponse{
			Response: &activityv5.ActivitySearchResponse_ErrorResponse{
				ErrorResponse: &activityv5.ActivitySearchErrorResponse{
					Header: common.ErrorHeaderV4(typesv4.ErrorCode_ERROR_CODE_BUSINESS_PROCESS_ERROR, common.TravelPeriodErrorStr),
				},
			},
		}, nil
	}

	filteredActivities := filterSearchResultActivitiesBySupplierCodes(mockdata.ActivitySearchResultV5, req.SearchParametersActivity.GetSupplierCodes())
	filteredActivities = filterSearchResultActivitiesByServiceCodes(filteredActivities, req.SearchParametersActivity.GetServiceCodes())
	filteredActivities = filterSearchResultActivitiesByCurrency(filteredActivities, req.SearchParameters.Currency)

	resultIDnum := uint32(0)
	validationPrices := []*state.UnifiedPrice{}

	for _, activity := range filteredActivities {
		activity.ResultId = resultIDnum
		validationPrice := state.PriceV5ToUnifiedPrice(activity.TotalPrice.Value)
		validationPrices = append(validationPrices, validationPrice)
		resultIDnum++
	}
	resp := &activityv5.ActivitySearchResponse{
		Response: &activityv5.ActivitySearchResponse_SuccessResponse{
			SuccessResponse: &activityv5.ActivitySearchSuccessResponse{
				Header:     common.SuccessHeaderV4(),
				SearchId:   common.NewExpiringUUID(),
				Travellers: req.Travellers,
				Results:    filteredActivities,
			},
		},
	}

	if len(filteredActivities) == 0 {
		common.AddHeaderAlertV4(resp.GetSuccessResponse().Header, typesv4.AlertCode_ALERT_CODE_NO_CONTENT, "No results found for search")
	} else {
		state.GetStore().AddSearchResult(resp.GetSuccessResponse().SearchId.Id.Value, state.SearchData{
			NumResults:   len(filteredActivities),
			NumTravelers: len(req.Travellers),
			Prices:       validationPrices,
			JSONRequest:  req.String(),
			JSONResponse: resp.String(),
			SeatMapID:    resp.GetSuccessResponse().Results[0].SeatMapId.GetId(),
		})
	}

	return resp, nil
}

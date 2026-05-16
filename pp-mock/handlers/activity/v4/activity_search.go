// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v4/activityv4grpc"
	activityv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/services/data"
)

var _ activityv4grpc.ActivitySearchServiceServer = (*activitySearchV4Server)(nil)

type activitySearchV4Server struct{}

func NewActivitySearchServer() activityv4grpc.ActivitySearchServiceServer {
	return &activitySearchV4Server{}
}

func (s *activitySearchV4Server) ActivitySearch(_ context.Context, req *activityv4.ActivitySearchRequest) (*activityv4.ActivitySearchResponse, error) {
	if !common.IsTravelPeriodAllowedV4(req.TravelPeriod) {
		return &activityv4.ActivitySearchResponse{
			Response: &activityv4.ActivitySearchResponse_ErrorResponse{
				ErrorResponse: &activityv4.ActivitySearchErrorResponse{
					Header: common.ErrorHeaderV4(typesv4.ErrorCode_ERROR_CODE_BUSINESS_PROCESS_ERROR, common.TravelPeriodErrorStr),
				},
			},
		}, nil
	}

	filteredActivities := filterSearchResultActivitiesBySupplierCodes(mockdata.ActivitySearchResultV4, req.SearchParametersActivity.GetSupplierCodes())
	filteredActivities = filterSearchResultActivitiesByServiceCodes(filteredActivities, req.SearchParametersActivity.GetServiceCodes())
	filteredActivities = filterSearchResultActivitiesByCurrency(filteredActivities, req.SearchParameters.Currency)

	resultIDnum := uint32(0)
	validationPrices := []*state.UnifiedPrice{}

	for _, activity := range filteredActivities {
		activity.ResultId = resultIDnum
		validationPrice := state.PriceV4ToUnifiedPrice(activity.TotalPrice.Value)
		validationPrices = append(validationPrices, validationPrice)
		resultIDnum++
	}
	resp := &activityv4.ActivitySearchResponse{
		Response: &activityv4.ActivitySearchResponse_SuccessResponse{
			SuccessResponse: &activityv4.ActivitySearchSuccessResponse{
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

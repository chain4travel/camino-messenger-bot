// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v4/activityv4grpc"
	activityv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ activityv4grpc.ActivitySearchServiceServer = (*activitySearchV3Server)(nil)

type activitySearchV3Server struct{}

func NewActivitySearchServer() activityv4grpc.ActivitySearchServiceServer {
	return &activitySearchV3Server{}
}

func (s *activitySearchV3Server) ActivitySearch(_ context.Context, req *activityv4.ActivitySearchRequest) (*activityv4.ActivitySearchResponse, error) {
	if !common.IsTravelPeriodAllowedV4(req.TravelPeriod) {
		return &activityv4.ActivitySearchResponse{
			Response: &activityv4.ActivitySearchResponse_ErrorResponse{
				ErrorResponse: &activityv4.ActivitySearchErrorResponse{
					Header: common.ErrorHeaderV4("Travel period is outside of the allowed constraints. The range is now() - now()+60 days. Additionally the start date must be before the end date."),
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
				Header: common.SuccessHeaderV4(),
				SearchId: &typesv4.ExpiringUUID{
					Id:         &typesv4.UUID{Value: uuid.New().String()},
					Expiration: timestamppb.New(time.Now().Add(state.EntryTimeout)),
				},
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

// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v3/activityv3grpc"
	activityv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data"
	"github.com/google/uuid"
)

var _ activityv3grpc.ActivitySearchServiceServer = (*activitySearchV3Server)(nil)

type activitySearchV3Server struct{}

func NewActivitySearchServer() activityv3grpc.ActivitySearchServiceServer {
	return &activitySearchV3Server{}
}

func (s *activitySearchV3Server) ActivitySearch(_ context.Context, req *activityv3.ActivitySearchRequest) (*activityv3.ActivitySearchResponse, error) {
	// check if SearchParametersGeneric is nil or if Currency is nil
	if req.SearchParametersGeneric == nil || req.SearchParametersGeneric.Currency == nil {
		return &activityv3.ActivitySearchResponse{
			Header: common.ErrorHeaderV1("Mandatory field SearchParametersGeneric.Currency is missing"),
		}, nil
	}

	// Validate travel period
	if req.TravelPeriod == nil {
		return &activityv3.ActivitySearchResponse{
			Header: common.ErrorHeaderV1("Mandatory field TravelPeriod is missing. A travel period is required to search for activities (with limits of start/end values of now() / now() + 60 days)"),
		}, nil
	}

	if !common.IsTravelPeriodAllowedV1(req.TravelPeriod) {
		return &activityv3.ActivitySearchResponse{
			Header: common.ErrorHeaderV1(common.TravelPeriodErrorStr),
		}, nil
	}

	// Validate travellers
	if len(req.Travellers) == 0 {
		return &activityv3.ActivitySearchResponse{
			Header: common.ErrorHeaderV1("Mandatory field Travellers is missing. At least one traveller is required to search for activities."),
		}, nil
	}

	resultIDnum := int32(1)
	validationPrices := []*state.UnifiedPrice{}

	filteredActivities := filterSearchResultActivitiesByProductCodes(mockdata.ActivitySearchResultV3, req.SearchParametersActivity.ProductCodes)
	filteredActivities = filterSearchResultActivitiesByServiceCodes(filteredActivities, req.SearchParametersActivity.ServiceCodes)
	filteredActivities = filterSearchResultByCurrency(filteredActivities, req.SearchParametersGeneric.Currency)

	for _, activity := range filteredActivities {
		activity.ResultId = resultIDnum
		validationPrice := state.PriceV3ToUnifiedPrice(activity.Price)
		validationPrices = append(validationPrices, validationPrice)
		resultIDnum++
	}

	response := &activityv3.ActivitySearchResponse{
		Header:     common.SuccessHeaderV1(),
		Results:    filteredActivities,
		Travellers: req.Travellers,
	}

	if len(filteredActivities) == 0 {
		response.Header.Alerts = []*typesv1.Alert{{
			Message: "No results found for search",
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
	} else {
		response.Metadata = &typesv3.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: uuid.New().String()},
		}
		state.GetStore().AddSearchResult(response.Metadata.SearchId.Value, state.SearchData{
			NumResults:   len(filteredActivities),
			NumTravelers: len(req.Travellers),
			Prices:       validationPrices,
			JSONRequest:  req.String(),
			JSONResponse: response.String(),
		})
	}

	return response, nil
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"
	"fmt"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
)

var _ activityv1grpc.ActivitySearchServiceServer = (*activitySearchV1Server)(nil)

type activitySearchV1Server struct{}

func NewActivitySearchServer() activityv1grpc.ActivitySearchServiceServer {
	return &activitySearchV1Server{}
}

func (s *activitySearchV1Server) ActivitySearch(_ context.Context, req *activityv1.ActivitySearchRequest) (*activityv1.ActivitySearchResponse, error) {
	fmt.Printf("Search generic params: %+v\n", req.SearchParametersGeneric)

	// check if SearchParametersGeneric is nil or if Currency is nil
	if req.SearchParametersGeneric == nil || req.SearchParametersGeneric.Currency == nil {
		return &activityv1.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Mandatory field SearchParametersGeneric.Currency is missing",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// Validate travel period
	if req.TravelPeriod == nil {
		return &activityv1.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Mandatory field TravelPeriod is missing. A travel period is required to search for activities (with limits of start/end values of now() / now() + 60 days)",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	if !common.IsTravelPeriodAllowedV1(req.TravelPeriod) {
		return &activityv1.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Travel period is outside of the allowed constraints. The range is now() - now()+60 days. Additionally the start date must be before the end date.",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// Validate travellers
	if len(req.Travellers) == 0 {
		return &activityv1.ActivitySearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Mandatory field Travellers is missing. At least one traveller is required to search for activities.",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	resultIDnum := int32(1)
	validationPrices := []*state.UnifiedPrice{}

	filteredActivities := filterSearchResultActivitiesByProductCodes(mockdata.ActivitySearchResultV1, req.SearchParametersActivity.ProductCodes)
	filteredActivities = filterSearchResultActivitiesByServiceCodes(filteredActivities, req.SearchParametersActivity.ServiceCodes)
	filteredActivities = filterSearchResultByCurrency(filteredActivities, req.SearchParametersGeneric.Currency)

	for _, activity := range filteredActivities {
		activity.ResultId = resultIDnum
		validationPrice := state.PriceV1ToUnifiedPrice(activity.Price)
		validationPrices = append(validationPrices, validationPrice)
		resultIDnum++
	}

	response := &activityv1.ActivitySearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Results:    filteredActivities,
		Travellers: req.Travellers,
	}

	if len(filteredActivities) == 0 {
		response.Header.Alerts = []*typesv1.Alert{{
			Message: "No results found for search",
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
	} else {
		response.Metadata = &typesv1.SearchResponseMetadata{
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

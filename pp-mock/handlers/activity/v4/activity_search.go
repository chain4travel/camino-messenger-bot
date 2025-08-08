// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"
	"fmt"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v4/activityv4grpc"
	activityv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
)

var _ activityv4grpc.ActivitySearchServiceServer = (*activitySearchV3Server)(nil)

type activitySearchV3Server struct{}

func NewActivitySearchServer() activityv4grpc.ActivitySearchServiceServer {
	return &activitySearchV3Server{}
}

func (s *activitySearchV3Server) ActivitySearch(_ context.Context, req *activityv4.ActivitySearchRequest) (*activityv4.ActivitySearchResponse, error) {
	fmt.Printf("Search params: %+v\n", req.SearchParameters)
	// check if SearchParameters is nil or if Currency is nil
	if req.SearchParameters == nil || req.SearchParameters.Currency == nil {
		return &activityv4.ActivitySearchResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Message: "Mandatory field SearchParameters.Currency is missing",
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// Validate travel period
	if req.TravelPeriod == nil {
		return &activityv4.ActivitySearchResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Message: "Mandatory field TravelPeriod is missing. A travel period is required to search for activities (with limits of start/end values of now() / now() + 60 days)",
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	if !common.IsTravelPeriodAllowedV4(req.TravelPeriod) {
		return &activityv4.ActivitySearchResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Message: "Travel period is outside of the allowed constraints. The range is now() - now()+60 days. Additionally the start date must be before the end date.",
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// Validate travellers
	if len(req.Travellers) == 0 {
		return &activityv4.ActivitySearchResponse{
			Header: &typesv4.ResponseHeader{
				BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
				Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv4.Alert{{
					Message: "Mandatory field Travellers is missing. At least one traveller is required to search for activities.",
					Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	resultIDnum := uint32(1)
	validationPrices := []*state.UnifiedPrice{}

	filteredActivities := filterSearchResultActivitiesByProductCodes(mockdata.ActivitySearchResultV4, req.SearchParametersActivity.ProductCodes)
	filteredActivities = filterSearchResultActivitiesByServiceCodes(filteredActivities, req.SearchParametersActivity.ServiceCodes)
	filteredActivities = filterSearchResultByCurrency(filteredActivities, req.SearchParameters.Currency)

	for _, activity := range filteredActivities {
		activity.ResultId = resultIDnum
		validationPrice := state.PriceV4ToUnifiedPrice(activity.TotalPrice.Value)
		validationPrices = append(validationPrices, validationPrice)
		resultIDnum++
	}

	response := &activityv4.ActivitySearchResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
			Status:     typesv4.StatusType_STATUS_TYPE_SUCCESS,
		},
		Results:    filteredActivities,
		Travellers: req.Travellers,
	}

	if len(filteredActivities) == 0 {
		response.Header.Alerts = []*typesv4.Alert{{
			Message: "No results found for search",
			Type:    typesv4.AlertType_ALERT_TYPE_INFO,
		}}
	} else {
		response.Metadata = &typesv4.SearchResponseMetadata{
			SearchId: &typesv4.UUID{Value: uuid.New().String()},
		}
		state.GetStore().AddSearchResult(response.Metadata.SearchId.Value, state.SearchData{
			NumResults:   len(filteredActivities),
			NumTravelers: len(req.Travellers),
			Prices:       validationPrices,
			JSONRequest:  req.String(),
			JSONResponse: response.String(),
			SeatMapIndex: mockdata.SeatMapActivityIndex,
		})
	}

	return response, nil
}

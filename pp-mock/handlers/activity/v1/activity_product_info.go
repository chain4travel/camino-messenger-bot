// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ activityv1grpc.ActivityProductInfoServiceServer = (*activityProductInfoV1Server)(nil)

type activityProductInfoV1Server struct{}

func NewActivityProductInfoServer() activityv1grpc.ActivityProductInfoServiceServer {
	return &activityProductInfoV1Server{}
}

func (s *activityProductInfoV1Server) ActivityProductInfo(_ context.Context, req *activityv1.ActivityProductInfoRequest) (*activityv1.ActivityProductInfoResponse, error) {
	filteredActivities := filterBySupplierCodes(mockdata.ActivityExtendedV1, req.SupplierCodes)
	filteredActivities = filterExtendedByLastModified(filteredActivities, req.ModifiedAfter.AsTime())
	filteredActivities = filterByLanguage(filteredActivities, req.Languages)

	response := &activityv1.ActivityProductInfoResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Activities: filteredActivities,
	}

	if len(filteredActivities) == 0 {
		response.Header.Alerts = []*typesv1.Alert{{
			Message: "No activities found that match request",
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
	}

	return response, nil
}

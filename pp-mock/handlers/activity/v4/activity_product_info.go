// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v4/activityv4grpc"
	activityv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ activityv4grpc.ActivityProductInfoServiceServer = (*activityProductInfoV3Server)(nil)

type activityProductInfoV3Server struct{}

func NewActivityProductInfoServer() activityv4grpc.ActivityProductInfoServiceServer {
	return &activityProductInfoV3Server{}
}

func (s *activityProductInfoV3Server) ActivityProductInfo(_ context.Context, req *activityv4.ActivityProductInfoRequest) (*activityv4.ActivityProductInfoResponse, error) {
	filteredActivities := filterBySupplierCodes(mockdata.ActivityExtendedV4, req.SupplierCodes)
	filteredActivities = filterExtendedByLastModified(filteredActivities, req.ModifiedAfter.AsTime())
	filteredActivities = filterExtendedByLanguage(filteredActivities, req.Languages)

	response := &activityv4.ActivityProductInfoResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
			Status:     typesv4.StatusType_STATUS_TYPE_SUCCESS,
		},
		Activities: filteredActivities,
	}

	if len(filteredActivities) == 0 {
		response.Header.Alerts = []*typesv4.Alert{{
			Message: "No activities found that match request",
			Type:    typesv4.AlertType_ALERT_TYPE_INFO,
		}}
	}

	return response, nil
}

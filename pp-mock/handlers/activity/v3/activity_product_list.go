// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v3/activityv3grpc"
	activityv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ activityv3grpc.ActivityProductListServiceServer = (*activityProductListV3Server)(nil)

type activityProductListV3Server struct{}

func NewActivityProductListServer() activityv3grpc.ActivityProductListServiceServer {
	return &activityProductListV3Server{}
}

func (s *activityProductListV3Server) ActivityProductList(_ context.Context, req *activityv3.ActivityProductListRequest) (*activityv3.ActivityProductListResponse, error) {
	filteredActivities := filterByLastModified(mockdata.ActivityV3, req.GetModifiedAfter().AsTime())

	response := &activityv3.ActivityProductListResponse{
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

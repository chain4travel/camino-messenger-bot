// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v2

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v2/activityv2grpc"
	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data"
)

var _ activityv2grpc.ActivityProductListServiceServer = (*activityProductListV2Server)(nil)

type activityProductListV2Server struct{}

func NewActivityProductListServer() activityv2grpc.ActivityProductListServiceServer {
	return &activityProductListV2Server{}
}

func (s *activityProductListV2Server) ActivityProductList(_ context.Context, req *activityv2.ActivityProductListRequest) (*activityv2.ActivityProductListResponse, error) {
	filteredActivities := filterByLastModified(mockdata.ActivityV2, req.GetModifiedAfter().AsTime())

	response := &activityv2.ActivityProductListResponse{
		Header:     common.SuccessHeaderV1(),
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

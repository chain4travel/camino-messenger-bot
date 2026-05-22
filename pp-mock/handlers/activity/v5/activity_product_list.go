// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v5

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/services/data"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v5/activityv5grpc"
	activityv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v5"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
)

var _ activityv5grpc.ActivityProductListServiceServer = (*activityProductListV5Server)(nil)

type activityProductListV5Server struct{}

func NewActivityProductListServer() activityv5grpc.ActivityProductListServiceServer {
	return &activityProductListV5Server{}
}

func (s *activityProductListV5Server) ActivityProductList(_ context.Context, req *activityv5.ActivityProductListRequest) (*activityv5.ActivityProductListResponse, error) {
	filteredActivities := filterExtendedBySupplierCodes(mockdata.ActivityExtendedV5, req.SupplierCodes)

	response := &activityv5.ActivityProductListResponse{
		Response: &activityv5.ActivityProductListResponse_SuccessResponse{
			SuccessResponse: &activityv5.ActivityProductListSuccessResponse{
				Header:     common.SuccessHeaderV4(),
				Activities: extendedToActivityInfo(filteredActivities),
			},
		},
	}

	if len(filteredActivities) == 0 {
		common.AddHeaderAlertV4(response.GetSuccessResponse().Header, typesv4.AlertCode_ALERT_CODE_NO_CONTENT, "No activities found that match request")
	}

	return response, nil
}

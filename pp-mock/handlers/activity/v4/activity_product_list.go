// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/services/data"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v4/activityv4grpc"
	activityv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
)

var _ activityv4grpc.ActivityProductListServiceServer = (*activityProductListV4Server)(nil)

type activityProductListV4Server struct{}

func NewActivityProductListServer() activityv4grpc.ActivityProductListServiceServer {
	return &activityProductListV4Server{}
}

func (s *activityProductListV4Server) ActivityProductList(_ context.Context, req *activityv4.ActivityProductListRequest) (*activityv4.ActivityProductListResponse, error) {
	filteredActivities := filterExtendedBySupplierCodes(mockdata.ActivityExtendedV4, req.SupplierCodes)

	response := &activityv4.ActivityProductListResponse{
		Response: &activityv4.ActivityProductListResponse_SuccessResponse{
			SuccessResponse: &activityv4.ActivityProductListSuccessResponse{
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

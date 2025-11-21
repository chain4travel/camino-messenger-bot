// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v4/activityv4grpc"
	activityv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
)

var _ activityv4grpc.ActivityProductInfoServiceServer = (*activityProductInfoV4Server)(nil)

type activityProductInfoV4Server struct{}

func NewActivityProductInfoServer() activityv4grpc.ActivityProductInfoServiceServer {
	return &activityProductInfoV4Server{}
}

func (s *activityProductInfoV4Server) ActivityProductInfo(_ context.Context, req *activityv4.ActivityProductInfoRequest) (*activityv4.ActivityProductInfoResponse, error) {
	filteredActivities := filterExtendedBySupplierCodes(mockdata.ActivityExtendedV4, req.SupplierCodes)
	filteredActivities = filterExtendedByLanguage(filteredActivities, req.Languages)

	response := &activityv4.ActivityProductInfoResponse{
		Response: &activityv4.ActivityProductInfoResponse_SuccessResponse{
			SuccessResponse: &activityv4.ActivityProductInfoSuccessResponse{
				Header:     common.SuccessHeaderV4(),
				Activities: filteredActivities,
			},
		},
	}

	if len(filteredActivities) == 0 {
		common.AddHeaderAlertV4(response.GetSuccessResponse().Header, typesv4.AlertCode_ALERT_CODE_NO_CONTENT, "No activities found that match request")
	}

	return response, nil
}

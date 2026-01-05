// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
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

var _ activityv4grpc.ActivityProductShortListServiceServer = (*activityProductShortListV4Server)(nil)

type activityProductShortListV4Server struct{}

func NewActivityProductShortListServer() activityv4grpc.ActivityProductShortListServiceServer {
	return &activityProductShortListV4Server{}
}

func (s *activityProductShortListV4Server) ActivityProductShortList(_ context.Context, req *activityv4.ActivityProductShortListRequest) (*activityv4.ActivityProductShortListResponse, error) {
	filteredActivities := filterExtendedByModifiedAfter(mockdata.ActivityExtendedV4, req.GetModifiedAfter().AsTime())

	response := &activityv4.ActivityProductShortListResponse{
		Response: &activityv4.ActivityProductShortListResponse_SuccessResponse{
			SuccessResponse: &activityv4.ActivityProductShortListSuccessResponse{
				Header:                 common.SuccessHeaderV4(),
				ActivityShortListItems: extendedToShortListItem(filteredActivities),
			},
		},
	}

	if len(filteredActivities) == 0 {
		common.AddHeaderAlertV4(response.GetSuccessResponse().Header, typesv4.AlertCode_ALERT_CODE_NO_CONTENT, "No activities found that match request")
	}

	return response, nil
}

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

var _ activityv5grpc.ActivityProductShortListServiceServer = (*activityProductShortListV5Server)(nil)

type activityProductShortListV5Server struct{}

func NewActivityProductShortListServer() activityv5grpc.ActivityProductShortListServiceServer {
	return &activityProductShortListV5Server{}
}

func (s *activityProductShortListV5Server) ActivityProductShortList(_ context.Context, req *activityv5.ActivityProductShortListRequest) (*activityv5.ActivityProductShortListResponse, error) {
	filteredActivities := filterExtendedByModifiedAfter(mockdata.ActivityExtendedV5, req.GetModifiedAfter().AsTime())

	response := &activityv5.ActivityProductShortListResponse{
		Response: &activityv5.ActivityProductShortListResponse_SuccessResponse{
			SuccessResponse: &activityv5.ActivityProductShortListSuccessResponse{
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

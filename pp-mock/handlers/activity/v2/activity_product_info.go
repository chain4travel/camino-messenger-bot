// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
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

var _ activityv2grpc.ActivityProductInfoServiceServer = (*activityProductInfoV2Server)(nil)

type activityProductInfoV2Server struct{}

func NewActivityProductInfoServer() activityv2grpc.ActivityProductInfoServiceServer {
	return &activityProductInfoV2Server{}
}

func (s *activityProductInfoV2Server) ActivityProductInfo(_ context.Context, req *activityv2.ActivityProductInfoRequest) (*activityv2.ActivityProductInfoResponse, error) {
	filteredActivities := filterBySupplierCodes(mockdata.ActivityExtendedV2, req.SupplierCodes)
	filteredActivities = filterExtendedByLastModified(filteredActivities, req.ModifiedAfter.AsTime())
	filteredActivities = filterByLanguage(filteredActivities, req.Languages)

	response := &activityv2.ActivityProductInfoResponse{
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

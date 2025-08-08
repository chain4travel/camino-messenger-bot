// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

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
	var filteredSupplierProductCodes []*typesv4.SupplierProductCode
	// filteredActivities := filterByLastModified(mockdata.ActivityV4, req.GetModifiedAfter().AsTime())
	// TODO@ activityv4 doesn't contain modified after field, so we probably need different source data

	response := &activityv4.ActivityProductListResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
			Status:     typesv4.StatusType_STATUS_TYPE_SUCCESS,
		},
		SupplierCodes: filteredSupplierProductCodes,
	}

	if len(filteredSupplierProductCodes) == 0 {
		response.Header.Alerts = []*typesv4.Alert{{
			Message: "No activities found that match request",
			Type:    typesv4.AlertType_ALERT_TYPE_INFO,
		}}
	}

	return response, nil
}

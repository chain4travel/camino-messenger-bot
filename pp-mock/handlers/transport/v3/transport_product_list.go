// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v3/transportv3grpc"
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ transportv3grpc.TransportProductListServiceServer = (*transportProductListV3Server)(nil)

type transportProductListV3Server struct{}

func NewTransportProductListServer() transportv3grpc.TransportProductListServiceServer {
	return &transportProductListV3Server{}
}

func (s *transportProductListV3Server) TransportProductList(_ context.Context, req *transportv3.TransportProductListRequest) (*transportv3.TransportProductListResponse, error) {
	filteredTrips := filterPropertiesByLastModified(mockdata.TripsBasicV3, req.GetModifiedAfter().AsTime())

	response := &transportv3.TransportProductListResponse{
		Header: common.SuccessHeaderV1(),
		Trips:  filteredTrips,
	}

	if len(filteredTrips) == 0 {
		response.Header.Alerts = []*typesv1.Alert{{
			Message: "No trips found that match request",
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
	}

	return response, nil
}

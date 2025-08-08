// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v4/transportv4grpc"
	transportv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ transportv4grpc.TransportProductListServiceServer = (*transportProductListV4Server)(nil)

type transportProductListV4Server struct{}

func NewTransportProductListServer() transportv4grpc.TransportProductListServiceServer {
	return &transportProductListV4Server{}
}

func (s *transportProductListV4Server) TransportProductList(_ context.Context, req *transportv4.TransportProductListRequest) (*transportv4.TransportProductListResponse, error) {
	filteredTrips := filterPropertiesByLastModified(mockdata.TripsBasicV4, req.GetModifiedAfter().AsTime())

	response := &transportv4.TransportProductListResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
			Status:     typesv4.StatusType_STATUS_TYPE_SUCCESS,
		},
		Trips: filteredTrips,
	}

	if len(filteredTrips) == 0 {
		response.Header.Alerts = []*typesv4.Alert{{
			Message: "No trips found that match request",
			Type:    typesv4.AlertType_ALERT_TYPE_INFO,
		}}
	}

	return response, nil
}

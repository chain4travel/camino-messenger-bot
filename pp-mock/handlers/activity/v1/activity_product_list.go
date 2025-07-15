// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"
	"log"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ activityv1grpc.ActivityProductListServiceServer = (*activityProductListV1Server)(nil)

type activityProductListV1Server struct {
	eventSender events.Sender
}

func NewActivityProductListV1Server(eventSender events.Sender) activityv1grpc.ActivityProductListServiceServer {
	return &activityProductListV1Server{eventSender: eventSender}
}

func (s *activityProductListV1Server) ActivityProductList(ctx context.Context, req *activityv1.ActivityProductListRequest) (*activityv1.ActivityProductListResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request (Activity Product List): %s", md.RequestID)

	var lastModifiedFilter time.Time
	if req.ModifiedAfter != nil {
		lastModifiedFilter = req.ModifiedAfter.AsTime()
	}

	filteredActivities := []*activityv1.Activity{}
	for _, activity := range mockdata.ActivityV1 {
		if activity.LastModified.AsTime().Before(lastModifiedFilter) {
			continue
		}

		filteredActivities = append(filteredActivities, activity)
	}

	log.Printf("Filtered activities: %v", filteredActivities)

	response := &activityv1.ActivityProductListResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Activities: filteredActivities,
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	return response, nil
}

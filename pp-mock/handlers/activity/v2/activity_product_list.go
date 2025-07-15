// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v2

import (
	"context"
	"log"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v2/activityv2grpc"
	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
)

var _ activityv2grpc.ActivityProductListServiceServer = (*activityProductListV2Server)(nil)

type activityProductListV2Server struct {
	eventSender events.Sender
}

func NewActivityProductListV2Server(eventSender events.Sender) activityv2grpc.ActivityProductListServiceServer {
	return &activityProductListV2Server{eventSender: eventSender}
}

func (s *activityProductListV2Server) ActivityProductList(ctx context.Context, req *activityv2.ActivityProductListRequest) (*activityv2.ActivityProductListResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.FromGRPCContext(ctx)

	log.Printf("Responding to request (Activity Product List): %s", md.RequestID)

	var lastModifiedFilter time.Time
	if req.ModifiedAfter != nil {
		lastModifiedFilter = req.ModifiedAfter.AsTime()
	}

	filteredActivities := []*activityv2.Activity{}
	for _, activity := range mockdata.ActivityV2 {
		if activity.LastModified.AsTime().Before(lastModifiedFilter) {
			continue
		}

		filteredActivities = append(filteredActivities, activity)
	}

	log.Printf("Filtered activities: %v", filteredActivities)

	response := &activityv2.ActivityProductListResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Activities: filteredActivities,
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	return response, nil
}

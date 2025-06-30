// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"
	"fmt"
	"log"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"

	"google.golang.org/grpc"
)

var _ activityv1grpc.ActivityProductListServiceServer = (*ActivityProductListV1Server)(nil)

type ActivityProductListV1Server struct{}

func (*ActivityProductListV1Server) ActivityProductList(ctx context.Context, req *activityv1.ActivityProductListRequest) (*activityv1.ActivityProductListResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
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

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return response, nil
}

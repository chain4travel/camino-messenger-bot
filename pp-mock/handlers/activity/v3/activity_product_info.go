// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v3/activityv3grpc"
	activityv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var _ activityv3grpc.ActivityProductInfoServiceServer = (*ActivityProductInfoV3Server)(nil)

type ActivityProductInfoV3Server struct{}

func (*ActivityProductInfoV3Server) ActivityProductInfo(ctx context.Context, req *activityv3.ActivityProductInfoRequest) (*activityv3.ActivityProductInfoResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request (Activity Product Info V3): %s", md.RequestID)

	// Initialize activitiesFiltered with the correct type
	activitiesFiltered := []*activityv3.ActivityExtendedInfo{}

	// Check if there are supplier codes in the request
	if len(req.SupplierCodes) > 0 {
		log.Printf("Supplier codes requested: %v", req.SupplierCodes)
		// Filter activities by supplier codes
		for _, activity := range mockdata.ActivityExtendedV3 {
			if activity.SupplierCode != nil {
				for _, sc := range req.SupplierCodes {
					if activity.SupplierCode.SupplierCode == sc.SupplierCode {
						activitiesFiltered = append(activitiesFiltered, proto.Clone(activity).(*activityv3.ActivityExtendedInfo))
						break
					}
				}
			}
		}
	} else {
		// If no supplier codes provided, return all activities
		for i := range mockdata.ActivityExtendedV3 {
			cloned := proto.Clone(mockdata.ActivityExtendedV3[i]).(*activityv3.ActivityExtendedInfo)
			activitiesFiltered = append(activitiesFiltered, cloned)
		}
	}

	// Apply modified after filter if provided
	if req.ModifiedAfter != nil {
		lastModifiedFilter := req.ModifiedAfter.AsTime()
		tempActivities := []*activityv3.ActivityExtendedInfo{}

		for _, activity := range activitiesFiltered {
			if activity != nil && activity.Activity != nil && activity.Activity.LastModified != nil &&
				!activity.Activity.LastModified.AsTime().Before(lastModifiedFilter) {
				tempActivities = append(tempActivities, activity)
			}
		}
		activitiesFiltered = tempActivities
	}

	// Filter by language if specified
	filteredActivities := []*activityv3.ActivityExtendedInfo{}

	if len(req.Languages) > 0 {
		log.Printf("Languages requested: %v", req.Languages)

		for _, activity := range activitiesFiltered {
			filteredDescriptions := []*typesv1.LocalizedDescriptionSet{}

			for _, descSet := range activity.Descriptions {
				for _, reqLang := range req.Languages {
					if descSet.Language == reqLang {
						filteredDescriptions = append(filteredDescriptions, descSet)
						break
					}
				}
			}

			if len(filteredDescriptions) > 0 && !containsActivityV3(filteredActivities, activity) {
				activity.Descriptions = filteredDescriptions
				filteredActivities = append(filteredActivities, activity)
			}
		}
	} else {
		filteredActivities = activitiesFiltered
	}

	if len(filteredActivities) == 0 {
		return &activityv3.ActivityProductInfoResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
				Alerts: []*typesv1.Alert{{
					Message: fmt.Sprintf("No activities found for supplier codes: %v", req.SupplierCodes),
					Type:    typesv1.AlertType_ALERT_TYPE_INFO,
				}},
			},
		}, nil
	}

	response := &activityv3.ActivityProductInfoResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Activities: filteredActivities,
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	log.Printf("Response: %v", response)

	return response, nil
}

// containsActivityV3 checks if an activity already exists in the slice
func containsActivityV3(activities []*activityv3.ActivityExtendedInfo, activity *activityv3.ActivityExtendedInfo) bool {
	if activity == nil || activity.Activity == nil || activity.SupplierCode == nil {
		return false
	}

	for _, a := range activities {
		if a.Activity != nil && a.SupplierCode != nil &&
			a.SupplierCode.SupplierCode == activity.SupplierCode.SupplierCode {
			return true
		}
	}
	return false
}

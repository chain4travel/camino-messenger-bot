// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"google.golang.org/protobuf/proto"
)

func filterExtendedActivitiesByProductCodes(
	activities []*activityv1.ActivityExtendedInfo,
	productCodes []*typesv1.ProductCode,
) []*activityv1.ActivityExtendedInfo {
	if len(productCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv1.ActivityExtendedInfo{}
	for _, activity := range activities {
		for _, code := range productCodes {
			if activity.Activity.ProductCode.Code == code.Code {
				filtered = append(filtered, common.CloneProto(activity))
				break
			}
		}
	}
	return filtered
}

func filterExtendedActivitiesBySupplierCodes(
	activities []*activityv1.ActivityExtendedInfo,
	supplierCodes []*typesv1.SupplierProductCode,
) []*activityv1.ActivityExtendedInfo {
	if len(supplierCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv1.ActivityExtendedInfo{}
	for _, activity := range activities {
		for _, code := range supplierCodes {
			if proto.Equal(activity.SupplierCode, code) {
				filtered = append(filtered, common.CloneProto(activity))
				break
			}
		}
	}
	return filtered
}

func filterExtendedActivitiesByServiceCodes(
	activities []*activityv1.ActivityExtendedInfo,
	serviceCodes []string,
) []*activityv1.ActivityExtendedInfo {
	if len(serviceCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv1.ActivityExtendedInfo{}

	for _, activity := range activities {
		for _, code := range serviceCodes {
			if activity.Activity.ServiceCode == code {
				filtered = append(filtered, common.CloneProto(activity))
				break
			}
		}
	}
	return filtered
}

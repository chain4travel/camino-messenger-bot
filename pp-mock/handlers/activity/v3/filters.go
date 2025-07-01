// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	activityv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v3"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
)

func filterSearchResultActivitiesByProductCodes(
	activities []*activityv3.ActivitySearchResult,
	productCodes []*typesv2.ProductCode,
) []*activityv3.ActivitySearchResult {
	if len(productCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv3.ActivitySearchResult{}
	for _, activity := range activities {
		for _, code := range productCodes {
			if activity.Info.ProductCode.Code == code.Code {
				filtered = append(filtered, common.CloneProto(activity))
				break
			}
		}
	}
	return filtered
}

func filterSearchResultActivitiesByServiceCodes(
	activities []*activityv3.ActivitySearchResult,
	serviceCodes []string,
) []*activityv3.ActivitySearchResult {
	if len(serviceCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv3.ActivitySearchResult{}

	for _, activity := range activities {
		for _, code := range serviceCodes {
			if activity.Info.ServiceCode == code {
				filtered = append(filtered, common.CloneProto(activity))
				break
			}
		}
	}
	return filtered
}

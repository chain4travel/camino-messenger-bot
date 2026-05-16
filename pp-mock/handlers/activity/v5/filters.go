// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v5

import (
	activityv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v5"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	"google.golang.org/protobuf/proto"
)

// Filters search results based on supplier codes
func filterSearchResultActivitiesBySupplierCodes(
	activities []*activityv5.ActivitySearchResult,
	supplierCodes []*typesv4.SupplierProductCode,
) []*activityv5.ActivitySearchResult {
	if len(supplierCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv5.ActivitySearchResult{}
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

// Filters search results based on service codes
func filterSearchResultActivitiesByServiceCodes(
	activities []*activityv5.ActivitySearchResult,
	serviceCodes []string,
) []*activityv5.ActivitySearchResult {
	if len(serviceCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv5.ActivitySearchResult{}
	for _, activity := range activities {
		for _, code := range serviceCodes {
			if activity.ServiceCode == code {
				filtered = append(filtered, common.CloneProto(activity))
				break
			}
		}
	}
	return filtered
}

// Filters search results based on currency
func filterSearchResultActivitiesByCurrency(
	activities []*activityv5.ActivitySearchResult,
	currency *typesv4.Currency,
) []*activityv5.ActivitySearchResult {
	if currency == nil {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv5.ActivitySearchResult{}
	for _, activity := range activities {
		if proto.Equal(activity.TotalPrice.Value.Currency, currency) {
			filtered = append(filtered, common.CloneProto(activity))
		}
	}
	return filtered
}

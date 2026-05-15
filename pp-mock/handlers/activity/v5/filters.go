// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v5

import (
	activityv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v4"
	activityv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v5"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/localization"
	"google.golang.org/protobuf/proto"
)

// Filters activities based on supplier codes
func filterExtendedActivitiesBySupplierCodes(
	activities []*activityv4.ActivityExtendedInfo,
	supplierCodes []*typesv4.SupplierProductCode,
) []*activityv4.ActivityExtendedInfo {
	if len(supplierCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv4.ActivityExtendedInfo{}
	for _, act := range activities {
		for _, code := range supplierCodes {
			if proto.Equal(act.Activity.SupplierCode, code) {
				filtered = append(filtered, common.CloneProto(act))
				break
			}
		}
	}
	return filtered
}

// Filters activities based on language
func filterExtendedActivitiesByLanguage(
	activities []*activityv4.ActivityExtendedInfo,
	languages []typesv1.Language,
) []*activityv4.ActivityExtendedInfo {
	if len(languages) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv4.ActivityExtendedInfo{}
	for _, activity := range activities {
		filteredDescriptions := localization.FilterDescriptionsV4(activity.Descriptions, languages)
		if len(filteredDescriptions) > 0 {
			clonedActivity := common.CloneProto(activity)
			clonedActivity.Descriptions = filteredDescriptions
			filtered = append(filtered, clonedActivity)
		}
	}
	return filtered
}

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

// Filters activities based on supplier codes for Activity Product List
func filterActivitiesBySupplierCodes(
	activities []*activityv4.ActivityExtendedInfo,
	supplierCodes []*typesv4.SupplierProductCode,
) []*activityv4.ActivityInfo {
	filtered := []*activityv4.ActivityInfo{}

	if len(supplierCodes) == 0 {
		for _, activity := range activities {
			filtered = append(filtered, common.CloneProto(activity.Activity))
		}
		return filtered
	}

	for _, activity := range activities {
		for _, code := range supplierCodes {
			if proto.Equal(activity.Activity.SupplierCode, code) {
				filtered = append(filtered, common.CloneProto(activity.Activity))
				break
			}
		}
	}

	return filtered
}

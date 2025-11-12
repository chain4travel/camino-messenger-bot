// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"time"

	activityv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v4"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/localization"
	"google.golang.org/protobuf/proto"
)

func filterExtendedBySupplierCodes(
	activities []*activityv4.ActivityExtendedInfo,
	supplierCodes []*typesv4.SupplierProductCode,
) []*activityv4.ActivityExtendedInfo {
	if len(supplierCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv4.ActivityExtendedInfo{}
	for _, activity := range activities {
		for _, code := range supplierCodes {
			if proto.Equal(activity.Activity.SupplierCode, code) {
				filtered = append(filtered, common.CloneProto(activity))
				break
			}
		}
	}
	return filtered
}

func filterExtendedByModifiedAfter(
	activities []*activityv4.ActivityExtendedInfo,
	modifiedAfter time.Time,
) []*activityv4.ActivityExtendedInfo {
	filtered := []*activityv4.ActivityExtendedInfo{}
	for _, activity := range activities {
		if activity.Activity.LastModified.AsTime().After(modifiedAfter) {
			filtered = append(filtered, common.CloneProto(activity))
		}
	}
	return filtered
}

func filterExtendedByLanguage(
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

func filterSearchResultActivitiesBySupplierCodes(
	activities []*activityv4.ActivitySearchResult,
	supplierCodes []*typesv4.SupplierProductCode,
) []*activityv4.ActivitySearchResult {
	if len(supplierCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv4.ActivitySearchResult{}
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

func filterSearchResultActivitiesByServiceCodes(
	activities []*activityv4.ActivitySearchResult,
	serviceCodes []string,
) []*activityv4.ActivitySearchResult {
	if len(serviceCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv4.ActivitySearchResult{}

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

func filterSearchResultActivitiesByCurrency(activities []*activityv4.ActivitySearchResult, currency *typesv4.Currency) []*activityv4.ActivitySearchResult {
	filtered := []*activityv4.ActivitySearchResult{}
	for _, activity := range activities {
		if proto.Equal(activity.TotalPrice.Value.Currency, currency) {
			filtered = append(filtered, common.CloneProto(activity))
		}
	}
	return filtered
}

func extendedToShortListItem(activities []*activityv4.ActivityExtendedInfo) []*activityv4.ActivityShortListItem {
	shortListItems := make([]*activityv4.ActivityShortListItem, 0, len(activities))
	for _, activity := range activities {
		shortListItems = append(shortListItems, &activityv4.ActivityShortListItem{
			SupplierCode: common.CloneProto(activity.Activity.SupplierCode),
			Status:       activity.Activity.Status,
		})
	}
	return shortListItems
}

func extendedToActivityInfo(activities []*activityv4.ActivityExtendedInfo) []*activityv4.ActivityInfo {
	infoItems := make([]*activityv4.ActivityInfo, 0, len(activities))
	for _, activity := range activities {
		infoItems = append(infoItems, common.CloneProto(activity.Activity))
	}
	return infoItems
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"time"

	activityv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v4"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/localization"
	"google.golang.org/protobuf/proto"
)

func filterBySupplierCodes(
	activities []*activityv4.ActivityExtendedInfo,
	supplierCodes []*typesv4.SupplierProductCode,
) []*activityv4.ActivityExtendedInfo {
	if len(supplierCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv4.ActivityExtendedInfo{}
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

func filterExtendedByLastModified(
	activities []*activityv4.ActivityExtendedInfo,
	lastModified time.Time,
) []*activityv4.ActivityExtendedInfo {
	filtered := []*activityv4.ActivityExtendedInfo{}
	for _, activity := range activities {
		if !activity.LastModified.AsTime().Before(lastModified) {
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
			if proto.Equal(activity.Info.SupplierCode, code) {
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
			if activity.Info.ServiceCode == code {
				filtered = append(filtered, common.CloneProto(activity))
				break
			}
		}
	}
	return filtered
}

func filterSearchResultByCurrency(activities []*activityv4.ActivitySearchResult, currency *typesv4.Currency) []*activityv4.ActivitySearchResult {
	filtered := []*activityv4.ActivitySearchResult{}
	for _, activity := range activities {
		if proto.Equal(activity.TotalPrice.Value.Currency, currency) {
			filtered = append(filtered, common.CloneProto(activity))
		}
	}
	return filtered
}

func extendedToSupplierProductCodes(activities []*activityv4.ActivityExtendedInfo) []*typesv4.SupplierProductCode {
	supplierProductCodes := make([]*typesv4.SupplierProductCode, 0, len(activities))
	for _, activity := range activities {
		supplierProductCodes = append(supplierProductCodes, activity.SupplierCode)
	}
	return supplierProductCodes
}

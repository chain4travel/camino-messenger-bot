// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"time"

	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"google.golang.org/protobuf/proto"
)

func filterBySupplierCodes(
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

func filterExtendedByLastModified(
	activities []*activityv1.ActivityExtendedInfo,
	lastModified time.Time,
) []*activityv1.ActivityExtendedInfo {
	filtered := []*activityv1.ActivityExtendedInfo{}
	for _, activity := range activities {
		if !activity.Activity.LastModified.AsTime().Before(lastModified) {
			filtered = append(filtered, common.CloneProto(activity))
		}
	}
	return filtered
}

func filterByLastModified(
	activities []*activityv1.Activity,
	lastModified time.Time,
) []*activityv1.Activity {
	filtered := []*activityv1.Activity{}
	for _, activity := range activities {
		if !activity.LastModified.AsTime().Before(lastModified) {
			filtered = append(filtered, common.CloneProto(activity))
		}
	}
	return filtered
}

func filterByLanguage(
	activities []*activityv1.ActivityExtendedInfo,
	languages []typesv1.Language,
) []*activityv1.ActivityExtendedInfo {
	if len(languages) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv1.ActivityExtendedInfo{}
	for _, activity := range activities {
		filteredDescriptions := []*typesv1.LocalizedDescriptionSet{}

		for _, descSet := range activity.Descriptions {
			for _, reqLang := range languages {
				if descSet.Language == reqLang {
					filteredDescriptions = append(filteredDescriptions, descSet)
					break
				}
			}
		}

		if len(filteredDescriptions) > 0 {
			clonedActivity := common.CloneProto(activity)
			clonedActivity.Descriptions = filteredDescriptions
			filtered = append(filtered, clonedActivity)
		}
	}
	return filtered
}

func filterSearchResultActivitiesByProductCodes(
	activities []*activityv1.ActivitySearchResult,
	productCodes []*typesv1.ProductCode,
) []*activityv1.ActivitySearchResult {
	if len(productCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv1.ActivitySearchResult{}
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
	activities []*activityv1.ActivitySearchResult,
	serviceCodes []string,
) []*activityv1.ActivitySearchResult {
	if len(serviceCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv1.ActivitySearchResult{}

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

func filterSearchResultByCurrency(activities []*activityv1.ActivitySearchResult, currency *typesv1.Currency) []*activityv1.ActivitySearchResult {
	filtered := []*activityv1.ActivitySearchResult{}
	for _, activity := range activities {
		if proto.Equal(activity.Price.Currency, currency) {
			filtered = append(filtered, common.CloneProto(activity))
		}
	}
	return filtered
}

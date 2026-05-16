// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"time"

	activityv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	"google.golang.org/protobuf/proto"
)

func filterBySupplierCodes(
	activities []*activityv3.ActivityExtendedInfo,
	supplierCodes []*typesv2.SupplierProductCode,
) []*activityv3.ActivityExtendedInfo {
	if len(supplierCodes) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv3.ActivityExtendedInfo{}
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
	activities []*activityv3.ActivityExtendedInfo,
	lastModified time.Time,
) []*activityv3.ActivityExtendedInfo {
	filtered := []*activityv3.ActivityExtendedInfo{}
	for _, activity := range activities {
		if !activity.Activity.LastModified.AsTime().Before(lastModified) {
			filtered = append(filtered, common.CloneProto(activity))
		}
	}
	return filtered
}

func filterByLastModified(
	activities []*activityv3.Activity,
	lastModified time.Time,
) []*activityv3.Activity {
	filtered := []*activityv3.Activity{}
	for _, activity := range activities {
		if !activity.LastModified.AsTime().Before(lastModified) {
			filtered = append(filtered, common.CloneProto(activity))
		}
	}
	return filtered
}

func filterByLanguage(
	activities []*activityv3.ActivityExtendedInfo,
	languages []typesv1.Language,
) []*activityv3.ActivityExtendedInfo {
	if len(languages) == 0 {
		return common.CloneProtoSlice(activities)
	}

	filtered := []*activityv3.ActivityExtendedInfo{}
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

func filterSearchResultByCurrency(activities []*activityv3.ActivitySearchResult, currency *typesv3.Currency) []*activityv3.ActivitySearchResult {
	filtered := []*activityv3.ActivitySearchResult{}
	for _, activity := range activities {
		if proto.Equal(activity.Price.Currency, currency) {
			filtered = append(filtered, common.CloneProto(activity))
		}
	}
	return filtered
}

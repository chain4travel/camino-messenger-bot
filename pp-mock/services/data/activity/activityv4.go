// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package activity

import (
	"fmt"
	"slices"

	activityv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v4"
	"google.golang.org/protobuf/proto"
)

func Verify(extended []*activityv4.ActivityExtendedInfo, searchResults []*activityv4.ActivitySearchResult) {
	expectedCount := 0
	for _, activity := range extended {
		expectedCount += len(activity.Units) * len(activity.Services)
	}
	if len(searchResults) != expectedCount {
		panic(fmt.Errorf("mock data error: expected %d search results but got %d", expectedCount, len(searchResults)))
	}

	i := 0
	for _, activity := range extended {
		for _, unit := range activity.Units {
			for _, service := range activity.Services {
				searchResult := searchResults[i]
				if !proto.Equal(activity.Activity.SupplierCode, searchResult.SupplierCode) {
					panic("activityExtendedInfo and searchResult supplier code mismatch")
				}
				if unit.Code != searchResult.UnitCode {
					panic("activityExtendedInfo and searchResult unit code mismatch")
				}
				if service.Code != searchResult.ServiceCode {
					panic("activityExtendedInfo and searchResult service code mismatch")
				}
				for _, searchResultZoneCode := range searchResult.ZoneCodes {
					if !slices.ContainsFunc(activity.Zones, func(z *activityv4.TransferZone) bool {
						return z.Code == searchResultZoneCode
					}) {
						panic("activityExtendedInfo and searchResult zone code mismatch")
					}
				}
				i++
			}
		}
	}
}

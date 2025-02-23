// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"time"

	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
)

const DefaultPricePerNight = 105.33

func DateV1ToTime(date *typesv1.Date) time.Time {
	return time.Date(int(date.GetYear()), time.Month(date.GetMonth()), int(date.GetDay()), 0, 0, 0, 0, time.UTC)
}

// only period between now + 60 days is allowed for bookings
func IsTravelPeriodAllowed(travelPeriod *typesv1.TravelPeriod) bool {
	startDate := time.Now()
	endDate := time.Now().Add(time.Hour * 24 * 60) // 60 days from now

	return DateV1ToTime(travelPeriod.StartDate).After(startDate) && DateV1ToTime(travelPeriod.EndDate).Before(endDate) && DateV1ToTime(travelPeriod.StartDate).Before(DateV1ToTime(travelPeriod.EndDate))
}

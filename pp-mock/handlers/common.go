// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"time"

	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
)

const DefaultPrice = "100"

var (
	allowedTimePeriodStart = time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC)
	allowedTimePeriodEnd   = time.Date(2025, time.June, 30, 0, 0, 0, 0, time.UTC)
)

func DateV1ToTime(date *typesv1.Date) time.Time {
	return time.Date(int(date.GetYear()), time.Month(date.GetMonth()), int(date.GetDay()), 0, 0, 0, 0, time.UTC)
}

// only period between 01.06.2025 and 30.06.2025 is allowed - represents available period for the booking
func IsTravelPeriodAllowed(travelPeriod *typesv1.TravelPeriod) bool {
	startDate := DateV1ToTime(travelPeriod.GetStartDate())
	endDate := DateV1ToTime(travelPeriod.GetEndDate())

	return !startDate.Before(allowedTimePeriodStart) && !endDate.After(allowedTimePeriodEnd)
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"time"

	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"google.golang.org/protobuf/proto"
)

const DefaultPricePerNight = 105.33

func DateV1ToTime(date *typesv1.Date) time.Time {
	return time.Date(int(date.GetYear()), time.Month(date.GetMonth()), int(date.GetDay()), 0, 0, 0, 0, time.UTC)
}

func TimeToDateV1(time time.Time) *typesv1.Date {
	return &typesv1.Date{
		Year:  int32(time.Year()),  //nolint:gosec
		Month: int32(time.Month()), //nolint:gosec
		Day:   int32(time.Day()),   //nolint:gosec
	}
}

// only period between now + 60 days is allowed for bookings
func IsTravelPeriodAllowed(travelPeriod *typesv1.TravelPeriod) bool {
	startDate := time.Now()
	endDate := time.Now().Add(time.Hour * 24 * 60) // 60 days from now

	return DateV1ToTime(travelPeriod.StartDate).After(startDate) && DateV1ToTime(travelPeriod.EndDate).Before(endDate) && DateV1ToTime(travelPeriod.StartDate).Before(DateV1ToTime(travelPeriod.EndDate))
}

func AreTravelDatesValid(departureDate, arrivalDate *typesv1.Date) bool {
	if departureDate == nil || arrivalDate == nil {
		return false
	}

	// Fail if departure is after arrival
	return !DateV1ToTime(departureDate).After(DateV1ToTime(arrivalDate))
}

// GetTravellerIDsV1 extracts traveller IDs from []*typesv1.BasicTraveller
func GetTravellerIDsV1(travellers []*typesv1.BasicTraveller) []int32 {
	ids := make([]int32, len(travellers))
	for i, traveller := range travellers {
		ids[i] = traveller.TravellerId
	}
	return ids
}

// GetTravellerIDsV2 extracts traveller IDs from []*typesv2.BasicTraveller
func GetTravellerIDsV2(travellers []*typesv2.BasicTraveller) []int32 {
	ids := make([]int32, len(travellers))
	for i, traveller := range travellers {
		ids[i] = traveller.TravellerId
	}
	return ids
}

// GetTravellerIDsV3 extracts traveller IDs from []*typesv3.BasicTraveller
func GetTravellerIDsV3(travellers []*typesv3.BasicTraveller) []int32 {
	ids := make([]int32, len(travellers))
	for i, traveller := range travellers {
		ids[i] = traveller.TravellerId
	}
	return ids
}

func CloneProtoSlice[T proto.Message](source []T) []T {
	clone := make([]T, len(source))
	for i, elem := range source {
		clone[i] = proto.Clone(elem).(T)
	}
	return clone
}

func CloneProto[T proto.Message](source T) T {
	return proto.Clone(source).(T)
}

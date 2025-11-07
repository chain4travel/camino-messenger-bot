// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"math/big"
	"time"

	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/price"
	"google.golang.org/protobuf/proto"
)

const (
	DefaultPricePerNight         int64 = 105_33 // 105.33 with 2 decimals
	DefaultPricePerNightStr            = "10533"
	DefaultPricePerNightDecimals       = 2
	BookingTokenPriceValue             = "1"
	FreeCancellationDuration           = 7 * 24 * time.Hour
	CancellationPolicyID               = "pp-mock-full-refund"
)

var DefaultPricePerNightNativeTokenBig *big.Int

func init() {
	var err error
	DefaultPricePerNightNativeTokenBig, err = price.ToBigInt(DefaultPricePerNightStr, DefaultPricePerNightDecimals, price.NativeTokenDecimals)
	if err != nil {
		panic("failed to convert default price per night to big.Int: " + err.Error())
	}
}

func DateV1ToTime(date *typesv1.Date) time.Time {
	return time.Date(int(date.GetYear()), time.Month(date.GetMonth()), int(date.GetDay()), 0, 0, 0, 0, time.UTC)
}

func DateV4ToTime(date *typesv4.Date) time.Time {
	return time.Date(int(date.GetYear()), time.Month(date.GetMonth()), int(date.GetDay()), 0, 0, 0, 0, time.UTC)
}

func DaysBetweenDatesV1(endDate, startDate *typesv1.Date) int64 {
	duration := DateV1ToTime(endDate).Sub(DateV1ToTime(startDate))
	return int64(duration / (time.Hour * 24))
}

func DaysBetweenDatesV4(endDate, startDate *typesv4.Date) int64 {
	duration := DateV4ToTime(endDate).Sub(DateV4ToTime(startDate))
	return int64(duration / (time.Hour * 24))
}

func TimeToDateV1(time time.Time) *typesv1.Date {
	return &typesv1.Date{
		Year:  int32(time.Year()),  //nolint:gosec
		Month: int32(time.Month()), //nolint:gosec
		Day:   int32(time.Day()),   //nolint:gosec
	}
}

func TimeToDateV4(time time.Time) *typesv4.Date {
	return &typesv4.Date{
		Year:  uint32(time.Year()),  //nolint:gosec
		Month: uint32(time.Month()), //nolint:gosec
		Day:   uint32(time.Day()),   //nolint:gosec
	}
}

// only period between now + 60 days is allowed for bookings
func IsTravelPeriodAllowedV1(travelPeriod *typesv1.TravelPeriod) bool {
	startDate := time.Now()
	endDate := time.Now().Add(time.Hour * 24 * 60) // 60 days from now

	return DateV1ToTime(travelPeriod.StartDate).After(startDate) && DateV1ToTime(travelPeriod.EndDate).Before(endDate) && DateV1ToTime(travelPeriod.StartDate).Before(DateV1ToTime(travelPeriod.EndDate))
}

// only period between now + 60 days is allowed for bookings
func IsTravelPeriodAllowedV4(travelPeriod *typesv4.TravelPeriod) bool {
	startDate := time.Now()
	endDate := time.Now().Add(time.Hour * 24 * 60) // 60 days from now

	return DateV4ToTime(travelPeriod.StartDate).After(startDate) && DateV4ToTime(travelPeriod.EndDate).Before(endDate) && DateV4ToTime(travelPeriod.StartDate).Before(DateV4ToTime(travelPeriod.EndDate))
}

func AreTravelDatesValidV1(departureDate, arrivalDate *typesv1.Date) bool {
	if departureDate == nil || arrivalDate == nil {
		return false
	}

	// Fail if departure is after arrival
	return !DateV1ToTime(departureDate).After(DateV1ToTime(arrivalDate))
}

// Fail if departure is after arrival
func AreTravelDatesValidV4(departureDate, arrivalDate *typesv4.Date) bool {
	return !DateV4ToTime(departureDate).After(DateV4ToTime(arrivalDate))
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

// GetTravellerIDsV4 extracts traveller IDs from []*typesv4.BasicTraveller
func GetTravellerIDsV4(travellers []*typesv4.BasicTraveller) []uint32 {
	ids := make([]uint32, len(travellers))
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

func SuccessHeaderV4() *typesv4.ResponseHeader {
	return &typesv4.ResponseHeader{
		BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
		Status:     typesv4.StatusType_STATUS_TYPE_SUCCESS,
	}
}

func ErrorHeaderV4(message string) *typesv4.ResponseHeader {
	return &typesv4.ResponseHeader{
		BaseHeader: &typesv4.Header{Version: &typesv4.Version{}},
		Status:     typesv4.StatusType_STATUS_TYPE_FAILURE,
		Alerts:     []*typesv4.Alert{{Message: message, Type: typesv4.AlertType_ALERT_TYPE_ERROR}},
	}
}

func AddHeaderErrorV4(header *typesv4.ResponseHeader, message string) {
	header.Alerts = append(header.Alerts, &typesv4.Alert{
		Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
		Message: message,
	})
	header.Status = typesv4.StatusType_STATUS_TYPE_FAILURE
}

func AddHeaderWarningV4(header *typesv4.ResponseHeader, message string) {
	header.Alerts = append(header.Alerts, &typesv4.Alert{
		Type:    typesv4.AlertType_ALERT_TYPE_WARNING,
		Message: message,
	})
}

func AddHeaderInfoV4(header *typesv4.ResponseHeader, message string) {
	header.Alerts = append(header.Alerts, &typesv4.Alert{
		Type:    typesv4.AlertType_ALERT_TYPE_INFO,
		Message: message,
	})
}

func ProtoSlicesEqual[T proto.Message](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !proto.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

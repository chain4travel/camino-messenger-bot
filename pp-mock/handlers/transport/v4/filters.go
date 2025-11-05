// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"time"

	transportv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data/transport"
	"google.golang.org/protobuf/proto"
)

func filterTripsBasicByModifiedAfter(
	trips []*transportv4.TripBasic,
	lastModified time.Time,
) []*transportv4.TripBasic {
	filtered := []*transportv4.TripBasic{}
	for _, trip := range trips {
		if !trip.LastModified.AsTime().Before(lastModified) {
			filtered = append(filtered, common.CloneProto(trip))
		}
	}
	return filtered
}

func filterTripsBySupplierCodes(trips []*transport.TripV4, supplierCodes []*typesv4.SupplierProductCode) []*transport.TripV4 {
	if len(supplierCodes) == 0 {
		return transport.CloneV4(trips)
	}

	filtered := []*transport.TripV4{}
	for _, trip := range trips {
		for _, code := range supplierCodes {
			if proto.Equal(trip.Basic.SupplierCode, code) {
				filtered = append(filtered, trip.Clone())
				break
			}
		}
	}
	return filtered
}

func filterTripsByMaxSegments(trips []*transport.TripV4, maxSegments uint32) []*transport.TripV4 {
	if maxSegments == 0 {
		return transport.CloneV4(trips)
	}

	filtered := []*transport.TripV4{}
	for _, trip := range trips {
		if len(trip.Basic.Segments) <= int(maxSegments) {
			filtered = append(filtered, trip.Clone())
		}
	}
	return filtered
}

// Filter the trips by dates -- note that it checks the first segment's departure date and the last segment's arrival date.
func filterTripsByDates(trips []*transport.TripV4, query *transportv4.QueryTrip) []*transport.TripV4 {
	filtered := []*transport.TripV4{}
	for _, trip := range trips {
		// We need first segment to compare departure date and last segment to compare arrival date
		tripDepartureDate := common.TimeToDateV4(trip.Basic.Segments[0].Departure.DateTime.AsTime())
		tripArrivalDate := common.TimeToDateV4(trip.Basic.Segments[len(trip.Basic.Segments)-1].Arrival.DateTime.AsTime())

		if proto.Equal(tripDepartureDate, query.Departure.Date) && proto.Equal(tripArrivalDate, query.Arrival.Date) {
			filtered = append(filtered, trip.Clone())
		}
	}
	return filtered
}

// Filter the trips by locations -- note that it checks the first segment's departure location and the last segment's arrival location.
func filterTripsByLocations(trips []*transport.TripV4, query *transportv4.QueryTrip) []*transport.TripV4 {
	queryDepartureLocationCodes := query.Departure.Location.GetLocationCodes()
	queryArrivalLocationCodes := query.Arrival.Location.GetLocationCodes()

	filtered := []*transport.TripV4{}
	for _, trip := range trips {
		// We need the first segment to compare the departure location
		firstSegmentDepartureLocationCode := trip.Basic.Segments[0].Departure.Location.GetLocationCode()

		foundDepartureMatch := false
		for _, queryDepartureLocationCode := range queryDepartureLocationCodes.Codes {
			if proto.Equal(queryDepartureLocationCode, firstSegmentDepartureLocationCode) {
				foundDepartureMatch = true
				break
			}
		}

		if !foundDepartureMatch {
			continue
		}

		// We need the last segment to compare the arrival location
		lastSegmentArrivalLocationCode := trip.Basic.Segments[len(trip.Basic.Segments)-1].Arrival.Location.GetLocationCode()

		for _, queryArrivalLocationCode := range queryArrivalLocationCodes.Codes {
			if proto.Equal(queryArrivalLocationCode, lastSegmentArrivalLocationCode) {
				filtered = append(filtered, trip.Clone())
				break
			}
		}
	}
	return filtered
}

func filterTripsByCurrency(trips []*transport.TripV4, currency *typesv4.Currency) []*transport.TripV4 {
	filtered := []*transport.TripV4{}
	for _, trip := range trips {
		if proto.Equal(trip.Extended.Price.Currency, currency) {
			filtered = append(filtered, trip.Clone())
		}
	}
	return filtered
}

// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v5

import (
	"time"

	transportv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v5"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/services/data/transport"
	"google.golang.org/protobuf/proto"
)

func filterTripsBasicByModifiedAfter(
	trips []*transportv5.TripBasic,
	lastModified time.Time,
) []*transportv5.TripBasic {
	filtered := []*transportv5.TripBasic{}
	for _, trip := range trips {
		if !trip.LastModified.AsTime().Before(lastModified) {
			filtered = append(filtered, common.CloneProto(trip))
		}
	}
	return filtered
}

func filterTripsBySupplierCodes(trips []*transport.TripV5, supplierCodes []*typesv4.SupplierProductCode) []*transport.TripV5 {
	if len(supplierCodes) == 0 {
		return transport.CloneV5(trips)
	}

	filtered := []*transport.TripV5{}
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

func filterTripsByMaxSegments(trips []*transport.TripV5, maxSegments uint32) []*transport.TripV5 {
	if maxSegments == 0 {
		return transport.CloneV5(trips)
	}

	filtered := []*transport.TripV5{}
	for _, trip := range trips {
		if len(trip.Basic.Segments) <= int(maxSegments) {
			filtered = append(filtered, trip.Clone())
		}
	}
	return filtered
}

// Filter the trips by dates -- note that it checks the first segment's departure date and the last segment's arrival date.
func filterTripsByDates(trips []*transport.TripV5, query *transportv5.QueryTrip) []*transport.TripV5 {
	filtered := []*transport.TripV5{}
	for _, trip := range trips {
		if len(trip.Basic.Segments) == 0 {
			continue
		}
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
func filterTripsByLocations(trips []*transport.TripV5, query *transportv5.QueryTrip) []*transport.TripV5 {
	queryDepartureLocationCodes := query.Departure.Location.GetLocationCodes()
	queryArrivalLocationCodes := query.Arrival.Location.GetLocationCodes()

	filtered := []*transport.TripV5{}
	for _, trip := range trips {
		if len(trip.Basic.Segments) == 0 {
			continue
		}
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

func filterTripsByCurrency(trips []*transport.TripV5, currency *typesv4.Currency) []*transport.TripV5 {
	if currency == nil {
		return transport.CloneV5(trips)
	}

	filtered := []*transport.TripV5{}
	for _, trip := range trips {
		if proto.Equal(trip.Extended.Price.Currency, currency) {
			filtered = append(filtered, trip.Clone())
		}
	}
	return filtered
}

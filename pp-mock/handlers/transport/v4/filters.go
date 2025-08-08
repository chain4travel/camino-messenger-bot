// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v4

import (
	"time"

	transportv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"google.golang.org/protobuf/proto"
)

func filterTripsByProductCodes(trips []*transportv4.TripExtended, productCodes []*typesv4.ProductCode) []*transportv4.TripExtended {
	if len(productCodes) == 0 {
		return common.CloneProtoSlice(trips)
	}

	filtered := []*transportv4.TripExtended{}
	for _, trip := range trips {
	segmentsLoop:
		for _, segment := range trip.Segments {
			for _, code := range productCodes {
				if proto.Equal(segment.GetInfo().GetProductCode(), code) {
					filtered = append(filtered, common.CloneProto(trip))
					break segmentsLoop
				}
			}
		}
	}
	return filtered
}

func filterTripsByMaxSegments(trips []*transportv4.TripExtended, maxSegments uint32) []*transportv4.TripExtended {
	filtered := []*transportv4.TripExtended{}
	for _, trip := range trips {
		if len(trip.Segments) <= int(maxSegments) {
			filtered = append(filtered, common.CloneProto(trip))
		}
	}
	return filtered
}

// Returns properties that have been modified not before [lastModified].
func filterPropertiesByLastModified(
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

// Filter the trips by dates -- note that it checks the first segment's departure date and the last segment's arrival date.
func filterTripsByDates(
	trips []*transportv4.TripExtended,
	query *transportv4.QueryTrip,
) []*transportv4.TripExtended {
	filtered := []*transportv4.TripExtended{}
	queryDepartureDate := common.DateV4ToTime(query.Departure.Date)

	// TODO @Noctunus - All assumptions that the fields are present.
	// This needs to be validated. Ideally with protovalidate on the unmashalled mockdata.
	for _, trip := range trips {
		if len(trip.Segments) == 0 {
			continue
		}

		// We need the first segment to compare the departure date
		firstSegment := trip.Segments[0]
		firstSegmentDepartureDateTime := time.Unix(firstSegment.Info.Departure.DateTime.Seconds, 0)
		firstSegmentDepartureDate := firstSegmentDepartureDateTime.Truncate(24 * time.Hour)

		// We need the last segment to compare the arrival date
		lastSegment := trip.Segments[len(trip.Segments)-1]
		if query.Arrival != nil && query.Arrival.Date != nil {
			queryArrivalDate := common.DateV4ToTime(query.Arrival.Date)
			lastSegmentArrivalDateTime := time.Unix(lastSegment.Info.Arrival.DateTime.Seconds, 0)
			lastSegmentArrivalDate := lastSegmentArrivalDateTime.Truncate(24 * time.Hour)
			// Now we can compare if the trip dates are exactly what the query is looking for
			if firstSegmentDepartureDate.Equal(queryDepartureDate) && lastSegmentArrivalDate.Equal(queryArrivalDate) {
				filtered = append(filtered, common.CloneProto(trip))
			}
		} else if firstSegmentDepartureDate.Equal(queryDepartureDate) {
			filtered = append(filtered, common.CloneProto(trip))
		}
	}
	return filtered
}

// Filter the trips by locations -- note that it checks the first segment's departure location and the last segment's arrival location.
func filterTripsByLocations(
	trips []*transportv4.TripExtended,
	query *transportv4.QueryTrip,
) []*transportv4.TripExtended {
	filtered := []*transportv4.TripExtended{}
	// TODO @Noctunus - All assumptions that the fields are present.
	// This needs to be validated. Ideally with protovalidate on the unmashalled mockdata.
	for _, trip := range trips {
		if len(trip.Segments) == 0 {
			continue
		}
		// We need the first segment to compare the departure location
		firstSegment := trip.Segments[0]
		firstSegmentDepartureLocationCode := firstSegment.Info.Departure.Location.GetLocationCode()

		if firstSegmentDepartureLocationCode == nil {
			continue
		}

		queryDepartureLocationCodes := query.Departure.Location.GetLocationCodes()
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

		// If arrival location is specified in the query, check for a match
		if query.Arrival != nil && query.Arrival.Location != nil && query.Arrival.Location.HasLocationCodes() {
			lastSegment := trip.Segments[len(trip.Segments)-1]
			lastSegmentArrivalLocationCode := lastSegment.Info.Arrival.Location.GetLocationCode()

			if lastSegmentArrivalLocationCode == nil {
				continue
			}

			queryArrivalLocationCodes := query.Arrival.Location.GetLocationCodes()
			for _, queryArrivalLocationCode := range queryArrivalLocationCodes.Codes {
				if proto.Equal(queryArrivalLocationCode, lastSegmentArrivalLocationCode) {
					filtered = append(filtered, common.CloneProto(trip))
					break
				}
			}
		} else {
			// If no arrival location is specified, add the trip if departure matches
			filtered = append(filtered, common.CloneProto(trip))
		}
	}
	return filtered
}

func filterTripsByCurrency(trips []*transportv4.TripExtended, currency *typesv4.Currency) []*transportv4.TripExtended {
	filtered := []*transportv4.TripExtended{}
	for _, trip := range trips {
		if proto.Equal(trip.Price.Currency, currency) {
			filtered = append(filtered, common.CloneProto(trip))
		}
	}
	return filtered
}

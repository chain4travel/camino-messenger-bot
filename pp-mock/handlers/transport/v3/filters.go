// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"time"

	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"google.golang.org/protobuf/proto"
)

func filterTripsByProductCodes(trips []*transportv3.TripExtended, productCodes []*typesv2.ProductCode) []*transportv3.TripExtended {
	if len(productCodes) == 0 {
		return common.CloneProtoSlice(trips)
	}

	filtered := []*transportv3.TripExtended{}
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

func filterTripsByMaxSegments(trips []*transportv3.TripExtended, maxSegments int32) []*transportv3.TripExtended {
	filtered := []*transportv3.TripExtended{}
	for _, trip := range trips {
		if len(trip.Segments) <= int(maxSegments) {
			filtered = append(filtered, common.CloneProto(trip))
		}
	}
	return filtered
}

// Returns properties that have been modified not before [lastModified].
func filterPropertiesByLastModified(
	trips []*transportv3.TripBasic,
	lastModified time.Time,
) []*transportv3.TripBasic {
	filtered := []*transportv3.TripBasic{}
	for _, trip := range trips {
		if !trip.LastModified.AsTime().Before(lastModified) {
			filtered = append(filtered, common.CloneProto(trip))
		}
	}
	return filtered
}

// Filter the trips by dates -- note that it checks the first segment's departure date and the last segment's arrival date.
func filterTripsByDates(
	trips []*transportv3.TripExtended,
	query *transportv3.QueryTrip,
) []*transportv3.TripExtended {
	filtered := []*transportv3.TripExtended{}
	queryDepartureDate := common.DateV1ToTime(query.Departure.Date)

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
			queryArrivalDate := common.DateV1ToTime(query.Arrival.Date)
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
	trips []*transportv3.TripExtended,
	query *transportv3.QueryTrip,
) []*transportv3.TripExtended {
	filtered := []*transportv3.TripExtended{}
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

func filterTripsByCurrency(trips []*transportv3.TripExtended, currency *typesv3.Currency) []*transportv3.TripExtended {
	filtered := []*transportv3.TripExtended{}
	for _, trip := range trips {
		if proto.Equal(trip.Price.Currency, currency) {
			filtered = append(filtered, common.CloneProto(trip))
		}
	}
	return filtered
}

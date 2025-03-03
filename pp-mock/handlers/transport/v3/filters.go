// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	common "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers"
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

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	transportv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	common "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers"
	"google.golang.org/protobuf/proto"
)

func filterTripsByProductCodes(trips []*transportv1.Trip, productCodes []*typesv1.ProductCode) []*transportv1.Trip {
	if len(productCodes) == 0 {
		return common.CloneProtoSlice(trips)
	}

	filtered := []*transportv1.Trip{}
	for _, trip := range trips {
	segmentsLoop:
		for _, segment := range trip.Segments {
			for _, code := range productCodes {
				if proto.Equal(segment.GetProductCode(), code) {
					filtered = append(filtered, common.CloneProto(trip))
					break segmentsLoop
				}
			}
		}
	}
	return filtered
}

func filterTripsByMaxSegments(trips []*transportv1.Trip, maxSegments int32) []*transportv1.Trip {
	filtered := []*transportv1.Trip{}
	for _, trip := range trips {
		if len(trip.Segments) <= int(maxSegments) {
			filtered = append(filtered, common.CloneProto(trip))
		}
	}
	return filtered
}

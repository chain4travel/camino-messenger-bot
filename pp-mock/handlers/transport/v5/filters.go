// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v5

import (
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/services/data/transport"
	"google.golang.org/protobuf/proto"
)

func filterTripsByCurrency(trips []*transport.TripV4, currency *typesv4.Currency) []*transport.TripV4 {
	if currency == nil {
		return transport.CloneV4(trips)
	}

	filtered := []*transport.TripV4{}
	for _, trip := range trips {
		if proto.Equal(trip.Extended.Price.Currency, currency) {
			filtered = append(filtered, trip.Clone())
		}
	}
	return filtered
}

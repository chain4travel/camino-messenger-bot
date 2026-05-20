// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package transport

import (
	"fmt"

	transportv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v5"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	"google.golang.org/protobuf/proto"
)

type TripV5 struct {
	Basic    *transportv5.TripBasic
	Extended *transportv5.TripExtended
}

func (t *TripV5) Clone() *TripV5 {
	return &TripV5{
		Basic:    common.CloneProto(t.Basic),
		Extended: common.CloneProto(t.Extended),
	}
}

func (t *TripV5) Verify() error {
	if !proto.Equal(t.Basic.SupplierCode, t.Extended.SupplierCode) {
		return fmt.Errorf("trip basic and extended supplier code mismatch")
	}
	if len(t.Basic.Segments) != len(t.Extended.Segments) {
		return fmt.Errorf("trip basic and extended segments length mismatch")
	}
	for i, segment := range t.Basic.Segments {
		if !segment.Departure.Location.HasLocationCode() || !segment.Arrival.Location.HasLocationCode() {
			return fmt.Errorf("trip segment %d missing departure or arrival location code", i)
		}
	}
	return nil
}

func VerifyAndGetTripsV5(basic []*transportv5.TripBasic, extended []*transportv5.TripExtended) []*TripV5 {
	if len(basic) != len(extended) {
		panic(fmt.Errorf("mock data error: number of transport v5 basic trips (%d) does not match number of extended trips (%d)", len(basic), len(extended)))
	}
	trips := make([]*TripV5, 0, len(basic))
	for i, tripBasic := range basic {
		trip := &TripV5{
			Basic:    tripBasic,
			Extended: extended[i],
		}
		if err := trip.Verify(); err != nil {
			panic(fmt.Errorf("mock data error: trip basic/extended at index %d are invalid: %w", i, err))
		}
		trips = append(trips, trip)
	}
	return trips
}

func CloneV5(trips []*TripV5) []*TripV5 {
	cloned := make([]*TripV5, len(trips))
	for i, trip := range trips {
		cloned[i] = trip.Clone()
	}
	return cloned
}

func BasicV5(trips []*TripV5) []*transportv5.TripBasic {
	basic := make([]*transportv5.TripBasic, len(trips))
	for i, trip := range trips {
		basic[i] = common.CloneProto(trip.Basic)
	}
	return basic
}

func ExtendedV5(trips []*TripV5) []*transportv5.TripExtended {
	extended := make([]*transportv5.TripExtended, len(trips))
	for i, trip := range trips {
		extended[i] = common.CloneProto(trip.Extended)
	}
	return extended
}

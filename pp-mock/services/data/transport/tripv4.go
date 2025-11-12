// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package transport

import (
	"fmt"

	transportv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"google.golang.org/protobuf/proto"
)

type TripV4 struct {
	Basic    *transportv4.TripBasic
	Extended *transportv4.TripExtended
}

func (t *TripV4) Clone() *TripV4 {
	return &TripV4{
		Basic:    common.CloneProto(t.Basic),
		Extended: common.CloneProto(t.Extended),
	}
}

func (t *TripV4) Verify() error {
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

func VerifyAndGetTrips(basic []*transportv4.TripBasic, extended []*transportv4.TripExtended) []*TripV4 {
	if len(basic) != len(extended) {
		panic(fmt.Errorf("mock data error: number of transport v4 basic trips (%d) does not match number of extended trips (%d)", len(basic), len(extended)))
	}
	trips := make([]*TripV4, 0, len(basic))
	for i, tripBasic := range basic {
		trip := &TripV4{
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

func CloneV4(trips []*TripV4) []*TripV4 {
	cloned := make([]*TripV4, len(trips))
	for i, trip := range trips {
		cloned[i] = trip.Clone()
	}
	return cloned
}

func BasicV4(trips []*TripV4) []*transportv4.TripBasic {
	basic := make([]*transportv4.TripBasic, len(trips))
	for i, trip := range trips {
		basic[i] = common.CloneProto(trip.Basic)
	}
	return basic
}

func ExtendedV4(trips []*TripV4) []*transportv4.TripExtended {
	extended := make([]*transportv4.TripExtended, len(trips))
	for i, trip := range trips {
		extended[i] = common.CloneProto(trip.Extended)
	}
	return extended
}

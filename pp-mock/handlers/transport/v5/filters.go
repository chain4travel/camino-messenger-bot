// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v5

import (
	transportv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v5"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data/transport"
	"google.golang.org/protobuf/proto"
)

// Filters trips based on product codes
func filterTripsBySearchParameters(
	trips []*transport.TripV4,
	params *transportv5.TransportSearchParameters,
) []*transport.TripV4 {
	if params == nil || len(params.GetTripSupplierCodes()) == 0 {
		return transport.CloneV4(trips)
	}

	filtered := []*transport.TripV4{}
	for _, trip := range trips {
		if len(params.GetTripSupplierCodes()) > 0 {
			match := false
			for _, supplierCode := range params.GetTripSupplierCodes() {
				if proto.Equal(trip.Basic.SupplierCode, supplierCode) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, trip.Clone())
	}
	return filtered
}

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

func filterTripsBasicBySupplierCodes(trips []*transportv5.TripBasic, supplierCodes []*typesv4.SupplierProductCode) []*transportv5.TripBasic {
	if len(supplierCodes) == 0 {
		return common.CloneProtoSlice(trips)
	}

	filtered := []*transportv5.TripBasic{}
	for _, trip := range trips {
		for _, code := range supplierCodes {
			if proto.Equal(trip.SupplierCode, code) {
				filtered = append(filtered, common.CloneProto(trip))
				break
			}
		}
	}
	return filtered
}

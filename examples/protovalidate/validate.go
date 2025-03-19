// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

// Test protovalidate for evm address
package main

import (
	"log"

	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/bufbuild/protovalidate-go"
	"go.uber.org/zap"
)

func main() {

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()
	sugar := logger.Sugar()

	sugar.Info("starting protovalidate example...")

	// ------------------------------------------------------------------------
	// EVM ADDRESS
	// ------------------------------------------------------------------------
	evmAddress1 := typesv3.EVMAddress{
		// Length is lower then 42 and is not a valid address because has "x" at the end (non-hex character)
		Address: "0xd41786599F2B225A5A1eA35cDc4A2a6Fa9E92ex",
	}

	sugar.Info("EVM Address: ", &evmAddress1)

	// Validate
	if err := protovalidate.Validate(&evmAddress1); err != nil {
		sugar.Errorf("validation failed:", err)
	} else {
		sugar.Info("validation succeeded")
	}

	// ------------------------------------------------------------------------
	// BASIC TRAVELLER
	// ------------------------------------------------------------------------
	basicTraveller := typesv3.BasicTraveller{
		TravellerId: -1,
		Type:        typesv3.TravellerType_TRAVELLER_TYPE_UNSPECIFIED, // 0
		// Birthdate: &typesv1.Date{
		// 	Year:  1980,
		// 	Month: 1,
		// 	Day:   1,
		// },
		//Nationality: typesv2.Country_COUNTRY_DE,
	}

	sugar.Info("BasicTraveller: ", &basicTraveller)

	// Validate
	if err := protovalidate.Validate(&basicTraveller); err != nil {
		sugar.Errorf("validation failed:", err)
	} else {
		sugar.Info("validation succeeded")
	}

	// ------------------------------------------------------------------------
	// EXTENSIVE TRAVELLER
	// ------------------------------------------------------------------------
	extensiveTraveller := typesv3.ExtensiveTraveller{
		TravellerId: -1,
	}

	sugar.Info("ExtensiveTraveller: ", &extensiveTraveller)

	// Validate
	if err := protovalidate.Validate(&extensiveTraveller); err != nil {
		sugar.Errorf("validation failed:", err)
	} else {
		sugar.Info("validation succeeded")
	}

}

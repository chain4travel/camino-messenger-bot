// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import "time"

var BuyableUntilDefault time.Duration

func SetE2EDefaults() {
	BuyableUntilDefault = 3 * time.Second
}

func SetDefaults() {
	BuyableUntilDefault = 5 * time.Minute
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package conversion

import (
	"fmt"
	"math"
)

// Safely converts an int to int32, returning an error if out of range.
func IntToInt32(value int) (int32, error) {
	if value < 0 || value > math.MaxInt32 {
		return 0, fmt.Errorf("value out of range for int32: %d", value)
	}
	return int32(value), nil // nolint:gosec
}

// Safely converts an int to uint32, panicking with error if out of range.
func MustIntToUInt32(value int) uint32 {
	if value < 0 || value > math.MaxUint32 {
		panic(fmt.Errorf("value out of range for uint32: %d", value))
	}
	return uint32(value) // nolint:gosec
}

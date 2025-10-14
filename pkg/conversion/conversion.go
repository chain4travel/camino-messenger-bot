// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package conversion

import (
	"fmt"
	"math"
	"time"
)

// Safely converts an int to int32, returning an error if out of range.
func IntToInt32(value int) (int32, error) {
	if value < 0 || value > math.MaxInt32 {
		return 0, fmt.Errorf("value out of range for int32: %d", value)
	}
	return int32(value), nil // nolint:gosec
}

// Safely converts an uint32 to int32, panicking with error if out of range.
func MustUInt32ToInt32(value uint32) int32 {
	if value > math.MaxInt32 {
		panic(fmt.Errorf("value out of range for int32: %d", value))
	}
	return int32(value) // nolint:gosec
}

// Safely converts an int32 to uint32, panicking with error if out of range.
func MustInt32ToUInt32(value int32) uint32 {
	if value < 0 {
		panic(fmt.Errorf("value out of range for uint32: %d", value))
	}
	return uint32(value) // nolint:gosec
}

// Safely converts an int to uint32, panicking with error if out of range.
func MustIntToUInt32(value int) uint32 {
	if value < 0 || value > math.MaxUint32 {
		panic(fmt.Errorf("value out of range for uint32: %d", value))
	}
	return uint32(value) // nolint:gosec
}

// Safely converts an int to uint16, panicking with error if out of range.
func MustIntToUInt16(value int) uint16 {
	if value < 0 || value > math.MaxUint16 {
		panic(fmt.Errorf("value out of range for uint16: %d", value))
	}
	return uint16(value) // nolint:gosec
}

// Safely converts an uint64 to int64, panicking with error if out of range.
func MustUInt64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		panic(fmt.Errorf("value out of range for int64: %d", value))
	}
	return int64(value) // nolint:gosec
}

// Safely converts an int64 to uint64, panicking with error if out of range.
func MustInt64ToUInt64(value int64) uint64 {
	if value < 0 {
		panic(fmt.Errorf("value out of range for uint64: %d", value))
	}
	return uint64(value) // nolint:gosec
}

// Safely converts an int64 to uint64, panicking with error if out of range.
func MustDurationToUInt64(value time.Duration) uint64 {
	if value < 0 {
		panic(fmt.Errorf("value out of range for uint64: %d", value))
	}
	return uint64(value) // nolint:gosec
}

// Safely converts an int32 to uint64, panicking with error if out of range.
func MustInt32ToUInt64(value int32) uint64 {
	if value < 0 {
		panic(fmt.Errorf("value out of range for uint64: %d", value))
	}
	return uint64(value) // nolint:gosec
}

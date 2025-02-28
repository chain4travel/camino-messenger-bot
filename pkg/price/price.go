// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package price

import (
	"fmt"
	"math/big"
)

const (
	NativeTokenDecimals = int32(18)
	ISODecimals         = int32(6)
)

// Converts the price to its integer representation
func ToBigInt(value string, decimals int32, totalDecimals int32) (*big.Int, error) {
	// Validate decimal parameters
	if decimals > totalDecimals {
		return nil, fmt.Errorf("decimals (%d) > (%d) totalDecimals", decimals, totalDecimals)
	}

	// Convert the value string to a big.Int
	valueBigInt, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("failed to convert value to big.Int: %s", value)
	}

	// Calculate the multiplier as 10^(totalDecimals - price.Decimals)
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(totalDecimals-decimals)), nil)

	return valueBigInt.Mul(valueBigInt, multiplier), nil
}

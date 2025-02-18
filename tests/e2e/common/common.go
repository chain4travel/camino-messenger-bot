// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"math/big"

	"github.com/chain4travel/caminogoeth-compat/caminogo/units"
)

const x2cRate = 1_000_000_000

var (
	X2CRateBig                 = big.NewInt(x2cRate)
	CAM                        = big.NewInt(0).Mul(big.NewInt(0).SetUint64(units.Avax), X2CRateBig)
	DefaultCMAccountOwnerFunds = big.NewInt(0).Mul(CAM, big.NewInt(100))
)

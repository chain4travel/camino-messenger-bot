// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"math/big"
	"testing"
	"time"

	"github.com/chain4travel/caminogoeth-compat/caminogo/units"
	"github.com/stretchr/testify/require"
)

const MinBuyableUntilInContract = 1 // seconds -- overrides the bookingtoken default of 1 minute when deployed for the e2e test

var (
	X2CRateBig                 = big.NewInt(1_000_000_000)
	CAM                        = big.NewInt(0).Mul(big.NewInt(0).SetUint64(units.Avax), X2CRateBig)
	DefaultCMAccountOwnerFunds = big.NewInt(0).Mul(CAM, big.NewInt(100))
)

func AwaitError(t *testing.T, errChan chan error, errContent string, timeout time.Duration) {
	t.Helper()

	select {
	case err := <-errChan:
		require.Error(t, err)
		require.Contains(t, err.Error(), errContent)
	case <-time.After(timeout):
		require.Fail(t, "timeout waiting for channel error with content: "+errContent)
	}
}

func ExpectNoErrorAsync(t *testing.T, errChan chan error) {
	t.Helper()
	go func() {
		require.NoError(t, <-errChan)
	}()
}

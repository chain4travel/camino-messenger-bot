// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-matrix-app-service/config"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/matrix"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/suite"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert" //nolint:depguard // we don't user assert's assertions, we use assert.CollectT type as needed in require pkg
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestCashIn)(nil)

func init() {
	// Test is deactivated temporarily, because it uses pre-erc20 ASB that depends on pre-erc20 CMB.
	// In order to update ASB, we need to merge ASB first. After that we can re-activate this test.
	// Tests["PeriodicCashIn"] = &TestCashIn{}
}

type TestCashIn struct {
	*suite.Environment

	cashInPeriodSeconds   int64
	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
	pingFee               int64
}

func (tt *TestCashIn) Setup(e *suite.Environment) {
	tt.Environment = e
	tt.cashInPeriodSeconds = 10 // 10s so we can test in reasonable time

	e.ASBOptions = []matrix.ASBOption{
		matrix.WithCashInPeriod(tt.cashInPeriodSeconds),
	}
}

func (tt *TestCashIn) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("Ping", func(t *testing.T) {
		tt.testPeriodicCashInWithPingV1(ctx, t)
	})
}

func (tt *TestCashIn) prepare(ctx context.Context, t *testing.T) {
	// Register all the services needed for the tests
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx, botGenerated.PingServiceV1))

	tt.pingFee = 5_000_000_000_000_000

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{{Name: botGenerated.PingServiceV1, Fee: tt.pingFee}}),
		bot.WithCashInPeriod(tt.cashInPeriodSeconds), // cash-in every 10 seconds
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil,
		// has nothing to cash in, so we'll just check that nothing unexpected happens
		bot.WithCashInPeriod(tt.cashInPeriodSeconds), // cash-in every 10 seconds
	)
}

func (tt *TestCashIn) testPeriodicCashInWithPingV1(ctx context.Context, t *testing.T) {
	initialDistributorBalance, err := tt.CaminoNetwork.Client.BalanceOf(ctx, tt.distributorBot.CMAccountAddress())
	require.NoError(t, err)

	initialSupplierBalance, err := tt.CaminoNetwork.Client.BalanceOf(ctx, tt.supplierBot.CMAccountAddress())
	require.NoError(t, err)

	initialASBBalance, err := tt.CaminoNetwork.Client.BalanceOf(ctx, tt.ASB.NetworkFeeRecipientCMAccountAddress())
	require.NoError(t, err)

	initialDistributorBalanceNullUSD, err := tt.CaminoNetwork.Client.NullUSD.BalanceOf(&bind.CallOpts{Context: ctx}, tt.distributorBot.CMAccountAddress())
	require.NoError(t, err)

	initialSupplierBalanceNullUSD, err := tt.CaminoNetwork.Client.NullUSD.BalanceOf(&bind.CallOpts{Context: ctx}, tt.supplierBot.CMAccountAddress())
	require.NoError(t, err)

	initialASBBalanceNullUSD, err := tt.CaminoNetwork.Client.NullUSD.BalanceOf(&bind.CallOpts{Context: ctx}, tt.ASB.NetworkFeeRecipientCMAccountAddress())
	require.NoError(t, err)

	tt.Logger.Debugf("Initial distributor CM account nullUSD (erc-20 service fee token) balance: %s", initialDistributorBalanceNullUSD.String())
	tt.Logger.Debugf("Initial supplier CM account nullUSD (erc-20 service fee token) balance: %s", initialSupplierBalanceNullUSD.String())
	tt.Logger.Debugf("Initial ASB CM account nullUSD (erc-20 service fee token) balance: %s", initialASBBalanceNullUSD.String())

	pingFeeBig := big.NewInt(tt.pingFee)

	pingMessage := "ping"
	expectedResponseMessageSubString := fmt.Sprintf("Ping response to [%s] with request ID:", pingMessage)

	req := &pingv1.PingRequest{
		Header:      &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		PingMessage: pingMessage,
		Timestamp:   timestamppb.Now(),
	}
	resp, err := tt.distributorBot.PingServiceV1.Ping(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)

	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")
	require.Contains(t, resp.PingMessage, expectedResponseMessageSubString, "unexpected response message")

	expectedDistributorBalanceNullUSD := initialDistributorBalanceNullUSD
	expectedDistributorBalanceNullUSD.Sub(expectedDistributorBalanceNullUSD, pingFeeBig)
	expectedDistributorBalanceNullUSD.Sub(expectedDistributorBalanceNullUSD, config.NetworkFee)

	supplierCashedIn, _ := calculateCashIn(pingFeeBig)
	asbCashedIn, _ := calculateCashIn(config.NetworkFee)

	expectedSupplierBalanceNullUSD := initialSupplierBalanceNullUSD.Add(initialSupplierBalanceNullUSD, supplierCashedIn)
	expectedASBBalanceNullUSD := initialASBBalanceNullUSD.Add(initialASBBalanceNullUSD, asbCashedIn)

	tt.Logger.Debugf("Expected distributor CM account nullUSD (erc-20 service fee token) balance: %s", expectedDistributorBalanceNullUSD.String())
	tt.Logger.Debugf("Expected supplier CM account nullUSD (erc-20 service fee token) balance: %s", expectedSupplierBalanceNullUSD.String())
	tt.Logger.Debugf("Expected ASB CM account nullUSD (erc-20 service fee token) balance: %s", expectedASBBalanceNullUSD.String())

	cashInTimeout := time.Duration(tt.cashInPeriodSeconds) * time.Second * 3 // ASB and supplier cash-in every 10s, triple that

	checkNativeBalanceNeverChanges := func(
		t *testing.T,
		message string,
		expectedBalance *big.Int,
		address common.Address,
	) {
		t.Run("Check "+message, func(t *testing.T) {
			t.Parallel()
			require.Neverf(t, func() bool {
				actualBalance, err := tt.CaminoNetwork.Client.BalanceOf(ctx, address)
				require.NoError(t, err)
				return actualBalance.Cmp(expectedBalance) != 0
			}, cashInTimeout, time.Second, "%s balance changed before timeout", message)
		})
	}

	checkNullUSDBalanceEventually := func(
		t *testing.T,
		message string,
		expectedBalance *big.Int,
		address common.Address,
	) {
		t.Run("Check "+message, func(t *testing.T) {
			t.Parallel()
			var actualBalance *big.Int
			require.EventuallyWithTf(t, func(t *assert.CollectT) {
				actualBalance, err = tt.CaminoNetwork.Client.NullUSD.BalanceOf(&bind.CallOpts{Context: ctx}, address)
				require.NoError(t, err)
				tt.Logger.Debugf("%s: %s", message, actualBalance.String())
				require.True(t, actualBalance.Cmp(expectedBalance) == 0)
			}, cashInTimeout, time.Second,
				"%s did not change by expected amount before timeout: expected %s, actual %s", message, expectedBalance.String(), actualBalance.String(),
			)
		})
	}

	checkNativeBalanceNeverChanges(t, "distributor CM account CAM balance", initialDistributorBalance, tt.distributorBot.CMAccountAddress())
	checkNativeBalanceNeverChanges(t, "supplier CM account CAM balance", initialSupplierBalance, tt.supplierBot.CMAccountAddress())
	checkNativeBalanceNeverChanges(t, "network fee receiver (ASB) CM account CAM balance", initialASBBalance, tt.ASB.NetworkFeeRecipientCMAccountAddress())

	checkNullUSDBalanceEventually(t, "distributor CM account (erc-20 service fee token) balance", expectedDistributorBalanceNullUSD, tt.distributorBot.CMAccountAddress())
	checkNullUSDBalanceEventually(t, "supplier CM account (erc-20 service fee token) balance", expectedSupplierBalanceNullUSD, tt.supplierBot.CMAccountAddress())
	checkNullUSDBalanceEventually(t, "network fee receiver (ASB) CM account (erc-20 service fee token) balance", expectedASBBalanceNullUSD, tt.ASB.NetworkFeeRecipientCMAccountAddress())
}

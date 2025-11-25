// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	pingv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v2"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"buf.build/go/protovalidate"
	"github.com/chain4travel/camino-matrix-app-service/config"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/bot"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/common"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/matrix"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/suite"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert" //nolint:depguard // we don't user assert's assertions, we use assert.CollectT type as needed in require pkg
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestCashIn)(nil)

func init() {
	Tests["PeriodicCashIn"] = &TestCashIn{}
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
		tt.testPeriodicCashInWithPingV2(ctx, t)
	})
}

func (tt *TestCashIn) prepare(ctx context.Context, t *testing.T) {
	// Register all the services needed for the tests
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx, botGenerated.PingServiceV2))

	tt.pingFee = 5_000_000_000_000_000

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{{Name: botGenerated.PingServiceV2, Fee: tt.pingFee}}),
		bot.WithCashInPeriod(tt.cashInPeriodSeconds), // cash-in every 10 seconds
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil,
		// has nothing to cash in, so we'll just check that nothing unexpected happens
		bot.WithCashInPeriod(tt.cashInPeriodSeconds), // cash-in every 10 seconds
	)
}

func (tt *TestCashIn) testPeriodicCashInWithPingV2(ctx context.Context, t *testing.T) {
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

	expectedResponseMessageSubString := fmt.Sprintf("Ping response to [%s] with request ID:", common.PingMessage)

	req := &pingv2.PingRequest{
		Header:    &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		Message:   common.PingMessage,
		Timestamp: timestamppb.Now(),
	}
	resp, err := tt.distributorBot.PingServiceV2.Ping(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)

	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")
	require.Contains(t, successResp.Message, expectedResponseMessageSubString, "unexpected response message")

	expectedDistributorBalanceNullUSD := big.NewInt(0).Set(initialDistributorBalanceNullUSD)
	expectedDistributorBalanceNullUSD.Sub(expectedDistributorBalanceNullUSD, pingFeeBig)
	expectedDistributorBalanceNullUSD.Sub(expectedDistributorBalanceNullUSD, config.NetworkFee)

	supplierCashedIn, _ := calculateCashIn(pingFeeBig)
	asbCashedIn, _ := calculateCashIn(config.NetworkFee)

	expectedSupplierBalanceNullUSD := big.NewInt(0).Add(initialSupplierBalanceNullUSD, supplierCashedIn)
	expectedASBBalanceNullUSD := big.NewInt(0).Add(initialASBBalanceNullUSD, asbCashedIn)

	cashInTimeout := time.Duration(tt.cashInPeriodSeconds) * time.Second * 3 // ASB and supplier cash-in every 10s, triple that

	checkNativeBalance := func(expectedBalance *big.Int, address ethCommon.Address) {
		t.Helper()
		actualBalance, err := tt.CaminoNetwork.Client.BalanceOf(ctx, address)
		require.NoError(t, err)
		require.True(t, actualBalance.Cmp(expectedBalance) == 0)
	}

	checkNullUSDBalanceEventually := func(
		message string,
		expectedBalance *big.Int,
		address ethCommon.Address,
	) {
		t.Helper()
		var actualBalance *big.Int
		require.EventuallyWithTf(t, func(t *assert.CollectT) {
			actualBalance, err = tt.CaminoNetwork.Client.NullUSD.BalanceOf(&bind.CallOpts{Context: ctx}, address)
			require.NoError(t, err)
			require.True(t, actualBalance.Cmp(expectedBalance) == 0)
		}, cashInTimeout, time.Second,
			"%s balance did not change by expected amount before timeout: expected %s, actual %s", message, expectedBalance.String(), actualBalance.String(),
		)
	}

	checkNullUSDBalanceEventually("distributor CM account (erc-20 service fee token)", expectedDistributorBalanceNullUSD, tt.distributorBot.CMAccountAddress())
	checkNullUSDBalanceEventually("supplier CM account (erc-20 service fee token)", expectedSupplierBalanceNullUSD, tt.supplierBot.CMAccountAddress())
	checkNullUSDBalanceEventually("network fee receiver (ASB) CM account (erc-20 service fee token)", expectedASBBalanceNullUSD, tt.ASB.NetworkFeeRecipientCMAccountAddress())

	checkNativeBalance(initialDistributorBalance, tt.distributorBot.CMAccountAddress())
	checkNativeBalance(initialSupplierBalance, tt.supplierBot.CMAccountAddress())
	checkNativeBalance(initialASBBalance, tt.ASB.NetworkFeeRecipientCMAccountAddress())
}

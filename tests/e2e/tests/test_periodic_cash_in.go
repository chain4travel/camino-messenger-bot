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
	e2eCommon "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/common"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func testPeriodicCashInSetup(ctx context.Context, t *testing.T, tt *Test) (
	supplierPartnerPlugin *partnerplugin.PartnerPlugin,
	supplierBot *bot.Bot,
	distributorBot *bot.Bot,
	pingFee int64,
) {
	// Register all the services needed for the tests
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMServices(ctx, botGenerated.PingServiceV1))
	supplierPartnerPlugin = tt.createPartnerPlugin(ctx, t)

	pingFee = 5_000_000_000_000_000_000

	// bot with partnerPlugin and without rpc server (supplier)
	supplierBot = tt.createBot(ctx, t, true, supplierPartnerPlugin, []bot.CMService{
		{Name: botGenerated.PingServiceV1, Fee: pingFee},
	})

	// bot without partnerPlugin and with rpc server (distributor)
	distributorBot = tt.createBot(ctx, t, true, nil, nil)

	return supplierPartnerPlugin, supplierBot, distributorBot, pingFee
}

func testPeriodicCashInWithPingV1(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	pingFee int64,
) {
	initialDistributorBalance, err := tt.caminoNetwork.Client.ETHClient().BalanceAt(ctx, distributorBot.CMAccountAddress(), nil)
	require.NoError(t, err)

	initialSupplierBalance, err := tt.caminoNetwork.Client.ETHClient().BalanceAt(ctx, supplierBot.CMAccountAddress(), nil)
	require.NoError(t, err)

	initialASBBalance, err := tt.caminoNetwork.Client.ETHClient().BalanceAt(ctx, tt.asb.NetworkFeeRecipientCMAccountAddress(), nil)
	require.NoError(t, err)

	tt.logger.Debugf("Initial distributor CM account balance: %s", initialDistributorBalance.String())
	tt.logger.Debugf("Initial supplier CM account balance: %s", initialSupplierBalance.String())
	tt.logger.Debugf("Initial ASB CM account balance: %s", initialASBBalance.String())

	pingFeeBig := big.NewInt(pingFee)

	pingMessage := "ping"
	expectedResponseMessageSubString := fmt.Sprintf("Ping response to [%s] with request ID:", pingMessage)

	req := &pingv1.PingRequest{
		Header:      &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		PingMessage: pingMessage,
		Timestamp:   timestamppb.Now(),
	}
	resp, err := distributorBot.PingServiceV1.Ping(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)

	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")
	require.Contains(t, resp.PingMessage, expectedResponseMessageSubString, "unexpected response message")

	expectedDistributorBalance := initialDistributorBalance
	expectedDistributorBalance.Sub(initialDistributorBalance, pingFeeBig)
	expectedDistributorBalance.Sub(expectedDistributorBalance, config.NetworkFee)

	supplierCashedIn, _ := calculateCashIn(pingFeeBig)
	asbCashedIn, _ := calculateCashIn(config.NetworkFee)

	expectedSupplierBalance := initialSupplierBalance.Add(initialSupplierBalance, supplierCashedIn)
	expectedASBBalance := initialASBBalance.Add(initialASBBalance, asbCashedIn)

	tt.logger.Debugf("Expected distributor CM account balance: %s", expectedDistributorBalance.String())
	tt.logger.Debugf("Expected supplier CM account balance: %s", expectedSupplierBalance.String())
	tt.logger.Debugf("Expected ASB CM account balance: %s", expectedASBBalance.String())

	cashInTimeout := e2eCommon.CashInPeriod * 3 // supplier and ASB cash-in every 10s, triple that

	t.Run("Check distributor balance", func(t *testing.T) {
		t.Parallel()
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			distributorBalance, err := tt.caminoNetwork.Client.ETHClient().BalanceAt(ctx, distributorBot.CMAccountAddress(), nil)
			require.NoError(t, err)
			tt.logger.Debugf("Distributor CM account balance: %s", distributorBalance.String())
			require.Equal(t, distributorBalance.Cmp(expectedDistributorBalance), 0)
		}, cashInTimeout, time.Second, "Distributor CM account balance did not decrease by expected amount before timeout")
	})

	t.Run("Check supplier balance", func(t *testing.T) {
		t.Parallel()
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			supplierBalance, err := tt.caminoNetwork.Client.ETHClient().BalanceAt(ctx, supplierBot.CMAccountAddress(), nil)
			require.NoError(t, err)
			tt.logger.Debugf("Supplier CM account balance: %s", supplierBalance.String())
			require.Equal(t, supplierBalance.Cmp(expectedSupplierBalance), 0)
		}, cashInTimeout, time.Second, "Supplier CM account balance did not increase by expected amount before timeout")
	})

	t.Run("Check network fee receiver (ASB) balance", func(t *testing.T) {
		t.Parallel()
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			asbBalance, err := tt.caminoNetwork.Client.ETHClient().BalanceAt(ctx, tt.asb.NetworkFeeRecipientCMAccountAddress(), nil)
			require.NoError(t, err)
			tt.logger.Debugf("ASB CM account balance: %s", asbBalance.String())
			require.Equal(t, asbBalance.Cmp(expectedASBBalance), 0)
		}, cashInTimeout, time.Second, "ASB CM account balance did not increase by expected amount before timeout")
	})
}

func TestPeriodicCashIn(t *testing.T, tt *Test) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()
	var supplierBot *bot.Bot
	var distributorBot *bot.Bot
	var pingFee int64

	t.Run("Setup", func(t *testing.T) {
		_, supplierBot, distributorBot, pingFee = testPeriodicCashInSetup(ctx, t, tt)
	})
	t.Run("Ping", func(t *testing.T) {
		testPeriodicCashInWithPingV1(ctx, t, tt, distributorBot, supplierBot, pingFee)
	})
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"fmt"
	"math/big"
	"sync"
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
	"github.com/ethereum/go-ethereum/common"
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
		tt.testPeriodicCashInWithPingV1(ctx, t)
	})
}

func (tt *TestCashIn) prepare(ctx context.Context, t *testing.T) {
	// Register all the services needed for the tests
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx, botGenerated.PingServiceV1))

	tt.pingFee = 5_000_000_000_000_000

	wg := sync.WaitGroup{}

	// bot with partnerPlugin and without rpc server (supplier)
	wg.Add(1)
	go func() {
		defer wg.Done()
		tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
		tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
			bot.WithServices([]bot.CMService{{Name: botGenerated.PingServiceV1, Fee: tt.pingFee}}),
			bot.WithCashInPeriod(tt.cashInPeriodSeconds), // cash-in every 10 seconds
		)
	}()

	// bot without partnerPlugin and with rpc server (distributor)
	wg.Add(1)
	go func() {
		defer wg.Done()
		tt.distributorBot = tt.CreateBot(ctx, t, true, nil,
			// has nothing to cash in, so we'll just check that nothing unexpected happens
			bot.WithCashInPeriod(tt.cashInPeriodSeconds), // cash-in every 10 seconds
		)
	}()

	wg.Wait()
}

func (tt *TestCashIn) testPeriodicCashInWithPingV1(ctx context.Context, t *testing.T) {
	initialDistributorBalance, err := tt.CaminoNetwork.Client.ETHClient().BalanceAt(ctx, tt.distributorBot.CMAccountAddress(), nil)
	require.NoError(t, err)

	initialSupplierBalance, err := tt.CaminoNetwork.Client.ETHClient().BalanceAt(ctx, tt.supplierBot.CMAccountAddress(), nil)
	require.NoError(t, err)

	initialASBBalance, err := tt.CaminoNetwork.Client.ETHClient().BalanceAt(ctx, tt.ASB.NetworkFeeRecipientCMAccountAddress(), nil)
	require.NoError(t, err)

	tt.Logger.Debugf("Initial distributor CM account balance: %s", initialDistributorBalance.String())
	tt.Logger.Debugf("Initial supplier CM account balance: %s", initialSupplierBalance.String())
	tt.Logger.Debugf("Initial ASB CM account balance: %s", initialASBBalance.String())

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

	expectedDistributorBalance := initialDistributorBalance
	expectedDistributorBalance.Sub(initialDistributorBalance, pingFeeBig)
	expectedDistributorBalance.Sub(expectedDistributorBalance, config.NetworkFee)

	supplierCashedIn, _ := calculateCashIn(pingFeeBig)
	asbCashedIn, _ := calculateCashIn(config.NetworkFee)

	expectedSupplierBalance := initialSupplierBalance.Add(initialSupplierBalance, supplierCashedIn)
	expectedASBBalance := initialASBBalance.Add(initialASBBalance, asbCashedIn)

	tt.Logger.Debugf("Expected distributor CM account balance: %s", expectedDistributorBalance.String())
	tt.Logger.Debugf("Expected supplier CM account balance: %s", expectedSupplierBalance.String())
	tt.Logger.Debugf("Expected ASB CM account balance: %s", expectedASBBalance.String())

	cashInTimeout := time.Duration(tt.cashInPeriodSeconds) * time.Second * 3 // ASB and supplier cash-in every 10s, triple that

	checkBalanceEventually := func(
		t *testing.T,
		message string,
		expectedBalance *big.Int,
		address common.Address,
	) {
		t.Run("Check "+message, func(t *testing.T) {
			t.Parallel()
			var actualBalance *big.Int
			require.EventuallyWithTf(t, func(t *assert.CollectT) {
				actualBalance, err = tt.CaminoNetwork.Client.ETHClient().BalanceAt(ctx, address, nil)
				require.NoError(t, err)
				tt.Logger.Debugf("%s: %s", message, actualBalance.String())
				require.True(t, actualBalance.Cmp(expectedBalance) == 0)
			}, cashInTimeout, time.Second,
				"%s did not change by expected amount before timeout: expected %s, actual %s", message, expectedBalance.String(), actualBalance.String(),
			)
		})
	}

	checkBalanceEventually(t, "distributor CM account balance", expectedDistributorBalance, tt.distributorBot.CMAccountAddress())
	checkBalanceEventually(t, "supplier CM account balance", expectedSupplierBalance, tt.supplierBot.CMAccountAddress())
	checkBalanceEventually(t, "network fee receiver (ASB) CM account balance", expectedASBBalance, tt.ASB.NetworkFeeRecipientCMAccountAddress())
}

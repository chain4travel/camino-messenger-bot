// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"
	"time"

	pingv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v2"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"buf.build/go/protovalidate"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/generated"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v12/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/blockchain"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/bot"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/common"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/suite"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const manyMessagesTestIterations = 1000

var _ suite.Test = (*TestBotSanity)(nil)

func init() {
	Tests["BotSanity"] = &TestBotSanity{}
}

type TestBotSanity struct {
	*suite.Environment

	supplierPartnerPlugin             *partnerplugin.PartnerPlugin
	distributorBot                    *bot.Bot
	supplierBot                       *bot.Bot
	supplierBotWithBigFee             *bot.Bot
	supplierBotUnregistered           *bot.Bot
	supplierBotUnregisteredNoServices *bot.Bot
	supplierBotNoServices             *bot.Bot
	supplierBotDifferentServices      *bot.Bot
}

func (tt *TestBotSanity) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestBotSanity) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepareBeforeCMManagerRegisterServices(ctx, t)

	t.Run("Missing global CM-Account-Manager services", func(t *testing.T) {
		// This should fail already before the bot is even started up
		// as the CM-Account-Manager services are not registered yet and the
		// factory tries to register a service which is not available
		err := tt.CreateBotWithError(ctx, false, tt.supplierPartnerPlugin,
			bot.WithServices([]bot.CMService{{Name: botGenerated.PingServiceV2, Fee: 100}}),
		)
		require.ErrorContains(t, err, blockchain.ErrorAddServiceTxFailed.Error())
	})

	tt.prepareAfterCMManagerRegisterServices(ctx, t)

	t.Run("Missing CM-Account", func(t *testing.T) {
		// This should fail already when the bot starts up as there is no
		// CM-Account to use - we check that accordingly with the return value
		tt.CreateBotAwaitError(ctx, t, false, tt.supplierPartnerPlugin, "exit status 1", 5*time.Second,
			bot.WithServices([]bot.CMService{{Name: botGenerated.PingServiceV2, Fee: 100}}),
			bot.WithSkips(&bot.Skip{CMAccountCreation: true}),
		)
	})
	t.Run("Supplier bot: unregistered / no services", func(t *testing.T) {
		alertMessage := tt.sendPingRequestAndGetErrorMessage(ctx, t, "unregistered / no services", tt.supplierBotUnregisteredNoServices)
		require.Contains(t, alertMessage, cmaccounts.ErrorNoChequeOperators.Error())
	})
	t.Run("Supplier bot: unregistered / with services", func(t *testing.T) {
		alertMessage := tt.sendPingRequestAndGetErrorMessage(ctx, t, "unregistered / with services", tt.supplierBotUnregistered)
		require.Contains(t, alertMessage, cmaccounts.ErrorNoChequeOperators.Error())
	})
	t.Run("Supplier bot: registered / no services", func(t *testing.T) {
		alertMessage := tt.sendPingRequestAndGetErrorMessage(ctx, t, "registered / no services", tt.supplierBotNoServices)
		require.Contains(t, alertMessage, cmaccounts.ErrorUnableToObtainServiceFee.Error())
	})
	t.Run("Supplier bot: registered / different services", func(t *testing.T) {
		alertMessage := tt.sendPingRequestAndGetErrorMessage(ctx, t, "registered / different services", tt.supplierBotDifferentServices)
		require.Contains(t, alertMessage, cmaccounts.ErrorUnableToObtainServiceFee.Error())
	})
	t.Run("Many messages", func(t *testing.T) {
		tt.testManyMessages(ctx, t)
	})
}

func (tt *TestBotSanity) prepareBeforeCMManagerRegisterServices(ctx context.Context, t *testing.T) {
	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// This bot skips the bot registration and the service registration
	// But the CM-Account is created - therefore the creation in the context
	// of the test should work but later when trying to use the bot it should fail
	tt.supplierBotUnregisteredNoServices = tt.CreateBot(ctx, t, false, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{{Name: botGenerated.PingServiceV2, Fee: 100}}),
		bot.WithSkips(&bot.Skip{PrefundOwner: true}),
	)
}

func (tt *TestBotSanity) prepareAfterCMManagerRegisterServices(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.PingServiceV2,
		botGenerated.MintServiceV4,
	))

	// This bot does actually have the CM-Account and prefunding of the owner
	// But only the bot registration is missing in the CM-Account
	// With that the distributor bot should not be able to find the supplier bot
	// and fail with an error in a later test
	tt.supplierBotUnregistered = tt.CreateBot(ctx, t, false, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{{Name: botGenerated.PingServiceV2, Fee: 100}}),
		bot.WithSkips(&bot.Skip{BotRegistration: true}),
	)

	// This bot does actually have the CM-Account and prefunding of the owner
	// and is also registered in the CM-Account **BUT** the service registration
	// is missing - this is checked by the bot but results only in a warning
	// inside of the logs. We can later use this bot to check if the distributor
	// bot acts correctly by rejecting this supplier bot as the required service is missing
	tt.supplierBotNoServices = tt.CreateBot(ctx, t, false, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{{Name: botGenerated.PingServiceV2, Fee: 100}}),
		bot.WithSkips(&bot.Skip{ServiceRegistration: true}),
	)

	// All good here - just a different service used which should then
	// fail in the later test
	tt.supplierBotDifferentServices = tt.CreateBot(ctx, t, false, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{{Name: botGenerated.MintServiceV4, Fee: 100}}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, false, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{{Name: botGenerated.PingServiceV2, Fee: 100}}),
	)

	// bot with partnerPlugin (supplier), with ping service fee being very high
	// this is needed, because conduit matrix server marshals json content of message and fails,
	// if json contains numbers bigger than javaScript safe integer limit (2^53-1)
	tt.supplierBotWithBigFee = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{{Name: botGenerated.PingServiceV2, Fee: 18014398509481984}}), // 2^54 > 2^53-1
	)
}

func (tt *TestBotSanity) testManyMessages(ctx context.Context, t *testing.T) {
	for range manyMessagesTestIterations {
		tt.pingMessage(ctx, t, tt.supplierBot)
	}
}

func (tt *TestBotSanity) pingMessage(ctx context.Context, t *testing.T, supplierBot *bot.Bot) {
	req := &pingv2.PingRequest{
		Header:    &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		Message:   common.PingMessage,
		Timestamp: timestamppb.Now(),
	}
	resp, err := tt.distributorBot.PingServiceV2.Ping(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	require.True(t, resp.HasSuccessResponse())
	tt.DebugPrintRequestResponse(req, resp)
}

func (tt *TestBotSanity) sendPingRequestAndGetErrorMessage(ctx context.Context, t *testing.T, pingMessage string, supplierBot *bot.Bot) string {
	req := &pingv2.PingRequest{
		Header:    &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		Message:   pingMessage,
		Timestamp: timestamppb.Now(),
	}
	resp, err := tt.distributorBot.PingServiceV2.Ping(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	require.NoError(t, protovalidate.Validate(resp))

	errResp := resp.GetErrorResponse()
	require.NotNil(t, errResp, "expected error response")
	require.Len(t, errResp.Header.Errors, 1, "expected one alert in response header")
	return errResp.Header.Errors[0].Message
}

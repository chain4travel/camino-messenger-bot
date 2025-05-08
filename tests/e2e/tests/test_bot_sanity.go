// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"
	"time"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v11/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/blockchain"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type botSanityServices struct {
	supplierPartnerPlugin             *partnerplugin.PartnerPlugin
	distributorBot                    *bot.Bot
	supplierBotUnregistered           *bot.Bot
	supplierBotUnregisteredNoServices *bot.Bot
	supplierBotNoServices             *bot.Bot
	supplierBotDifferentServices      *bot.Bot
}

func testBotSanitySetupWithSanityChecks(ctx context.Context, t *testing.T, tt *Test) (services *botSanityServices) {
	services = &botSanityServices{}
	services.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
	var errChan chan error
	var err error

	t.Run("Missing CM-Account", func(t *testing.T) {
		_, errChan, err = tt.botFactory.CreateBot(ctx, false, services.supplierPartnerPlugin,
			[]bot.CMService{{Name: botGenerated.PingServiceV1, Fee: 100}},
			&bot.Skip{CMAccountCreation: true},
		)

		// This should fail already when the bot starts up as there is no
		// CM-Account to use - we check that accordingly with the return value
		require.NoError(t, err)
		expectChannelErrorWithTimeout(t, errChan, "exit status 1", 5*time.Second)
	})

	t.Run("Missing CM-Account owner funds", func(t *testing.T) {
		services.supplierBotUnregisteredNoServices, errChan, err = tt.botFactory.CreateBot(ctx, false, services.supplierPartnerPlugin,
			[]bot.CMService{{Name: botGenerated.PingServiceV1, Fee: 100}},
			&bot.Skip{PrefundOwner: true},
		)

		// This bot skips the bot registration and the service registration
		// But the CM-Account is created - therefore the creation in the context
		// of the test should work but later when trying to use the bot it should fail
		require.NoError(t, err)
		expectNoErrorAsync(t, errChan)
	})

	t.Run("Missing global CM-Account-Manager services", func(t *testing.T) {
		_, errChan, err = tt.botFactory.CreateBot(ctx, false, services.supplierPartnerPlugin,
			[]bot.CMService{{Name: botGenerated.PingServiceV1, Fee: 100}},
			&bot.Skip{},
		)

		// This should fail already before the bot is even started up
		// as the CM-Account-Manager services are not registered yet and the
		// factory tries to register a service which is not available
		require.Error(t, err)
		require.Contains(t, err.Error(), blockchain.ErrorAddServiceTxFailed.Error())
	})

	require.NoError(t, tt.caminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.PingServiceV1,
		botGenerated.MintServiceV3,
	))

	t.Run("Missing Bot-Registration", func(t *testing.T) {
		services.supplierBotUnregistered, errChan, err = tt.botFactory.CreateBot(ctx, false, services.supplierPartnerPlugin,
			[]bot.CMService{{Name: botGenerated.PingServiceV1, Fee: 100}},
			&bot.Skip{BotRegistration: true},
		)

		// This bot does actually have the CM-Account and prefunding of the owner
		// But only the bot registration is missing in the CM-Account
		// With that the distributor bot should not be able to find the supplier bot
		// and fail with an error in a later test
		require.NoError(t, err)
		expectNoErrorAsync(t, errChan)
	})

	t.Run("Missing Bot service registration", func(t *testing.T) {
		services.supplierBotNoServices, errChan, err = tt.botFactory.CreateBot(ctx, false, services.supplierPartnerPlugin,
			[]bot.CMService{{Name: botGenerated.PingServiceV1, Fee: 100}},
			&bot.Skip{ServiceRegistration: true},
		)

		// This bot does actually have the CM-Account and prefunding of the owner
		// and is also registered in the CM-Account **BUT** the service registration
		// is missing - this is checked by the bot but results only in a warning
		// inside of the logs. We can later use this bot to check if the distributor
		// bot acts correctly by rejecting this supplier bot as the required service is missing
		require.NoError(t, err)
		expectNoErrorAsync(t, errChan)
	})

	t.Run("Different services", func(t *testing.T) {
		services.supplierBotDifferentServices, errChan, err = tt.botFactory.CreateBot(ctx, false, services.supplierPartnerPlugin,
			[]bot.CMService{{Name: botGenerated.MintServiceV3, Fee: 100}},
			&bot.Skip{},
		)

		// All good here - just a different service used which should then
		// fail in the later test
		require.NoError(t, err)
		expectNoErrorAsync(t, errChan)
	})

	// bot without partnerPlugin and with rpc server (distributor)
	services.distributorBot = tt.CreateBot(ctx, t, true, nil, nil)

	return services
}

func testBotSanitySendCommonRequest(ctx context.Context, pingMessage string, distributorBot *bot.Bot, supplierBot *bot.Bot) error {
	req := &pingv1.PingRequest{
		Header:      &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		PingMessage: pingMessage,
		Timestamp:   timestamppb.Now(),
	}
	_, err := distributorBot.PingServiceV1.Ping(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	return err
}

func testBotSanityVerify(ctx context.Context, t *testing.T, services *botSanityServices) {
	t.Run("Supplier bot: unregistered / no services", func(t *testing.T) {
		err := testBotSanitySendCommonRequest(ctx, "unregistered / no services", services.distributorBot, services.supplierBotUnregisteredNoServices)
		require.Error(t, err)
		require.Contains(t, err.Error(), cmaccounts.ErrorNoChequeOperators.Error())
	})

	t.Run("Supplier bot: unregistered / with services", func(t *testing.T) {
		err := testBotSanitySendCommonRequest(ctx, "unregistered / with services", services.distributorBot, services.supplierBotUnregistered)
		require.Error(t, err)
		require.Contains(t, err.Error(), cmaccounts.ErrorNoChequeOperators.Error())
	})

	t.Run("Supplier bot: registered / no services", func(t *testing.T) {
		err := testBotSanitySendCommonRequest(ctx, "registered / no services", services.distributorBot, services.supplierBotNoServices)
		require.Error(t, err)
		require.Contains(t, err.Error(), cmaccounts.ErrorUnableToObtainServiceFee.Error())
	})

	t.Run("Supplier bot: registered / different services", func(t *testing.T) {
		err := testBotSanitySendCommonRequest(ctx, "registered / different services", services.distributorBot, services.supplierBotDifferentServices)
		require.Error(t, err)
		require.Contains(t, err.Error(), cmaccounts.ErrorUnableToObtainServiceFee.Error())
	})
}

func TestBotSanity(t *testing.T, tt *Test) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()
	var services *botSanityServices

	t.Run("Startup and sanity checks", func(t *testing.T) {
		services = testBotSanitySetupWithSanityChecks(ctx, t, tt)
	})

	t.Run("Verification", func(t *testing.T) {
		testBotSanityVerify(ctx, t, services)
	})
}

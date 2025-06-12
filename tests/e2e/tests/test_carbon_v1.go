package tests

import (
	"context"
	"testing"

	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/stretchr/testify/require"
)

func testCarbonCompensateV1Setup(
	ctx context.Context,
	t *testing.T,
	tt *Test,
) (
	supplierPartnerPlugin *partnerplugin.PartnerPlugin,
	supplierBot *bot.Bot,
	distributorBot *bot.Bot,
	distributorBotWithoutFunds *bot.Bot,
) {
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationSearchServiceV3,
		botGenerated.ValidationServiceV2,
		botGenerated.MintServiceV2,
	))
	supplierPartnerPlugin = tt.createPartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	supplierBot = tt.createBot(ctx, t, false, supplierPartnerPlugin, []bot.CMService{
		{Name: botGenerated.AccommodationSearchServiceV3, Fee: 120},
		{Name: botGenerated.ValidationServiceV2, Fee: 130},
		{Name: botGenerated.MintServiceV2, Fee: 140},
		{Name: botGenerated.CarbonCompensateServiceV1, Fee: 100},
	})

	// bot without partnerPlugin and with rpc server (distributor)
	distributorBot = tt.createBot(ctx, t, true, nil, nil)

	// bot without partnerPlugin and with rpc server (distributor) but with the
	// catch, that the bot account does not have funds to pay for the fees when
	// trying to buy the booking token.
	distributorBotWithoutFunds, errChan, err := tt.botFactory.CreateBot(ctx, true, nil, nil,
		&bot.Skip{PrefundBot: true},
	)
	require.NoError(t, err)
	expectNoErrorAsync(t, errChan)

	return supplierPartnerPlugin, supplierBot, distributorBot, distributorBotWithoutFunds
}

func TestCarbonCompensateV1(
	t *testing.T,
	tt *Test,
) {
}

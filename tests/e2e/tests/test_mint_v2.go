// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"sync"
	"testing"

	notificationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v1"
	botGenerated "github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/tests/e2e/partner_plugin"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// Setting up the basic applications and services used in all sub-test-cases
func testMintV2Setup(
	ctx context.Context,
	t *testing.T,
	tt *Test,
) (
	supplierPartnerPlugin *partnerplugin.PartnerPlugin,
	supplierBot *bot.Bot,
	distributorBot *bot.Bot,
) {
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationSearchServiceV3,
		botGenerated.ValidationServiceV2,
		botGenerated.MintServiceV2,
	))
	supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	supplierBot = tt.CreateBot(ctx, t, false, supplierPartnerPlugin, []bot.CMService{
		{Name: botGenerated.AccommodationSearchServiceV3, Fee: 120},
		{Name: botGenerated.ValidationServiceV2, Fee: 130},
		{Name: botGenerated.MintServiceV2, Fee: 140},
	})

	// bot without partnerPlugin and with rpc server (distributor)
	distributorBot = tt.CreateBot(ctx, t, true, nil, nil)

	return supplierPartnerPlugin, supplierBot, distributorBot
}

func TestMintV2Setup(t *testing.T, tt *Test) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()
	var supplierBot *bot.Bot
	var distributorBot *bot.Bot
	var supplierPartnerPlugin *partnerplugin.PartnerPlugin

	t.Run("Setup", func(t *testing.T) {
		supplierPartnerPlugin, supplierBot, distributorBot = testMintV2Setup(ctx, t, tt)
	})

	t.Run("Search->Validate->Mint->TokenBoughtNotification", func(t *testing.T) {
		searchID, resultID, totalPrice := testAccommodationV3SearchServiceWithTravelPeriod(ctx, t, tt, distributorBot, supplierBot) // see test_accommodation_v3.go
		validationID := testAccommodationV3ValidateV2(ctx, t, tt, distributorBot, supplierBot, searchID, resultID, totalPrice)      // see test_accommodation_v3.go

		ppEventStream, err := supplierPartnerPlugin.SubscribeForEvents(ctx)
		require.NoError(t, err)

		var tokenID uint64
		var mintID string

		wg := sync.WaitGroup{}

		wg.Add(1)
		go func() {
			eventMsg, err := ppEventStream.Recv()
			require.NoError(t, err)
			debugPrintProtoMessage(tt, eventMsg)
			tokenBoughtNotification := &notificationv1.TokenBought{}
			require.NoError(t, proto.Unmarshal(eventMsg.Data, tokenBoughtNotification))
			require.Equal(t, tokenBoughtNotification.TokenId, tokenID)
			require.Equal(t, tokenBoughtNotification.MintId, mintID)
			require.NotEmpty(t, tokenBoughtNotification.TxId)
			wg.Done()
		}()

		tokenID, _, mintID = testAccommodationV3MintV2(ctx, t, tt, distributorBot, supplierBot, validationID)

		wg.Wait()
	})
}

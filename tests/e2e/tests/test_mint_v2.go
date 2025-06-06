// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"

	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	notificationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/proto/pb/events"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
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
	distributorBotWithoutFunds *bot.Bot,
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

func testMintV2FullWorkflow(ctx context.Context, t *testing.T, tt *Test, ppEventStream events.EventsService_SubscribeClient, distributorBot *bot.Bot, supplierBot *bot.Bot) {
	// Don't mind the eventStream receives without further processing.
	// We just receive all the messages from the pp-mock event stream without any
	// further checks as we're only really interested in the last one.

	searchID, resultID, totalPrice := testAccommodationV3SearchServiceWithTravelPeriod(ctx, t, tt, distributorBot, supplierBot) // see test_accommodation_v3.go
	_, err := ppEventStream.Recv()                                                                                              // skip AccommodationSearchRequest
	require.NoError(t, err)

	validationID := testAccommodationV3ValidateV2(ctx, t, tt, distributorBot, supplierBot, searchID, resultID, totalPrice) // see test_accommodation_v3.go
	_, err = ppEventStream.Recv()                                                                                          // skip ValidateRequest
	require.NoError(t, err)

	tokenID, _, mintID := testAccommodationV3MintV2(ctx, t, tt, distributorBot, supplierBot, validationID) // see test_accommodation_v3.go
	_, err = ppEventStream.Recv()                                                                          // skip MintRequest
	require.NoError(t, err)

	// We're actually interested in this message which is
	// the TokenBoughtNotification
	eventMsg, err := ppEventStream.Recv()
	require.NoError(t, err)
	debugPrintProtoMessage(tt, eventMsg)
	tokenBoughtNotification := &notificationv2.TokenBought{}
	require.NoError(t, proto.Unmarshal(eventMsg.Data, tokenBoughtNotification))
	require.Equal(t, tokenBoughtNotification.TokenId, tokenID)
	require.NotNil(t, tokenBoughtNotification.MintId)
	require.Equal(t, tokenBoughtNotification.MintId.Value, mintID)
	require.NotEmpty(t, tokenBoughtNotification.TxId)
}

// Lastly we do the mint request based on the validation id
func testMintV2MintV2ExpectedError(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	validationID string,
) (
	tokenID uint64,
	mintID string,
) {
	req := &bookv2.MintRequest{
		Header:       &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		ValidationId: &typesv1.UUID{Value: validationID},
	}
	resp, err := distributorBot.MintServiceV2.Mint(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")

	// Check if the MintId is set
	require.NotEmpty(t, resp.MintId, "unexpected empty response MintId")
	require.NotEmpty(t, resp.MintId.Value, "unexpected empty response MintId.Value")

	require.NotEmpty(t, resp.MintTransactionId, "unexpected empty response MintTransactionId")
	require.Empty(t, resp.BuyTransactionId, "unexpected response BuyTransactionId")

	return resp.BookingTokenId, resp.MintId.Value
}

func testMintV2TokenExpiredCase(ctx context.Context, t *testing.T, tt *Test, ppEventStream events.EventsService_SubscribeClient, distributorBot *bot.Bot, supplierBot *bot.Bot) {
	// Don't mind the eventStream receives without further processing.
	// We just receive all the messages from the pp-mock event stream without any
	// further checks as we're only really interested in the last one.

	searchID, resultID, totalPrice := testAccommodationV3SearchServiceWithTravelPeriod(ctx, t, tt, distributorBot, supplierBot) // see test_accommodation_v3.go
	_, err := ppEventStream.Recv()                                                                                              // skip AccommodationSearchRequest
	require.NoError(t, err)

	validationID1 := testAccommodationV3ValidateV2(ctx, t, tt, distributorBot, supplierBot, searchID, resultID, totalPrice) // see test_accommodation_v3.go
	_, err = ppEventStream.Recv()                                                                                           // skip ValidateRequest
	require.NoError(t, err)

	searchID, resultID, totalPrice = testAccommodationV3SearchServiceWithTravelPeriod(ctx, t, tt, distributorBot, supplierBot) // see test_accommodation_v3.go
	_, err = ppEventStream.Recv()                                                                                              // skip AccommodationSearchRequest
	require.NoError(t, err)

	validationID2 := testAccommodationV3ValidateV2(ctx, t, tt, distributorBot, supplierBot, searchID, resultID, totalPrice) // see test_accommodation_v3.go
	_, err = ppEventStream.Recv()                                                                                           // skip ValidateRequest
	require.NoError(t, err)

	var tokenID1 uint64
	var mintID1 string

	tokenID1, mintID1 = testMintV2MintV2ExpectedError(ctx, t, tt, distributorBot, supplierBot, validationID1)
	_, err = ppEventStream.Recv() // skip MintRequest
	require.NoError(t, err)

	tokenID2, mintID2 := testMintV2MintV2ExpectedError(ctx, t, tt, distributorBot, supplierBot, validationID2)
	_, err = ppEventStream.Recv() // skip MintRequest
	require.NoError(t, err)

	// Following code relies on specific order of token expired notifications.
	// We can safely assume that 2nd token expired notification will come after the first one,
	// because timeout is set by mint request, and 2nd mint is happening after 1st.

	eventMsg, err := ppEventStream.Recv()
	require.NoError(t, err)
	debugPrintProtoMessage(tt, eventMsg)
	tokenExpiredNotification := &notificationv2.TokenExpired{}
	require.NoError(t, proto.Unmarshal(eventMsg.Data, tokenExpiredNotification))
	require.Equal(t, tokenExpiredNotification.TokenId, tokenID1)
	require.NotNil(t, tokenExpiredNotification.MintId)
	require.Equal(t, tokenExpiredNotification.MintId.Value, mintID1)
	eventMsg, err = ppEventStream.Recv()
	require.NoError(t, err)
	debugPrintProtoMessage(tt, eventMsg)
	tokenExpiredNotification = &notificationv2.TokenExpired{}
	require.NoError(t, proto.Unmarshal(eventMsg.Data, tokenExpiredNotification))
	require.Equal(t, tokenExpiredNotification.TokenId, tokenID2)
	require.NotNil(t, tokenExpiredNotification.MintId)
	require.Equal(t, tokenExpiredNotification.MintId.Value, mintID2)
}

func TestMintV2(t *testing.T, tt *Test) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	supplierPartnerPlugin, supplierBot, distributorBot, distributorBotWithoutFunds := testMintV2Setup(ctx, t, tt)
	ppEventStream, err := supplierPartnerPlugin.SubscribeForEvents(ctx)
	require.NoError(t, err)

	t.Run("Search->Validate->Mint->TokenBoughtNotification", func(t *testing.T) {
		// We're doing this > 1 times to make sure that even with multiple
		// mint requests everything is working as expected.
		for range 3 {
			testMintV2FullWorkflow(ctx, t, tt, ppEventStream, distributorBot, supplierBot)
		}
	})

	t.Run("Search->Validate->Mint->TokenTimeoutNotification", func(t *testing.T) {
		testMintV2TokenExpiredCase(ctx, t, tt, ppEventStream, distributorBotWithoutFunds, supplierBot)
	})
}

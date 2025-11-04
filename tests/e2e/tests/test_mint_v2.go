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
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/proto/pb/events"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/suite"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var _ suite.Test = (*TestMintV2)(nil)

func init() {
	Tests["MintV2"] = &TestMintV2{}
}

type TestMintV2 struct {
	*suite.Environment

	supplierPartnerPlugin      *partnerplugin.PartnerPlugin
	supplierPPEventStream      events.EventsService_SubscribeClient
	supplierBot                *bot.Bot
	distributorBot             *bot.Bot
	distributorBotWithoutFunds *bot.Bot
}

func (tt *TestMintV2) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestMintV2) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("Search->Validate->Mint->TokenBoughtNotification", func(t *testing.T) {
		// We're doing this > 1 times to make sure that even with multiple
		// mint requests everything is working as expected.
		for range 3 {
			tt.testMintV2FullWorkflow(ctx, t)
		}
	})

	t.Run("Search->Validate->Mint->TokenTimeoutNotification", func(t *testing.T) {
		tt.testMintV2TokenExpiredCase(ctx, t)
	})
}

func (tt *TestMintV2) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationSearchServiceV3,
		botGenerated.ValidationServiceV2,
		botGenerated.MintServiceV2,
	))

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.AccommodationSearchServiceV3, Fee: 120},
			{Name: botGenerated.ValidationServiceV2, Fee: 130},
			{Name: botGenerated.MintServiceV2, Fee: 140},
		}),
	)

	var err error
	tt.supplierPPEventStream, err = tt.supplierPartnerPlugin.SubscribeForEvents(ctx)
	require.NoError(t, err)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)

	// bot without partnerPlugin and with rpc server (distributor) but with the
	// catch, that the bot account does not have funds to pay for the fees when
	// trying to buy the booking token.
	tt.distributorBotWithoutFunds = tt.CreateBot(ctx, t, true, nil,
		bot.WithSkips(&bot.Skip{PrefundBot: true}),
	)
}

func (tt *TestMintV2) testMintV2FullWorkflow(ctx context.Context, t *testing.T) {
	// Don't mind the eventStream receives without further processing.
	// We just receive all the messages from the pp-mock event stream without any
	// further checks as we're only really interested in the last one.

	searchID, resultID, totalPrice := testAccommodationV3SearchServiceWithTravelPeriod(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot) // see test_accommodation_v3.go
	_, err := tt.supplierPPEventStream.Recv()                                                                                                     // skip AccommodationSearchRequest
	require.NoError(t, err)

	validationID := testValidateV2(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
	_, err = tt.supplierPPEventStream.Recv() // skip ValidateRequest
	require.NoError(t, err)

	tokenID, mintID, _ := testMintV2(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID)
	_, err = tt.supplierPPEventStream.Recv() // skip MintRequest
	require.NoError(t, err)

	// We're actually interested in this message which is
	// the TokenBoughtNotification
	eventMsg, err := tt.supplierPPEventStream.Recv()
	require.NoError(t, err)
	tt.DebugPrintProtoMessage(eventMsg)
	tokenBoughtNotification := &notificationv2.TokenBought{}
	require.NoError(t, proto.Unmarshal(eventMsg.Data, tokenBoughtNotification))
	require.Equal(t, tokenBoughtNotification.TokenId, tokenID)
	require.NotNil(t, tokenBoughtNotification.MintId)
	require.Equal(t, tokenBoughtNotification.MintId.Value, mintID)
	require.NotEmpty(t, tokenBoughtNotification.TxId)
}

func (tt *TestMintV2) testMintV2TokenExpiredCase(ctx context.Context, t *testing.T) {
	// Don't mind the eventStream receives without further processing.
	// We just receive all the messages from the pp-mock event stream without any
	// further checks as we're only really interested in the last one.

	searchID, resultID, totalPrice := testAccommodationV3SearchServiceWithTravelPeriod(ctx, t, tt.Environment, tt.distributorBotWithoutFunds, tt.supplierBot) // see test_accommodation_v3.go
	_, err := tt.supplierPPEventStream.Recv()                                                                                                                 // skip AccommodationSearchRequest
	require.NoError(t, err)

	validationID1 := testValidateV2(ctx, t, tt.Environment, tt.distributorBotWithoutFunds, tt.supplierBot, searchID, resultID, totalPrice)
	_, err = tt.supplierPPEventStream.Recv() // skip ValidateRequest
	require.NoError(t, err)

	searchID, resultID, totalPrice = testAccommodationV3SearchServiceWithTravelPeriod(ctx, t, tt.Environment, tt.distributorBotWithoutFunds, tt.supplierBot) // see test_accommodation_v3.go
	_, err = tt.supplierPPEventStream.Recv()                                                                                                                 // skip AccommodationSearchRequest
	require.NoError(t, err)

	validationID2 := testValidateV2(ctx, t, tt.Environment, tt.distributorBotWithoutFunds, tt.supplierBot, searchID, resultID, totalPrice)
	_, err = tt.supplierPPEventStream.Recv() // skip ValidateRequest
	require.NoError(t, err)

	var tokenID1 uint64
	var mintID1 string

	tokenID1, mintID1 = tt.testMintV2MintV2ExpectedError(ctx, t, validationID1)
	_, err = tt.supplierPPEventStream.Recv() // skip MintRequest
	require.NoError(t, err)

	tokenID2, mintID2 := tt.testMintV2MintV2ExpectedError(ctx, t, validationID2)
	_, err = tt.supplierPPEventStream.Recv() // skip MintRequest
	require.NoError(t, err)

	// Following code relies on specific order of token expired notifications.
	// We can safely assume that 2nd token expired notification will come after the first one,
	// because timeout is set by mint request, and 2nd mint is happening after 1st.

	eventMsg, err := tt.supplierPPEventStream.Recv()
	require.NoError(t, err)
	tt.DebugPrintProtoMessage(eventMsg)
	tokenExpiredNotification := &notificationv2.TokenExpired{}
	require.NoError(t, proto.Unmarshal(eventMsg.Data, tokenExpiredNotification))
	require.Equal(t, tokenExpiredNotification.TokenId, tokenID1)
	require.NotNil(t, tokenExpiredNotification.MintId)
	require.Equal(t, tokenExpiredNotification.MintId.Value, mintID1)

	eventMsg, err = tt.supplierPPEventStream.Recv()
	require.NoError(t, err)
	tt.DebugPrintProtoMessage(eventMsg)
	tokenExpiredNotification = &notificationv2.TokenExpired{}
	require.NoError(t, proto.Unmarshal(eventMsg.Data, tokenExpiredNotification))
	require.Equal(t, tokenExpiredNotification.TokenId, tokenID2)
	require.NotNil(t, tokenExpiredNotification.MintId)
	require.Equal(t, tokenExpiredNotification.MintId.Value, mintID2)
}

func (tt *TestMintV2) testMintV2MintV2ExpectedError(
	ctx context.Context,
	t *testing.T,
	validationID string,
) (
	tokenID uint64,
	mintID string,
) {
	req := &bookv2.MintRequest{
		Header:       &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		ValidationId: &typesv1.UUID{Value: validationID},
	}
	resp, err := tt.distributorBotWithoutFunds.MintServiceV2.Mint(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")

	// Check if the MintId is set
	require.NotEmpty(t, resp.MintId, "unexpected empty response MintId")
	require.NotEmpty(t, resp.MintId.Value, "unexpected empty response MintId.Value")

	require.NotEmpty(t, resp.MintTransactionId, "unexpected empty response MintTransactionId")
	require.Empty(t, resp.BuyTransactionId, "unexpected response BuyTransactionId")

	return resp.BookingTokenId, resp.MintId.Value
}

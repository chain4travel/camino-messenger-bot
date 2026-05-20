// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"
	"time"

	activityv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v5"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	typesv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v5"
	"buf.build/go/protovalidate"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v13/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/services/data"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/suite"
	"github.com/stretchr/testify/require"
)

var _ suite.Test = (*TestActivityV5)(nil)

func init() {
	Tests["Activityv5"] = &TestActivityV5{}
}

type TestActivityV5 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestActivityV5) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestActivityV5) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("Search with travel period oob", func(t *testing.T) {
		// ERROR path: with travel period outside of allowed constraints it should return an error
		tt.testActivityV5SearchServiceTravelPeriodOutOfBounds(ctx, t)
	})
	t.Run("Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		searchID, resultID, totalPrice := testActivityV5SearchService(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)
		validationID := testValidateV5(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		balanceBefore := tt.Balance(ctx, t, tt.distributorBot)
		tokenID, _, mintRespPrice := testMintV5(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID, common.BookingTokenPriceV5)
		verifyBookingTokenStateBoughtWithPriceV5(ctx, t, tt.Environment, tt.distributorBot, tokenID, mintRespPrice, balanceBefore)
	})
}

func (tt *TestActivityV5) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.ActivitySearchServiceV5,
		botGenerated.ValidationServiceV5,
		botGenerated.MintServiceV5,
	))

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.ActivitySearchServiceV5, Fee: 130},
			{Name: botGenerated.ValidationServiceV5, Fee: 140},
			{Name: botGenerated.MintServiceV5, Fee: 150},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

func (tt *TestActivityV5) testActivityV5SearchServiceTravelPeriodOutOfBounds(ctx context.Context, t *testing.T) {
	const nights = 12                                                                                              // 12 nights
	startDate := time.Now().Add(common.TravelPeriodMinStartOffset + common.TravelPeriodMaxDuration + 24*time.Hour) // outside of allowed travel period
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &activityv5.ActivitySearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{Currency: &typesv4.Currency_NativeToken{}},
			Language: typesv1.Language_LANGUAGE_EN,
		},
		SearchParametersActivity: &activityv5.ActivitySearchParameters{
			SupplierCodes: []*typesv4.SupplierProductCode{{Code: "XPTFAOH15O"}},
		},
		TravelPeriod: &typesv4.TravelPeriod{
			StartDate: common.TimeToDateV4(startDate),
			EndDate:   common.TimeToDateV4(endDate),
		},
		Travellers: []*typesv4.BasicTraveller{{
			Type: typesv4.TravellerType_TRAVELLER_TYPE_ADULT,
		}},
	}
	resp, err := tt.distributorBot.ActivitySearchServiceV5.ActivitySearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	require.True(t, resp.HasErrorResponse(), "unexpected response status")
}

// Test search with a valid travel period. Expect valid search results.
func testActivityV5SearchService(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) (
	searchID string,
	resultID uint32,
	totalPrice *typesv5.Price,
) {
	const nights = 12
	startDate := time.Now().Add(time.Hour * 24)
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))
	expectedSearchResults := []*activityv5.ActivitySearchResult{
		mockdata.ActivitySearchResultV5[0],
		mockdata.ActivitySearchResultV5[1],
	}

	req := &activityv5.ActivitySearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{
				Currency: &typesv4.Currency_IsoCurrency{IsoCurrency: typesv4.IsoCurrency_ISO_CURRENCY_EUR},
			},
			Language: typesv1.Language_LANGUAGE_EN,
		},
		SearchParametersActivity: &activityv5.ActivitySearchParameters{
			SupplierCodes: []*typesv4.SupplierProductCode{{Code: "XPTFAOH15O"}},
		},
		TravelPeriod: &typesv4.TravelPeriod{
			StartDate: common.TimeToDateV4(startDate),
			EndDate:   common.TimeToDateV4(endDate),
		},
		Travellers: []*typesv4.BasicTraveller{{
			TravellerId: 0,
			Type:        typesv4.TravellerType_TRAVELLER_TYPE_ADULT,
			Birthdate:   &typesv4.Date{Year: 1990, Month: 1, Day: 1},
			Nationality: typesv2.Country_COUNTRY_ES,
		}},
	}

	resp, err := distributorBot.ActivitySearchServiceV5.ActivitySearch(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")

	// check resultIDs and reset them for clean comparison with mock data
	for i, result := range successResp.Results {
		require.Equal(t, uint32(i), result.ResultId, "unexpected ResultId in response") //nolint:gosec
		result.ResultId = 0
	}

	requireProtoSlicesElementsMatch(t, expectedSearchResults, successResp.Results)

	return successResp.SearchId.Id.Value, successResp.Results[0].ResultId, successResp.Results[0].TotalPrice.Value
}

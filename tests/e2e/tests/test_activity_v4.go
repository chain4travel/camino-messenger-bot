// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"
	"time"

	activityv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v4"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/suite"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestActivityV4)(nil)

func init() {
	Tests["Activityv4"] = &TestActivityV4{}
}

type TestActivityV4 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestActivityV4) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestActivityV4) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("Product short list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testActivityV4ProductShortListService(ctx, t)
	})
	t.Run("Product short list with filter", func(t *testing.T) {
		// Happy path: will return only one property
		tt.testActivityV4ProductShortListServiceWithFilter(ctx, t)
	})
	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testActivityV4ProductListService(ctx, t)
	})
	t.Run("Product info", func(t *testing.T) {
		// Happy path: will return the detailed info of a property
		tt.testActivityV4ProductInfoService(ctx, t)
	})
	t.Run("Search with travel period oob", func(t *testing.T) {
		// ERROR path: with travel period outside of allowed constraints it should return an error
		tt.testActivityV4SearchServiceTravelPeriodOutOfBounds(ctx, t)
	})
	t.Run("Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		searchID, resultID, totalPrice := testActivityV4SearchService(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)
		validationID := testValidateV4(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		tokenID, _, price := testMintV4(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID, totalPrice)
		verifyBookingTokenStateWithPriceV4(ctx, t, tt.Environment, tt.distributorBot, tokenID, price)
	})
}

func (tt *TestActivityV4) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.ActivityProductShortListServiceV4,
		botGenerated.ActivityProductListServiceV4,
		botGenerated.ActivityProductInfoServiceV4,
		botGenerated.ActivitySearchServiceV4,
		botGenerated.ValidationServiceV4,
		botGenerated.MintServiceV4,
	))

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.ActivityProductShortListServiceV4, Fee: 100},
			{Name: botGenerated.ActivityProductListServiceV4, Fee: 110},
			{Name: botGenerated.ActivityProductInfoServiceV4, Fee: 120},
			{Name: botGenerated.ActivitySearchServiceV4, Fee: 130},
			{Name: botGenerated.ValidationServiceV4, Fee: 140},
			{Name: botGenerated.MintServiceV4, Fee: 150},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

// Simple product list request which shall return all activities. Checking if all are present
func (tt *TestActivityV4) testActivityV4ProductShortListService(ctx context.Context, t *testing.T) {
	expectedItems := make([]*activityv4.ActivityShortListItem, 0, len(mockdata.ActivityExtendedV4))
	for _, activity := range mockdata.ActivityExtendedV4 {
		expectedItems = append(expectedItems, &activityv4.ActivityShortListItem{
			SupplierCode: activity.Activity.SupplierCode,
			Status:       activity.Activity.Status,
		})
	}

	req := &activityv4.ActivityProductShortListRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
	}

	resp, err := tt.distributorBot.ActivityProductShortListServiceV4.ActivityProductShortList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	requireProtoSlicesElementsMatch(t, expectedItems, resp.ActivityShortListItems)
}

// Product list request with a modification filter set. It should only return one fitting result.
func (tt *TestActivityV4) testActivityV4ProductShortListServiceWithFilter(ctx context.Context, t *testing.T) {
	modifiedAfter := time.Unix(1710237631, 0)
	expectedItems := make([]*activityv4.ActivityShortListItem, 0, len(mockdata.ActivityExtendedV4))
	for _, activity := range mockdata.ActivityExtendedV4 {
		if activity.Activity.LastModified.AsTime().After(modifiedAfter) {
			expectedItems = append(expectedItems, &activityv4.ActivityShortListItem{
				SupplierCode: activity.Activity.SupplierCode,
				Status:       activity.Activity.Status,
			})
		}
	}
	require.NotEmpty(t, expectedItems)

	req := &activityv4.ActivityProductShortListRequest{
		Header:        &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		ModifiedAfter: timestamppb.New(modifiedAfter),
	}
	resp, err := tt.distributorBot.ActivityProductShortListServiceV4.ActivityProductShortList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	requireProtoSlicesElementsMatch(t, expectedItems, resp.ActivityShortListItems)
}

// Simple product list request which shall return all activities. Checking if all are present
func (tt *TestActivityV4) testActivityV4ProductListService(ctx context.Context, t *testing.T) {
	expectedItems := []*activityv4.ActivityInfo{mockdata.ActivityExtendedV4[1].Activity}

	req := &activityv4.ActivityProductListRequest{
		Header:        &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SupplierCodes: []*typesv4.SupplierProductCode{expectedItems[0].SupplierCode},
	}

	resp, err := tt.distributorBot.ActivityProductListServiceV4.ActivityProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	requireProtoSlicesElementsMatch(t, expectedItems, resp.Activities)
}

// Get detailed activity information for a specific supplier code.
func (tt *TestActivityV4) testActivityV4ProductInfoService(ctx context.Context, t *testing.T) {
	expectedSupplierCode := &typesv4.SupplierProductCode{Code: "XPTFAOH15O"}
	req := &activityv4.ActivityProductInfoRequest{
		Header:        &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SupplierCodes: []*typesv4.SupplierProductCode{expectedSupplierCode},
		Languages:     []typesv1.Language{typesv1.Language_LANGUAGE_EN},
	}
	resp, err := tt.distributorBot.ActivityProductInfoServiceV4.ActivityProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain only the one activity filtered in the request
	require.Len(t, resp.Activities, 1, "unexpected number of activities in response")

	expectedActivity := activityExtendedV4WithSupplierCode(t, mockdata.ActivityExtendedV4, expectedSupplierCode)
	require.True(t, proto.Equal(resp.Activities[0], expectedActivity), "activity fields does not match expected mock data activity, but their supplier codes match (%+v)", expectedSupplierCode)
}

func (tt *TestActivityV4) testActivityV4SearchServiceTravelPeriodOutOfBounds(ctx context.Context, t *testing.T) {
	const nights = 12                                 // 12 nights
	startDate := time.Now().Add(time.Hour * 24 * 100) // in 100 days, outside of allowed travel period
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &activityv4.ActivitySearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{Currency: &typesv4.Currency_NativeToken{}},
			Language: typesv1.Language_LANGUAGE_EN,
		},
		SearchParametersActivity: &activityv4.ActivitySearchParameters{
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
	resp, err := tt.distributorBot.ActivitySearchServiceV4.ActivitySearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search with a valid travel period. Expect valid search results.
func testActivityV4SearchService(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) (
	searchID string,
	resultID uint32,
	totalPrice *typesv4.Price,
) {
	const nights = 12
	startDate := time.Now().Add(time.Hour * 24)
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))
	expectedSearchResults := []*activityv4.ActivitySearchResult{
		mockdata.ActivitySearchResultV4[0],
		mockdata.ActivitySearchResultV4[1],
	}

	req := &activityv4.ActivitySearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{
				Currency: &typesv4.Currency_IsoCurrency{IsoCurrency: typesv4.IsoCurrency_ISO_CURRENCY_EUR},
			},
			Language: typesv1.Language_LANGUAGE_EN,
		},
		SearchParametersActivity: &activityv4.ActivitySearchParameters{
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

	resp, err := distributorBot.ActivitySearchServiceV4.ActivitySearch(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// check resultIDs and reset them for clean comparison with mock data
	for i, result := range resp.Results {
		require.Equal(t, uint32(i), result.ResultId, "unexpected ResultId in response") //nolint:gosec
		result.ResultId = 0
	}

	requireProtoSlicesElementsMatch(t, expectedSearchResults, resp.Results)

	return resp.SearchId.Id.Value, resp.Results[0].ResultId, resp.Results[0].TotalPrice.Value
}

func activityExtendedV4WithSupplierCode(
	t *testing.T,
	activities []*activityv4.ActivityExtendedInfo,
	supplierCode *typesv4.SupplierProductCode,
) *activityv4.ActivityExtendedInfo {
	for _, activity := range activities {
		if proto.Equal(activity.Activity.GetSupplierCode(), supplierCode) {
			return activity
		}
	}
	require.FailNow(t, "activity with supplier code not found", "supplier code: %s", supplierCode)
	return nil
}

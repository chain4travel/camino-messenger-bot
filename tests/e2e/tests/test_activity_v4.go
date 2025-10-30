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

var _ suite.Test = (*TestActivityv4)(nil)

func init() {
	Tests["Activityv4"] = &TestActivityv4{}
}

type TestActivityv4 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestActivityv4) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestActivityv4) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testActivityv4ProductListService(ctx, t)
	})
	t.Run("Product list with filter", func(t *testing.T) {
		// Happy path: will return only one property
		tt.testActivityv4ProductListServiceWithFilter(ctx, t)
	})
	t.Run("Product info", func(t *testing.T) {
		// Happy path: will return the detailed info of a property
		tt.testActivityv4ProductInfoService(ctx, t)
	})
	t.Run("Search with travel period oob", func(t *testing.T) {
		// ERROR path: with travel period outside of allowed constraints it should return an error
		tt.testActivityv4SearchServiceTravelPeriodOutOfBounds(ctx, t)
	})
	t.Run("Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		searchID, resultID, totalPrice := testActivityv4SearchService(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)
		validationID := testValidateV4(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		tokenID, _, price := testMintV4(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID, totalPrice)
		verifyBookingTokenStateWithPriceV4(ctx, t, tt.Environment, tt.distributorBot, tokenID, price)
	})
}

func (tt *TestActivityv4) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
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
			{Name: botGenerated.ActivityProductListServiceV4, Fee: 100},
			{Name: botGenerated.ActivityProductInfoServiceV4, Fee: 110},
			{Name: botGenerated.ActivitySearchServiceV4, Fee: 120},
			{Name: botGenerated.ValidationServiceV4, Fee: 130},
			{Name: botGenerated.MintServiceV4, Fee: 140},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

// Simple product list request which shall return all activities. Checking if all are present
func (tt *TestActivityv4) testActivityv4ProductListService(ctx context.Context, t *testing.T) {
	req := &activityv4.ActivityProductListRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
	}

	resp, err := tt.distributorBot.ActivityProductListServiceV4.ActivityProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	require.Len(t, resp.SupplierCodes, len(mockdata.ActivityExtendedV4), "unexpected number of supplier codes in response")

	expectedSupplierCodes := make([]*typesv4.SupplierProductCode, 0, len(mockdata.ActivityExtendedV4))
	for _, activity := range mockdata.ActivityExtendedV4 {
		expectedSupplierCodes = append(expectedSupplierCodes, activity.SupplierCode)
	}

	requireProtoSlicesElementsMatch(t, expectedSupplierCodes, resp.SupplierCodes)
}

// Product list request with a modification filter set. It should only return one fitting result.
func (tt *TestActivityv4) testActivityv4ProductListServiceWithFilter(ctx context.Context, t *testing.T) {
	const modifiedAfterSecs int64 = 1710237631
	expectedSupplierCodes := []*typesv4.SupplierProductCode{}
	for _, activity := range mockdata.ActivityExtendedV4 {
		if activity.LastModified.AsTime().Unix() > modifiedAfterSecs {
			expectedSupplierCodes = append(expectedSupplierCodes, activity.SupplierCode)
		}
	}
	require.NotEmpty(t, expectedSupplierCodes)

	req := &activityv4.ActivityProductListRequest{
		Header:        &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		ModifiedAfter: &timestamppb.Timestamp{Seconds: modifiedAfterSecs},
	}
	resp, err := tt.distributorBot.ActivityProductListServiceV4.ActivityProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	requireProtoSlicesElementsMatch(t, expectedSupplierCodes, resp.SupplierCodes)
}

// Get detailed activity information for a specific supplier code.
func (tt *TestActivityv4) testActivityv4ProductInfoService(ctx context.Context, t *testing.T) {
	expectedSupplierCode := &typesv4.SupplierProductCode{
		Code:   "XPTFAOH15O",
		Number: 31345,
	}
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

func (tt *TestActivityv4) testActivityv4SearchServiceTravelPeriodOutOfBounds(ctx context.Context, t *testing.T) {
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
func testActivityv4SearchService(
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
	const nights = 12                           // 12 nights
	startDate := time.Now().Add(time.Hour * 24) // tomorrow
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

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

	require.Len(t, resp.Results, 1, "unexpected number of results in response")
	require.Equal(t, resp.Results[0].ResultId, uint32(0), "unexpected ResultId in response")
	resultID = resp.Results[0].ResultId
	resp.Results[0].ResultId = 0 // Reset ResultId for comparison with mock data

	expectedActivity := activitySearchV4WithSupplierCode(t, mockdata.ActivitySearchResultV4, req.SearchParametersActivity.SupplierCodes[0])
	require.True(t, proto.Equal(resp.Results[0], expectedActivity), "activity fields does not match expected mock data activity, but their supplier codes match (%s)", req.SearchParametersActivity.SupplierCodes[0].Code)

	return resp.SearchId.Value, resultID, resp.Results[0].TotalPrice.Value
}

func activityExtendedV4WithSupplierCode(
	t *testing.T,
	activities []*activityv4.ActivityExtendedInfo,
	supplierCode *typesv4.SupplierProductCode,
) *activityv4.ActivityExtendedInfo {
	for _, activity := range activities {
		if proto.Equal(activity.GetSupplierCode(), supplierCode) {
			return activity
		}
	}
	require.FailNow(t, "activity with supplier code not found", "supplier code: %s", supplierCode)
	return nil
}

func activitySearchV4WithSupplierCode(
	t *testing.T,
	activities []*activityv4.ActivitySearchResult,
	supplierCode *typesv4.SupplierProductCode,
) *activityv4.ActivitySearchResult {
	for _, activity := range activities {
		if proto.Equal(activity.GetInfo().SupplierCode, supplierCode) {
			return activity
		}
	}
	require.FailNow(t, "activity with supplier code not found", "supplier code: %s", supplierCode)
	return nil
}

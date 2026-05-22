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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestActivityV5)(nil)

func init() {
	Tests["ActivityV5"] = &TestActivityV5{}
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

	t.Run("Product short list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testActivityV5ProductShortListService(ctx, t)
	})
	t.Run("Product short list with filter", func(t *testing.T) {
		// Happy path: will return only one property
		tt.testActivityV5ProductShortListServiceWithFilter(ctx, t)
	})
	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testActivityV5ProductListService(ctx, t)
	})
	t.Run("Product info", func(t *testing.T) {
		// Happy path: will return the detailed info of a property
		tt.testActivityV5ProductInfoService(ctx, t)
	})
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
		botGenerated.ActivityProductShortListServiceV5,
		botGenerated.ActivityProductListServiceV5,
		botGenerated.ActivityProductInfoServiceV5,
		botGenerated.ActivitySearchServiceV5,
		botGenerated.ValidationServiceV5,
		botGenerated.MintServiceV5,
	))

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.ActivityProductShortListServiceV5, Fee: 100},
			{Name: botGenerated.ActivityProductListServiceV5, Fee: 110},
			{Name: botGenerated.ActivityProductInfoServiceV5, Fee: 120},
			{Name: botGenerated.ActivitySearchServiceV5, Fee: 130},
			{Name: botGenerated.ValidationServiceV5, Fee: 140},
			{Name: botGenerated.MintServiceV5, Fee: 150},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

// Simple product list request which shall return all activities. Checking if all are present
func (tt *TestActivityV5) testActivityV5ProductShortListService(ctx context.Context, t *testing.T) {
	expectedItems := make([]*activityv5.ActivityShortListItem, 0, len(mockdata.ActivityExtendedV5))
	for _, activity := range mockdata.ActivityExtendedV5 {
		expectedItems = append(expectedItems, &activityv5.ActivityShortListItem{
			SupplierCode: activity.Activity.SupplierCode,
			Status:       activity.Activity.Status,
		})
	}

	req := &activityv5.ActivityProductShortListRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
	}

	resp, err := tt.distributorBot.ActivityProductShortListServiceV5.ActivityProductShortList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")

	requireProtoSlicesElementsMatch(t, expectedItems, successResp.ActivityShortListItems)
}

// Product list request with a modification filter set. It should only return one fitting result.
func (tt *TestActivityV5) testActivityV5ProductShortListServiceWithFilter(ctx context.Context, t *testing.T) {
	modifiedAfter := time.Unix(1710237631, 0)
	expectedItems := make([]*activityv5.ActivityShortListItem, 0, len(mockdata.ActivityExtendedV5))
	for _, activity := range mockdata.ActivityExtendedV5 {
		if activity.Activity.LastModified.AsTime().After(modifiedAfter) {
			expectedItems = append(expectedItems, &activityv5.ActivityShortListItem{
				SupplierCode: activity.Activity.SupplierCode,
				Status:       activity.Activity.Status,
			})
		}
	}
	require.NotEmpty(t, expectedItems)

	req := &activityv5.ActivityProductShortListRequest{
		Header:        &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		ModifiedAfter: timestamppb.New(modifiedAfter),
	}
	resp, err := tt.distributorBot.ActivityProductShortListServiceV5.ActivityProductShortList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")

	requireProtoSlicesElementsMatch(t, expectedItems, successResp.ActivityShortListItems)
}

// Simple product list request which shall return all activities. Checking if all are present
func (tt *TestActivityV5) testActivityV5ProductListService(ctx context.Context, t *testing.T) {
	expectedItems := []*activityv5.ActivityInfo{mockdata.ActivityExtendedV5[1].Activity}

	req := &activityv5.ActivityProductListRequest{
		Header:        &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SupplierCodes: []*typesv4.SupplierProductCode{expectedItems[0].SupplierCode},
	}

	resp, err := tt.distributorBot.ActivityProductListServiceV5.ActivityProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")

	requireProtoSlicesElementsMatch(t, expectedItems, successResp.Activities)
}

// Get detailed activity information for a specific supplier code.
func (tt *TestActivityV5) testActivityV5ProductInfoService(ctx context.Context, t *testing.T) {
	expectedSupplierCode := &typesv4.SupplierProductCode{Code: "XPTFAOH15O"}
	req := &activityv5.ActivityProductInfoRequest{
		Header:        &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SupplierCodes: []*typesv4.SupplierProductCode{expectedSupplierCode},
		Languages:     []typesv1.Language{typesv1.Language_LANGUAGE_EN},
	}
	resp, err := tt.distributorBot.ActivityProductInfoServiceV5.ActivityProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")

	// The response should contain only the one activity filtered in the request
	require.Len(t, successResp.Activities, 1, "unexpected number of activities in response")

	expectedActivity := activityExtendedV5WithSupplierCode(t, mockdata.ActivityExtendedV5, expectedSupplierCode)
	require.True(t, proto.Equal(successResp.Activities[0], expectedActivity), "activity fields does not match expected mock data activity, but their supplier codes match (%+v)", expectedSupplierCode)
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

func activityExtendedV5WithSupplierCode(
	t *testing.T,
	activities []*activityv5.ActivityExtendedInfo,
	supplierCode *typesv4.SupplierProductCode,
) *activityv5.ActivityExtendedInfo {
	for _, activity := range activities {
		if proto.Equal(activity.Activity.GetSupplierCode(), supplierCode) {
			return activity
		}
	}
	require.FailNow(t, "activity with supplier code not found", "supplier code: %s", supplierCode)
	return nil
}

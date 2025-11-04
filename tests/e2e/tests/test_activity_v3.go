// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	activityv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/suite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestActivityV3)(nil)

func init() {
	Tests["ActivityV3"] = &TestActivityV3{}
}

type TestActivityV3 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestActivityV3) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestActivityV3) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testActivityV3ProductListService(ctx, t)
	})
	t.Run("Product list with filter", func(t *testing.T) {
		// Happy path: will return only one property
		tt.testActivityV3ProductListServiceWithFilter(ctx, t)
	})
	t.Run("Product info", func(t *testing.T) {
		// Happy path: will return the detailed info of a property
		tt.testActivityV3ProductInfoService(ctx, t)
	})
	t.Run("Search w/o currency", func(t *testing.T) {
		// ERROR path: without currency it should return an error
		tt.testActivityV3SearchServiceWithoutCurrency(ctx, t)
	})
	t.Run("Search w/o travel period", func(t *testing.T) {
		// ERROR path: without travel period it should return an error
		tt.testActivityV3SearchServiceWithoutTravelPeriod(ctx, t)
	})
	t.Run("Search with travel period oob", func(t *testing.T) {
		// ERROR path: with travel period outside of allowed constraints it should return an error
		tt.testActivityV3SearchServiceTravelPeriodOutOfBounds(ctx, t)
	})
	t.Run("Search with travel period reversed", func(t *testing.T) {
		// ERROR path: with travel period reversed it should return an error
		tt.testActivityV3SearchServiceTravelPeriodReversed(ctx, t)
	})
	t.Run("Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		searchID, resultID, totalPrice := testActivityV3SearchServiceWithTravelPeriod(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)
		validationID := testValidateV2(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		tokenID, _, price := testMintV2(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID)
		verifyBookingTokenStateWithPriceV2(ctx, t, tt.Environment, tt.distributorBot, tokenID, price)
	})
}

func (tt *TestActivityV3) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.ActivityProductListServiceV3,
		botGenerated.ActivityProductInfoServiceV3,
		botGenerated.ActivitySearchServiceV3,
		botGenerated.ValidationServiceV2,
		botGenerated.MintServiceV2,
	))

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.ActivityProductListServiceV3, Fee: 100},
			{Name: botGenerated.ActivityProductInfoServiceV3, Fee: 110},
			{Name: botGenerated.ActivitySearchServiceV3, Fee: 120},
			{Name: botGenerated.ValidationServiceV2, Fee: 130},
			{Name: botGenerated.MintServiceV2, Fee: 140},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

// Simple product list request which shall return all activities. Checking if all are present
func (tt *TestActivityV3) testActivityV3ProductListService(ctx context.Context, t *testing.T) {
	req := &activityv3.ActivityProductListRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
	}

	resp, err := tt.distributorBot.ActivityProductListServiceV3.ActivityProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	require.Len(t, resp.Activities, len(mockdata.ActivityV3), "unexpected number of activities in response")

	expectedActivities := make([]*activityv3.Activity, 0, len(mockdata.ActivityV3))
	for _, activity := range resp.Activities {
		expectedActivities = append(expectedActivities, activityV3WithProductCode(t, mockdata.ActivityV3, activity.GetProductCode().GetCode()))
	}
	require.Len(t, expectedActivities, len(mockdata.ActivityV3), "not all expected activities found in response")

	for i, activity := range resp.Activities {
		require.True(t, proto.Equal(activity, expectedActivities[i]), "activities[%d] fields does not match expected mock data activity, but their product codes match (%s)", i, activity.GetProductCode().GetCode())
	}
}

// Product list request with a modification filter set. It should only return one fitting result.
func (tt *TestActivityV3) testActivityV3ProductListServiceWithFilter(ctx context.Context, t *testing.T) {
	// Modification timestamp which should exactly return one result (see expectedProductCode).
	// See the activityv3.json file in the pp-mock for more info
	const modifiedAfterSecs int64 = 1710237631
	const expectedProductCode = "TC000000"

	req := &activityv3.ActivityProductListRequest{
		Header:        &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		ModifiedAfter: &timestamppb.Timestamp{Seconds: modifiedAfterSecs},
	}
	resp, err := tt.distributorBot.ActivityProductListServiceV3.ActivityProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain only one activity as only one is modified after the given timestamp
	require.Len(t, resp.Activities, 1, "unexpected number of activities in response")

	require.Equal(t, expectedProductCode, resp.Activities[0].GetProductCode().GetCode(), "unexpected product code in response")
	require.Greater(t, resp.Activities[0].GetLastModified().GetSeconds(), modifiedAfterSecs, "activity timestamp is not after filter time")

	expectedActivity := activityV3WithProductCode(t, mockdata.ActivityV3, expectedProductCode)
	require.True(t, proto.Equal(resp.Activities[0], expectedActivity), "activity fields does not match expected mock data activity, but their product codes match (%s)", expectedProductCode)
}

// Get detailed activity information for a specific supplier code.
func (tt *TestActivityV3) testActivityV3ProductInfoService(ctx context.Context, t *testing.T) {
	req := &activityv3.ActivityProductInfoRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		// No filter to get all activities
	}

	resp, err := tt.distributorBot.ActivityProductInfoServiceV3.ActivityProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Len(t, resp.Activities, len(mockdata.ActivityExtendedV3), "unexpected number of activities in response")

	expectedActivities := make([]*activityv3.ActivityExtendedInfo, 0, len(mockdata.ActivityExtendedV3))
	for _, activity := range resp.Activities {
		expectedActivities = append(expectedActivities, activityExtendedV3WithSupplierCode(t, mockdata.ActivityExtendedV3, activity.GetSupplierCode()))
	}
	require.Len(t, expectedActivities, len(mockdata.ActivityExtendedV3), "not all expected activities found in response")

	for i, activity := range resp.Activities {
		require.True(t, proto.Equal(activity, expectedActivities[i]), "activities[%d] fields does not match expected mock data activity, but their supplier codes match (%+v)", i, activity.GetSupplierCode().GetSupplierCode())
	}

	expectedSupplierCode := &typesv2.SupplierProductCode{
		SupplierCode:   "XPTFAOH15O",
		SupplierNumber: 31345,
	}
	req = &activityv3.ActivityProductInfoRequest{
		Header:        &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SupplierCodes: []*typesv2.SupplierProductCode{expectedSupplierCode},
	}
	resp, err = tt.distributorBot.ActivityProductInfoServiceV3.ActivityProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain only the one activity filtered in the request
	require.Len(t, resp.Activities, 1, "unexpected number of activities in response")

	expectedActivity := activityExtendedV3WithSupplierCode(t, mockdata.ActivityExtendedV3, expectedSupplierCode)
	require.True(t, proto.Equal(resp.Activities[0], expectedActivity), "activity fields does not match expected mock data activity, but their supplier codes match (%+v)", expectedSupplierCode)
}

func (tt *TestActivityV3) testActivityV3SearchServiceWithoutCurrency(ctx context.Context, t *testing.T) {
	req := &activityv3.ActivitySearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Metadata: &typesv3.SearchRequestMetadata{
			RequestId: &typesv1.UUID{Value: uuid.New().String()},
		},
		SearchParametersGeneric: &typesv3.SearchParameters{},
	}

	resp, err := tt.distributorBot.ActivitySearchServiceV3.ActivitySearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)

	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

func (tt *TestActivityV3) testActivityV3SearchServiceWithoutTravelPeriod(ctx context.Context, t *testing.T) {
	req := &activityv3.ActivitySearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Metadata: &typesv3.SearchRequestMetadata{
			RequestId: &typesv1.UUID{Value: uuid.New().String()},
		},
		SearchParametersGeneric: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{Currency: &typesv3.Currency_NativeToken{}},
		},
		SearchParametersActivity: &activityv3.ActivitySearchParameters{
			ProductCodes: []*typesv2.ProductCode{{Code: "XPTFAOH15O"}},
		},
	}
	resp, err := tt.distributorBot.ActivitySearchServiceV3.ActivitySearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
	require.NotEmpty(t, resp.Header.Alerts, "unexpected empty response alerts")
	require.Equal(t, 1, len(resp.Header.Alerts), "unexpected number of alerts in response")
	require.Equal(t, typesv1.AlertType_ALERT_TYPE_ERROR, resp.Header.Alerts[0].Type, "unexpected alert type")
}

func (tt *TestActivityV3) testActivityV3SearchServiceTravelPeriodOutOfBounds(ctx context.Context, t *testing.T) {
	const nights = 12                                 // 12 nights
	startDate := time.Now().Add(time.Hour * 24 * 100) // in 100 days, outside of allowed travel period
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &activityv3.ActivitySearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Metadata: &typesv3.SearchRequestMetadata{
			RequestId: &typesv1.UUID{Value: uuid.New().String()},
		},
		SearchParametersGeneric: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{Currency: &typesv3.Currency_NativeToken{}},
		},
		SearchParametersActivity: &activityv3.ActivitySearchParameters{
			ProductCodes: []*typesv2.ProductCode{{Code: "XPTFAOH15O"}},
		},
		TravelPeriod: &typesv1.TravelPeriod{
			StartDate: common.TimeToDateV1(startDate),
			EndDate:   common.TimeToDateV1(endDate),
		},
	}
	resp, err := tt.distributorBot.ActivitySearchServiceV3.ActivitySearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

func (tt *TestActivityV3) testActivityV3SearchServiceTravelPeriodReversed(ctx context.Context, t *testing.T) {
	const nights = 12                           // 12 nights
	startDate := time.Now().Add(time.Hour * 24) // tomorrow
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &activityv3.ActivitySearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Metadata: &typesv3.SearchRequestMetadata{
			RequestId: &typesv1.UUID{Value: uuid.New().String()},
		},
		SearchParametersGeneric: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{Currency: &typesv3.Currency_NativeToken{}},
		},
		SearchParametersActivity: &activityv3.ActivitySearchParameters{
			ProductCodes: []*typesv2.ProductCode{{Code: "XPTFAOH15O"}},
		},
		TravelPeriod: &typesv1.TravelPeriod{
			StartDate: common.TimeToDateV1(endDate),   // End date used as start
			EndDate:   common.TimeToDateV1(startDate), // Start date used as end
		},
	}
	resp, err := tt.distributorBot.ActivitySearchServiceV3.ActivitySearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
	require.NotEmpty(t, resp.Header.Alerts, "unexpected empty response alerts")
}

// Test search with a valid travel period. Expect valid search results.
func testActivityV3SearchServiceWithTravelPeriod(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) (
	searchID string,
	resultID int32,
	totalPrice *big.Int,
) {
	const nights = 12                           // 12 nights
	startDate := time.Now().Add(time.Hour * 24) // tomorrow
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &activityv3.ActivitySearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Metadata: &typesv3.SearchRequestMetadata{
			RequestId: &typesv1.UUID{Value: uuid.New().String()},
		},
		SearchParametersGeneric: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{
				Currency: &typesv3.Currency_IsoCurrency{IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR},
			},
		},
		SearchParametersActivity: &activityv3.ActivitySearchParameters{
			ProductCodes: []*typesv2.ProductCode{{Code: "XPTFAOH15O"}},
			ServiceCodes: []string{"XO"},
		},
		TravelPeriod: &typesv1.TravelPeriod{
			StartDate: common.TimeToDateV1(startDate),
			EndDate:   common.TimeToDateV1(endDate),
		},
		Travellers: []*typesv3.BasicTraveller{
			{
				TravellerId: 0,
				Type:        typesv3.TravellerType_TRAVELLER_TYPE_ADULT,
				Birthdate:   &typesv1.Date{Year: 1990, Month: 1, Day: 1},
				Nationality: typesv2.Country_COUNTRY_ES,
			},
		},
	}

	resp, err := distributorBot.ActivitySearchServiceV3.ActivitySearch(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	require.Len(t, resp.Results, 1, "unexpected number of results in response")
	require.Equal(t, resp.Results[0].ResultId, int32(1), "unexpected ResultId in response")
	resultID = resp.Results[0].ResultId
	resp.Results[0].ResultId = 0 // Reset ResultId for comparison with mock data

	expectedActivity := activitySearchV3WithProductCode(t, mockdata.ActivitySearchResultV3, req.SearchParametersActivity.ProductCodes[0].Code)
	require.True(t, proto.Equal(resp.Results[0], expectedActivity), "activity fields does not match expected mock data activity, but their product codes match (%s)", req.SearchParametersActivity.ProductCodes[0].Code)

	return resp.Metadata.SearchId.Value, resultID, protoPriceBigV3(t, resp.Results[0].Price)
}

func activityV3WithProductCode(
	t *testing.T,
	activities []*activityv3.Activity,
	productCode string,
) *activityv3.Activity {
	for _, activity := range activities {
		if activity.GetProductCode().GetCode() == productCode {
			return activity
		}
	}
	require.FailNow(t, "activity with product code not found", "product code: %s", productCode)
	return nil
}

func activityExtendedV3WithSupplierCode(
	t *testing.T,
	activities []*activityv3.ActivityExtendedInfo,
	supplierCode *typesv2.SupplierProductCode,
) *activityv3.ActivityExtendedInfo {
	for _, activity := range activities {
		if proto.Equal(activity.GetSupplierCode(), supplierCode) {
			return activity
		}
	}
	require.FailNow(t, "activity with supplier code not found", "supplier code: %s", supplierCode)
	return nil
}

func activitySearchV3WithProductCode(
	t *testing.T,
	activities []*activityv3.ActivitySearchResult,
	productCode string,
) *activityv3.ActivitySearchResult {
	for _, activity := range activities {
		if activity.GetInfo().GetProductCode().GetCode() == productCode {
			return activity
		}
	}
	require.FailNow(t, "activity with product code not found", "product code: %s", productCode)
	return nil
}

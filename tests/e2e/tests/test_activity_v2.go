// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
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

var _ suite.Test = (*TestActivityV2)(nil)

func init() {
	Tests["ActivityV2"] = &TestActivityV2{}
}

type TestActivityV2 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestActivityV2) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestActivityV2) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testActivityV2ProductListService(ctx, t)
	})
	t.Run("Product list with filter", func(t *testing.T) {
		// Happy path: will return only one property
		tt.testActivityV2ProductListServiceWithFilter(ctx, t)
	})
	t.Run("Product info", func(t *testing.T) {
		// Happy path: will return the detailed info of a property
		tt.testActivityV2ProductInfoService(ctx, t)
	})
	t.Run("Search w/o currency", func(t *testing.T) {
		// ERROR path: without currency it should return an error
		tt.testActivityV2SearchServiceWithoutCurrency(ctx, t)
	})
	t.Run("Search w/o travel period", func(t *testing.T) {
		// ERROR path: without travel period it should return an error
		tt.testActivityV2SearchServiceWithoutTravelPeriod(ctx, t)
	})
	t.Run("Search with travel period oob", func(t *testing.T) {
		// ERROR path: with travel period outside of allowed constraints it should return an error
		tt.testActivityV2SearchServiceTravelPeriodOutOfBounds(ctx, t)
	})
	t.Run("Search with travel period reversed", func(t *testing.T) {
		// ERROR path: with travel period reversed it should return an error
		tt.testActivityV2SearchServiceTravelPeriodReversed(ctx, t)
	})
	t.Run("Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		searchID, resultID, totalPrice := testActivityV2SearchServiceWithTravelPeriod(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)
		validationID := testValidateV2(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		tokenID, _, price := testMintV2(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID)
		verifyBookingTokenStateWithPriceV2(ctx, t, tt.Environment, tt.distributorBot, tokenID, price)
	})
}

func (tt *TestActivityV2) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.ActivityProductListServiceV2,
		botGenerated.ActivityProductInfoServiceV2,
		botGenerated.ActivitySearchServiceV2,
		botGenerated.ValidationServiceV2,
		botGenerated.MintServiceV2,
	))

	wg := sync.WaitGroup{}

	// bot with partnerPlugin and without rpc server (supplier)
	wg.Add(1)
	go func() {
		defer wg.Done()
		tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
		tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
			bot.WithServices([]bot.CMService{
				{Name: botGenerated.ActivityProductListServiceV2, Fee: 100},
				{Name: botGenerated.ActivityProductInfoServiceV2, Fee: 110},
				{Name: botGenerated.ActivitySearchServiceV2, Fee: 120},
				{Name: botGenerated.ValidationServiceV2, Fee: 130},
				{Name: botGenerated.MintServiceV2, Fee: 140},
			}),
		)
	}()

	// bot without partnerPlugin and with rpc server (distributor)
	wg.Add(1)
	go func() {
		defer wg.Done()
		tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
	}()

	wg.Wait()
}

// Simple product list request which shall return all activities. Checking if all are present
func (tt *TestActivityV2) testActivityV2ProductListService(ctx context.Context, t *testing.T) {
	req := &activityv2.ActivityProductListRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
	}

	resp, err := tt.distributorBot.ActivityProductListServiceV2.ActivityProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	require.Len(t, resp.Activities, len(mockdata.ActivityV2), "unexpected number of activities in response")

	expectedActivities := make([]*activityv2.Activity, 0, len(mockdata.ActivityV2))
	for _, activity := range resp.Activities {
		expectedActivities = append(expectedActivities, activityV2WithProductCode(t, mockdata.ActivityV2, activity.GetProductCode().GetCode()))
	}
	require.Len(t, expectedActivities, len(mockdata.ActivityV2), "not all expected activities found in response")

	for i, activity := range resp.Activities {
		require.True(t, proto.Equal(activity, expectedActivities[i]), "activities[%d] fields does not match expected mock data activity, but their product codes match (%s)", i, activity.GetProductCode().GetCode())
	}
}

// Product list request with a modification filter set. It should only return one fitting result.
func (tt *TestActivityV2) testActivityV2ProductListServiceWithFilter(ctx context.Context, t *testing.T) {
	// Modification timestamp which should exactly return one result (see expectedProductCode).
	// See the activityv2.json file in the pp-mock for more info
	const modifiedAfterSecs int64 = 1710237631
	const expectedProductCode = "TC000000"

	req := &activityv2.ActivityProductListRequest{
		Header:        &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		ModifiedAfter: &timestamppb.Timestamp{Seconds: modifiedAfterSecs},
	}
	resp, err := tt.distributorBot.ActivityProductListServiceV2.ActivityProductList(
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

	expectedActivity := activityV2WithProductCode(t, mockdata.ActivityV2, expectedProductCode)
	require.True(t, proto.Equal(resp.Activities[0], expectedActivity), "activity fields does not match expected mock data activity, but their product codes match (%s)", expectedProductCode)
}

// Get detailed activity information for a specific supplier code.
func (tt *TestActivityV2) testActivityV2ProductInfoService(ctx context.Context, t *testing.T) {
	req := &activityv2.ActivityProductInfoRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		// No filter to get all activities
	}

	resp, err := tt.distributorBot.ActivityProductInfoServiceV2.ActivityProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Len(t, resp.Activities, len(mockdata.ActivityExtendedV2), "unexpected number of activities in response")

	expectedActivities := make([]*activityv2.ActivityExtendedInfo, 0, len(mockdata.ActivityExtendedV2))
	for _, activity := range resp.Activities {
		expectedActivities = append(expectedActivities, activityExtendedV2WithSupplierCode(t, mockdata.ActivityExtendedV2, activity.GetSupplierCode()))
	}
	require.Len(t, expectedActivities, len(mockdata.ActivityExtendedV2), "not all expected activities found in response")

	for i, activity := range resp.Activities {
		require.True(t, proto.Equal(activity, expectedActivities[i]), "activities[%d] fields does not match expected mock data activity, but their supplier codes match (%+v)", i, activity.GetSupplierCode().GetSupplierCode())
	}

	expectedSupplierCode := &typesv2.SupplierProductCode{
		SupplierCode:   "XPTFAOH15O",
		SupplierNumber: 31345,
	}
	req = &activityv2.ActivityProductInfoRequest{
		Header:        &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SupplierCodes: []*typesv2.SupplierProductCode{expectedSupplierCode},
	}
	resp, err = tt.distributorBot.ActivityProductInfoServiceV2.ActivityProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain only the one activity filtered in the request
	require.Len(t, resp.Activities, 1, "unexpected number of activities in response")

	expectedActivity := activityExtendedV2WithSupplierCode(t, mockdata.ActivityExtendedV2, expectedSupplierCode)
	require.True(t, proto.Equal(resp.Activities[0], expectedActivity), "activity fields does not match expected mock data activity, but their supplier codes match (%+v)", expectedSupplierCode)
}

func (tt *TestActivityV2) testActivityV2SearchServiceWithoutCurrency(ctx context.Context, t *testing.T) {
	req := &activityv2.ActivitySearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Metadata: &typesv2.SearchRequestMetadata{
			RequestId: &typesv1.UUID{Value: uuid.New().String()},
		},
		SearchParametersGeneric: &typesv2.SearchParameters{},
	}

	resp, err := tt.distributorBot.ActivitySearchServiceV2.ActivitySearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)

	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

func (tt *TestActivityV2) testActivityV2SearchServiceWithoutTravelPeriod(ctx context.Context, t *testing.T) {
	req := &activityv2.ActivitySearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Metadata: &typesv2.SearchRequestMetadata{
			RequestId: &typesv1.UUID{Value: uuid.New().String()},
		},
		SearchParametersGeneric: &typesv2.SearchParameters{
			Currency: &typesv2.Currency{Currency: &typesv2.Currency_NativeToken{}},
		},
		SearchParametersActivity: &activityv2.ActivitySearchParameters{
			ProductCodes: []*typesv2.ProductCode{{Code: "XPTFAOH15O"}},
		},
	}
	resp, err := tt.distributorBot.ActivitySearchServiceV2.ActivitySearch(
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

func (tt *TestActivityV2) testActivityV2SearchServiceTravelPeriodOutOfBounds(ctx context.Context, t *testing.T) {
	const nights = 12                                 // 12 nights
	startDate := time.Now().Add(time.Hour * 24 * 100) // in 100 days, outside of allowed travel period
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &activityv2.ActivitySearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Metadata: &typesv2.SearchRequestMetadata{
			RequestId: &typesv1.UUID{Value: uuid.New().String()},
		},
		SearchParametersGeneric: &typesv2.SearchParameters{
			Currency: &typesv2.Currency{Currency: &typesv2.Currency_NativeToken{}},
		},
		SearchParametersActivity: &activityv2.ActivitySearchParameters{
			ProductCodes: []*typesv2.ProductCode{{Code: "XPTFAOH15O"}},
		},
		TravelPeriod: &typesv1.TravelPeriod{
			StartDate: common.TimeToDateV1(startDate),
			EndDate:   common.TimeToDateV1(endDate),
		},
	}
	resp, err := tt.distributorBot.ActivitySearchServiceV2.ActivitySearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

func (tt *TestActivityV2) testActivityV2SearchServiceTravelPeriodReversed(ctx context.Context, t *testing.T) {
	const nights = 12                           // 12 nights
	startDate := time.Now().Add(time.Hour * 24) // tomorrow
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &activityv2.ActivitySearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Metadata: &typesv2.SearchRequestMetadata{
			RequestId: &typesv1.UUID{Value: uuid.New().String()},
		},
		SearchParametersGeneric: &typesv2.SearchParameters{
			Currency: &typesv2.Currency{Currency: &typesv2.Currency_NativeToken{}},
		},
		SearchParametersActivity: &activityv2.ActivitySearchParameters{
			ProductCodes: []*typesv2.ProductCode{{Code: "XPTFAOH15O"}},
		},
		TravelPeriod: &typesv1.TravelPeriod{
			StartDate: common.TimeToDateV1(endDate),   // End date used as start
			EndDate:   common.TimeToDateV1(startDate), // Start date used as end
		},
	}
	resp, err := tt.distributorBot.ActivitySearchServiceV2.ActivitySearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
	require.NotEmpty(t, resp.Header.Alerts, "unexpected empty response alerts")
}

// Test search with a valid travel period. Expect valid search results.
func testActivityV2SearchServiceWithTravelPeriod(
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

	req := &activityv2.ActivitySearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Metadata: &typesv2.SearchRequestMetadata{
			RequestId: &typesv1.UUID{Value: uuid.New().String()},
		},
		SearchParametersGeneric: &typesv2.SearchParameters{
			Currency: &typesv2.Currency{
				Currency: &typesv2.Currency_IsoCurrency{IsoCurrency: typesv2.IsoCurrency_ISO_CURRENCY_EUR},
			},
		},
		SearchParametersActivity: &activityv2.ActivitySearchParameters{
			ProductCodes: []*typesv2.ProductCode{{Code: "XPTFAOH15O"}},
			ServiceCodes: []string{"XO"},
		},
		TravelPeriod: &typesv1.TravelPeriod{
			StartDate: common.TimeToDateV1(startDate),
			EndDate:   common.TimeToDateV1(endDate),
		},
		Travellers: []*typesv2.BasicTraveller{
			{
				TravellerId: 0,
				Type:        typesv2.TravellerType_TRAVELLER_TYPE_ADULT,
				Birthdate:   &typesv1.Date{Year: 1990, Month: 1, Day: 1},
				Nationality: typesv2.Country_COUNTRY_ES,
			},
		},
	}

	resp, err := distributorBot.ActivitySearchServiceV2.ActivitySearch(
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

	expectedActivity := activitySearchV2WithProductCode(t, mockdata.ActivitySearchResultV2, req.SearchParametersActivity.ProductCodes[0].Code)
	require.True(t, proto.Equal(resp.Results[0], expectedActivity), "activity fields does not match expected mock data activity, but their product codes match (%s)", req.SearchParametersActivity.ProductCodes[0].Code)

	return resp.Metadata.SearchId.Value, resultID, priceBigV2(t, resp.Results[0].Price)
}

func activityV2WithProductCode(
	t *testing.T,
	activities []*activityv2.Activity,
	productCode string,
) *activityv2.Activity {
	for _, activity := range activities {
		if activity.GetProductCode().GetCode() == productCode {
			return activity
		}
	}
	require.FailNow(t, "activity with product code not found", "product code: %s", productCode)
	return nil
}

func activityExtendedV2WithSupplierCode(
	t *testing.T,
	activities []*activityv2.ActivityExtendedInfo,
	supplierCode *typesv2.SupplierProductCode,
) *activityv2.ActivityExtendedInfo {
	for _, activity := range activities {
		if proto.Equal(activity.GetSupplierCode(), supplierCode) {
			return activity
		}
	}
	require.FailNow(t, "activity with supplier code not found", "supplier code: %s", supplierCode)
	return nil
}

func activitySearchV2WithProductCode(
	t *testing.T,
	activities []*activityv2.ActivitySearchResult,
	productCode string,
) *activityv2.ActivitySearchResult {
	for _, activity := range activities {
		if activity.GetInfo().GetProductCode().GetCode() == productCode {
			return activity
		}
	}
	require.FailNow(t, "activity with product code not found", "product code: %s", productCode)
	return nil
}

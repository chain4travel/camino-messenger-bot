// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
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
		tokenID, price, _ := testMintV2(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID)
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

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.ActivityProductListServiceV2, Fee: 100},
			{Name: botGenerated.ActivityProductInfoServiceV2, Fee: 110},
			{Name: botGenerated.ActivitySearchServiceV2, Fee: 120},
			{Name: botGenerated.ValidationServiceV2, Fee: 130},
			{Name: botGenerated.MintServiceV2, Fee: 140},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

const activityV2ProductCode = "XPTFAOH15O"

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

// Get detailed activity information for a specific product code.
func (tt *TestActivityV2) testActivityV2ProductInfoService(ctx context.Context, t *testing.T) {
	req := &activityv2.ActivityProductInfoRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		// No filter to get all activities
	}

	allActivitiesResp, err := tt.distributorBot.ActivityProductInfoServiceV2.ActivityProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, allActivitiesResp)

	supplierCode1 := activityV2ProductCode

	req2 := &activityv2.ActivityProductInfoRequest{
		Header:        &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SupplierCodes: []*typesv2.SupplierProductCode{{SupplierCode: supplierCode1}},
	}
	resp, err := tt.distributorBot.ActivityProductInfoServiceV2.ActivityProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req2,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain only the one activity filtered in the request
	require.Len(t, resp.Activities, 1, "unexpected number of activities in response")
	activity := resp.Activities[0]

	// Validate the activity data
	require.NotNil(t, activity.Activity, "unexpected nil Activity in response")
	require.NotEmpty(t, activity.Activity.Context, "unexpected empty activity Context")

	// Check supplier code
	require.NotNil(t, activity.SupplierCode, "unexpected nil SupplierCode")
	require.Equal(t, supplierCode1, activity.SupplierCode.SupplierCode, "unexpected SupplierCode value")

	// Check additional activity data
	require.NotEmpty(t, activity.CategoryCode, "unexpected empty CategoryCode")
	require.NotEmpty(t, activity.CategoryName, "unexpected empty CategoryName")
	require.NotEmpty(t, activity.TypeCode, "unexpected empty TypeCode")
	require.NotEmpty(t, activity.TypeName, "unexpected empty TypeName")

	// Check location data
	require.NotNil(t, activity.Location, "unexpected nil Location")
	require.NotNil(t, activity.Location.Address, "unexpected nil Address")

	// Check units
	require.NotEmpty(t, activity.Units, "unexpected empty Units")
	require.NotEmpty(t, activity.Units[0].Code, "unexpected empty unit Code")
	require.NotEmpty(t, activity.Units[0].Name, "unexpected empty unit Name")

	// Check services
	require.NotEmpty(t, activity.Services, "unexpected empty Services")
	require.NotEmpty(t, activity.Services[0].Code, "unexpected empty service Code")
	require.NotEmpty(t, activity.Services[0].Name, "unexpected empty service Name")

	// Check zones and pickup/dropoff events if available
	if len(activity.Zones) > 0 {
		require.NotEmpty(t, activity.Zones[0].Code, "unexpected empty zone Code")
		if len(activity.Zones[0].PickupDropoffEvents) > 0 {
			event := activity.Zones[0].PickupDropoffEvents[0]
			require.NotEmpty(t, event.LocationCode, "unexpected empty LocationCode")
			require.NotEmpty(t, event.LocationName, "unexpected empty LocationName")
		}
	}

	// Check media
	require.NotEmpty(t, activity.Images, "unexpected empty Images")
	require.NotEmpty(t, activity.Images[0].File, "unexpected empty image File")
	require.NotEmpty(t, activity.Images[0].Width, "unexpected empty image Width")
	require.NotEmpty(t, activity.Images[0].Height, "unexpected empty image Height")
	require.NotEmpty(t, activity.Images[0].Category, "unexpected empty image Category")

	// Check features and tags
	require.NotEmpty(t, activity.Features, "unexpected empty Features")
	require.NotEmpty(t, activity.Tags, "unexpected empty Tags")

	// Check availability and delivery options
	require.True(t, activity.InstantConfirmation, "unexpected InstantConfirmation value")
	require.NotEmpty(t, activity.DeliveryFormats, "unexpected empty DeliveryFormats")
	require.NotEmpty(t, activity.DeliveryMethods, "unexpected empty DeliveryMethods")

	// NEW CHECKS START HERE

	// Check descriptions
	require.NotEmpty(t, activity.Descriptions, "unexpected empty Descriptions")
	if len(activity.Descriptions) > 0 {
		require.NotNil(t, activity.Descriptions[0], "unexpected nil Description")
		require.NotEmpty(t, activity.Descriptions[0].Descriptions, "unexpected empty Description texts")
	}

	// Check contact info
	require.NotNil(t, activity.ContactInfo, "unexpected nil ContactInfo")
	if activity.ContactInfo != nil {
		require.NotNil(t, activity.ContactInfo.Address, "unexpected nil ContactInfo.Address")
		require.NotEmpty(t, activity.ContactInfo.Emails, "unexpected empty ContactInfo.Emails")
		require.NotNil(t, activity.ContactInfo.Phones, "unexpected nil ContactInfo.Phones")
	}

	// Check videos if available
	if len(activity.Videos) > 0 {
		require.NotEmpty(t, activity.Videos[0].File, "unexpected empty video File")
		require.NotEmpty(t, activity.Videos[0].Category, "unexpected empty video Category")
	}

	// Check languages
	require.NotEmpty(t, activity.Languages, "unexpected empty Languages")

	// Check duration range
	require.NotNil(t, activity.DurationRange, "unexpected nil DurationRange")
	if activity.DurationRange != nil {
		require.NotNil(t, activity.DurationRange.MinDuration, "unexpected nil MinDuration")
		require.NotNil(t, activity.DurationRange.MaxDuration, "unexpected nil MaxDuration")
	}

	// Check max confirmation duration
	require.NotNil(t, activity.MaxConfirmationDuration, "unexpected nil MaxConfirmationDuration")

	// Check redemption methods if available
	if len(activity.RedemptionMethods) > 0 {
		require.NotEmpty(t, activity.RedemptionMethods, "unexpected empty RedemptionMethods")
	}

	// Check specific feature content (at least one feature should have a meaningful code and description)
	featureFound := false
	for _, feature := range activity.Features {
		if feature.Code != "" && feature.Description != "" {
			featureFound = true
			break
		}
	}
	require.True(t, featureFound, "no feature with valid code and description found")

	// Check specific tag content (at least one tag should have a name and slug)
	tagFound := false
	for _, tag := range activity.Tags {
		if tag.Name != "" && tag.Slug != "" {
			tagFound = true
			break
		}
	}
	require.True(t, tagFound, "no tag with valid name and slug found")

	// Check supplier code name
	require.NotEmpty(t, activity.SupplierCodeName, "unexpected empty SupplierCodeName")

	// Check availability related fields
	require.NotEmpty(t, activity.AvailabilityType, "unexpected empty AvailabilityType")
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
			ProductCodes: []*typesv2.ProductCode{{Code: activityV2ProductCode}},
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
			ProductCodes: []*typesv2.ProductCode{{Code: activityV2ProductCode}},
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
			ProductCodes: []*typesv2.ProductCode{{Code: activityV2ProductCode}},
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
			Currency: &typesv2.Currency{Currency: &typesv2.Currency_NativeToken{}},
		},
		SearchParametersActivity: &activityv2.ActivitySearchParameters{
			ProductCodes: []*typesv2.ProductCode{{Code: activityV2ProductCode}},
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

	// We expect results - check at least one exists
	require.NotEmpty(t, resp.Results, "unexpected empty results in response")

	// Let's check if the first result is as expected
	require.NotEmpty(t, resp.Results[0].ResultId, "unexpected empty response Results[0].ResultId")
	require.NotEmpty(t, resp.Results[0].Info, "unexpected empty response Results[0].Info")

	// Extract the total price from the response
	require.NotEmpty(t, resp.Results[0], "unexpected empty TotalPriceDetail")
	require.NotEmpty(t, resp.Results[0].Price, "unexpected empty Price")

	totalPrice = priceBigV2(t, resp.Results[0].Price)

	// Check if this adds up with the total price of the unit // TODO@ copy pasted from accommodation v3, does it make sense here?
	expectedTotalPrice := big.NewInt(0).Mul(common.DefaultPricePerNightNativeTokenBig, big.NewInt(nights))
	require.True(t, totalPrice.Cmp(expectedTotalPrice) == 0, "unexpected total price: got %s, expected %s", totalPrice.String(), expectedTotalPrice.String())

	// Now extract all the values needed for the validate step which comes next
	require.NotEmpty(t, resp.Metadata, "unexpected empty response Metadata")
	require.NotEmpty(t, resp.Metadata.SearchId, "unexpected empty response Metadata.SearchId")
	require.NotEmpty(t, resp.Metadata.SearchId.Value, "unexpected empty response Metadata.SearchId.Value")

	return resp.Metadata.SearchId.Value, resp.Results[0].ResultId, totalPrice
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

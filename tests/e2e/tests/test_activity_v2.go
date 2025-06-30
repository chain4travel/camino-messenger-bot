// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"strconv"
	"testing"
	"time"

	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/booking"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const activityV2ProductCode = "XPTFAOH15O"

// Setting up the basic applications and services used in all sub-test-cases
func testActivityV2Setup(
	ctx context.Context,
	t *testing.T,
	tt *Test,
) (
	supplierPartnerPlugin *partnerplugin.PartnerPlugin,
	supplierBot *bot.Bot,
	distributorBot *bot.Bot,
) {
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.ActivityProductListServiceV2,
		botGenerated.ActivityProductInfoServiceV2,
		botGenerated.ActivitySearchServiceV2,
		botGenerated.ValidationServiceV2,
		botGenerated.MintServiceV2,
	))
	supplierPartnerPlugin = tt.createPartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	supplierBot = tt.createBot(ctx, t, false, supplierPartnerPlugin, []bot.CMService{
		{Name: botGenerated.ActivityProductListServiceV2, Fee: 100},
		{Name: botGenerated.ActivityProductInfoServiceV2, Fee: 110},
		{Name: botGenerated.ActivitySearchServiceV2, Fee: 120},
		{Name: botGenerated.ValidationServiceV2, Fee: 130},
		{Name: botGenerated.MintServiceV2, Fee: 140},
	})

	// bot without partnerPlugin and with rpc server (distributor)
	distributorBot = tt.createBot(ctx, t, true, nil, nil)

	return supplierPartnerPlugin, supplierBot, distributorBot
}

// Simple product list request which shall return all activities. Checking if all are present
func testActivityV2ProductListService(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
	activityProductCodes := []string{
		"TC000000",
		"ACTIVITY345678",
		"87456",
	}

	expectedTotalResults := len(activityProductCodes)

	req := &activityv2.ActivityProductListRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
	}

	resp, err := distributorBot.ActivityProductListServiceV2.ActivityProductList(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)

	tt.logger.Debug("ActivityProductListServiceV2.ActivityProductList response:\n", protoMessageToJSON(tt, resp))
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain all activities defined in our expectation
	require.Len(t, resp.Activities, expectedTotalResults, "unexpected number of activities in response")

	for i := range resp.Activities {
		require.NotNil(t, resp.Activities[i].ProductCode, "unexpected nil response activities[%d].ProductCode", i)
		require.NotEmpty(t, resp.Activities[i].ProductCode.Code, "unexpected empty response activities[%d].ProductCode.Code", i)
		require.Contains(t, activityProductCodes, resp.Activities[i].ProductCode.Code,
			"unexpected response activities[%d].ProductCode.Code: %s", i, resp.Activities[i].ProductCode.Code)

		require.NotEmpty(t, resp.Activities[i].Context, "activities[%d].Context should not be empty", i)
		require.NotNil(t, resp.Activities[i].LastModified, "activities[%d].LastModified should not be nil", i)
		require.NotEmpty(t, resp.Activities[i].ExternalSessionId, "activities[%d].ExternalSessionId should not be empty", i)
		require.NotEmpty(t, resp.Activities[i].UnitCode, "activities[%d].UnitCode should not be empty", i)
		require.NotEmpty(t, resp.Activities[i].ServiceCode, "activities[%d].ServiceCode should not be empty", i)
		require.NotNil(t, resp.Activities[i].Bookability, "activities[%d].Bookability should not be nil", i)
	}

	// Make sure every expected product code is found in the response
	foundCodes := make(map[string]bool)
	for _, activity := range resp.Activities {
		foundCodes[activity.ProductCode.Code] = true
	}

	for _, code := range activityProductCodes {
		require.True(t, foundCodes[code], "expected product code %s not found in response", code)
	}
}

// Product list request with a modification filter set. It should only return one fitting result.
func testActivityV2ProductListServiceWithFilter(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
	// Modification timestamp which should exactly return one result (see expectedProductCode).
	// See the activityv2.json file in the pp-mock for more info
	const modifiedAfterSecs int64 = 1710237631
	const expectedProductCode = "TC000000"

	req := &activityv2.ActivityProductListRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		ModifiedAfter: &timestamppb.Timestamp{
			Seconds: modifiedAfterSecs,
		},
	}
	resp, err := distributorBot.ActivityProductListServiceV2.ActivityProductList(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)

	tt.logger.Debug("ActivityProductListServiceV2.ActivityProductList response:\n", protoMessageToJSON(tt, resp))

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain only one activity as only one is modified after the given timestamp
	require.Len(t, resp.Activities, 1, "unexpected number of activities in response")

	require.NotNil(t, resp.Activities[0].ProductCode, "unexpected nil response activities[0].ProductCode")
	require.Equal(t, expectedProductCode, resp.Activities[0].ProductCode.Code, "unexpected product code in response")

	// Verify the timestamp is correct
	require.NotNil(t, resp.Activities[0].LastModified, "activity has no last_modified timestamp")
	require.Greater(t, resp.Activities[0].LastModified.Seconds, modifiedAfterSecs,
		"activity timestamp is not after filter time")

	require.NotEmpty(t, resp.Activities[0].Context, "activity context should not be empty")
	require.NotEmpty(t, resp.Activities[0].ExternalSessionId, "activity external_session_id should not be empty")
	require.NotEmpty(t, resp.Activities[0].UnitCode, "activity unit_code should not be empty")
	require.NotEmpty(t, resp.Activities[0].ServiceCode, "activity service_code should not be empty")
	require.NotNil(t, resp.Activities[0].Bookability, "activity bookability should not be nil")
}

// Get detailed activity information for a specific product code.
func testActivityV2ProductInfoService(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
	req := &activityv2.ActivityProductInfoRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		// No filter to get all activities
	}

	allActivitiesResp, err := distributorBot.ActivityProductInfoServiceV2.ActivityProductInfo(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)

	tt.logger.Debug("ActivityProductInfoServiceV2.ActivityProductInfo response:\n", protoMessageToJSON(tt, allActivitiesResp))

	supplierCode2 := activityV2ProductCode

	req2 := &activityv2.ActivityProductInfoRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SupplierCodes: []*typesv2.SupplierProductCode{
			{SupplierCode: supplierCode2},
		},
	}
	resp, err := distributorBot.ActivityProductInfoServiceV2.ActivityProductInfo(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req2,
	)
	require.NoError(t, err)

	tt.logger.Debug("ActivityProductInfoServiceV2.ActivityProductInfo response:\n", protoMessageToJSON(tt, resp))

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
	require.Equal(t, supplierCode2, activity.SupplierCode.SupplierCode, "unexpected SupplierCode value")

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

func testActivityV2SearchServiceWithoutCurrency(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
	req := &activityv2.ActivitySearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Metadata: &typesv2.SearchRequestMetadata{
			RequestId: &typesv1.UUID{Value: uuid.New().String()},
		},
		SearchParametersGeneric: &typesv2.SearchParameters{},
	}

	resp, err := distributorBot.ActivitySearchServiceV2.ActivitySearch(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)

	require.NoError(t, err)

	tt.logger.Debug("ActivitySearchServiceV2.ActivitySearch response:\n", protoMessageToJSON(tt, resp))
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

func testActivityV2SearchServiceWithoutTravelPeriod(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
	req := &activityv2.ActivitySearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Metadata: &typesv2.SearchRequestMetadata{
			RequestId: &typesv1.UUID{Value: uuid.New().String()},
		},
		SearchParametersGeneric: &typesv2.SearchParameters{
			Currency: &typesv2.Currency{Currency: &typesv2.Currency_NativeToken{}},
		},
		SearchParametersActivity: &activityv2.ActivitySearchParameters{
			ProductCodes: []*typesv2.ProductCode{
				{
					Code: activityV2ProductCode,
				},
			},
		},
	}
	resp, err := distributorBot.ActivitySearchServiceV2.ActivitySearch(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	tt.logger.Debug("ActivitySearchServiceV2.ActivitySearch response:\n", protoMessageToJSON(tt, resp))
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
	require.NotEmpty(t, resp.Header.Alerts, "unexpected empty response alerts")
	require.Equal(t, 1, len(resp.Header.Alerts), "unexpected number of alerts in response")
	require.Equal(t, typesv1.AlertType_ALERT_TYPE_ERROR, resp.Header.Alerts[0].Type, "unexpected alert type")
}

func testActivityV2SearchServiceTravelPeriodOutOfBounds(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
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
			ProductCodes: []*typesv2.ProductCode{
				{
					Code: activityV2ProductCode,
				},
			},
		},
		TravelPeriod: &typesv1.TravelPeriod{
			StartDate: common.TimeToDateV1(startDate),
			EndDate:   common.TimeToDateV1(endDate),
		},
	}
	resp, err := distributorBot.ActivitySearchServiceV2.ActivitySearch(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)

	tt.logger.Debug("ActivitySearchServiceV2.ActivitySearch response:\n", protoMessageToJSON(tt, resp))
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

func testActivityV2SearchServiceTravelPeriodReversed(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
	const nights = 12                           // 12 nights
	startDate := time.Now().Add(time.Hour * 24) // tomorrow
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	resp, err := distributorBot.ActivitySearchServiceV2.ActivitySearch(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		&activityv2.ActivitySearchRequest{
			Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
			Metadata: &typesv2.SearchRequestMetadata{
				RequestId: &typesv1.UUID{Value: uuid.New().String()},
			},
			SearchParametersGeneric: &typesv2.SearchParameters{
				Currency: &typesv2.Currency{Currency: &typesv2.Currency_NativeToken{}},
			},
			SearchParametersActivity: &activityv2.ActivitySearchParameters{
				ProductCodes: []*typesv2.ProductCode{
					{
						Code: activityV2ProductCode,
					},
				},
			},
			TravelPeriod: &typesv1.TravelPeriod{
				StartDate: common.TimeToDateV1(endDate),   // End date used as start
				EndDate:   common.TimeToDateV1(startDate), // Start date used as end
			},
		},
	)
	require.NoError(t, err)

	tt.logger.Debug("ActivitySearchServiceV2.ActivitySearch response:\n", protoMessageToJSON(tt, resp))
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
	require.NotEmpty(t, resp.Header.Alerts, "unexpected empty response alerts")
}

// Test search with a valid travel period. Expect valid search results.
func testActivityV2SearchServiceWithTravelPeriod(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) (
	searchID string,
	resultID int32,
	totalPrice float64,
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
			ProductCodes: []*typesv2.ProductCode{
				{
					Code: activityV2ProductCode,
				},
			},
			ServiceCodes: []string{
				"XO",
			},
		},
		TravelPeriod: &typesv1.TravelPeriod{
			StartDate: common.TimeToDateV1(startDate),
			EndDate:   common.TimeToDateV1(endDate),
		},
		Travellers: []*typesv2.BasicTraveller{
			{
				TravellerId: 0,
				Type:        typesv2.TravellerType_TRAVELLER_TYPE_ADULT,
				Birthdate: &typesv1.Date{
					Year:  1990,
					Month: 1,
					Day:   1,
				},
				Nationality: typesv2.Country_COUNTRY_ES,
			},
		},
	}

	tt.logger.Debug("ActivitySearchServiceV2.ActivitySearch request:\n", protoMessageToJSON(tt, req))

	resp, err := distributorBot.ActivitySearchServiceV2.ActivitySearch(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)

	tt.logger.Debug("ActivitySearchServiceV2.ActivitySearch response:\n", protoMessageToJSON(tt, resp))

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
	require.NotEmpty(t, resp.Results[0].Price.Value, "unexpected empty Price.Value")

	totalPrice, err = strconv.ParseFloat(resp.Results[0].Price.Value, 64)
	require.NoError(t, err)

	// Now extract all the values needed for the validate step which comes next
	require.NotEmpty(t, resp.Metadata, "unexpected empty response Metadata")
	require.NotEmpty(t, resp.Metadata.SearchId, "unexpected empty response Metadata.SearchId")
	require.NotEmpty(t, resp.Metadata.SearchId.Value, "unexpected empty response Metadata.SearchId.Value")

	return resp.Metadata.SearchId.Value, resp.Results[0].ResultId, totalPrice
}

// Let's test the validation step with the values extracted from the search request
func testActivityV2ValidateV2(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	searchID string,
	resultID int32,
	expectedTotalPrice float64,
) (validateID string) {
	resp, err := distributorBot.ValidationServiceV2.Validation(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		&bookv2.ValidationRequest{
			ValidationObject: &bookv2.ValidationObject{
				SearchIdentifier: &typesv2.SearchIdentifier{
					SearchId: &typesv1.UUID{Value: searchID},
					ResultId: resultID,
				},
			},
		},
	)
	require.NoError(t, err)

	tt.logger.Debug("ValidationServiceV2.Validation response:\n", protoMessageToJSON(tt, resp))

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// Check if the validationObject is correct in the response
	require.NotEmpty(t, resp.ValidationObject, "unexpected empty response ValidationObject")
	require.NotEmpty(t, resp.ValidationObject.SearchIdentifier, "unexpected empty response ValidationObject.SearchIdentifier")
	require.NotEmpty(t, resp.ValidationObject.SearchIdentifier.SearchId, "unexpected empty response ValidationObject.SearchIdentifier.SearchId")
	require.NotEmpty(t, resp.ValidationObject.SearchIdentifier.SearchId.Value, "unexpected empty response ValidationObject.SearchIdentifier.SearchId.Value")
	require.Equal(t, searchID, resp.ValidationObject.SearchIdentifier.SearchId.Value, "unexpected searchID in response")
	require.Equal(t, resultID, resp.ValidationObject.SearchIdentifier.ResultId, "unexpected resultID in response")

	// Check if the price is as expected
	require.NotEmpty(t, resp.PriceDetail, "unexpected empty response PriceDetail")
	require.NotEmpty(t, resp.PriceDetail.Price, "unexpected empty response PriceDetail.Price")
	require.NotEmpty(t, resp.PriceDetail.Price.Value, "unexpected empty response PriceDetail.Price.Value")
	totalPriceResponse, err := strconv.ParseFloat(resp.PriceDetail.Price.Value, 64)
	require.NoError(t, err)
	require.Equal(t, expectedTotalPrice, totalPriceResponse, "unexpected total price in validation")

	// Last check if the validationID is set and if yes extract it and pass it back for the mint step
	require.NotEmpty(t, resp.ValidationId, "unexpected empty response validationID")
	require.NotEmpty(t, resp.ValidationId.Value, "unexpected empty response validationID.Value")
	return resp.ValidationId.Value
}

// Lastly we do the mint request based on the validation id
func testActivityV2MintV2(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	validationID string,
) (
	tokenID uint64,
	price *typesv2.Price,
) {
	resp, err := distributorBot.MintServiceV2.Mint(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		&bookv2.MintRequest{
			Header:       &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
			ValidationId: &typesv1.UUID{Value: validationID},
		},
	)
	require.NoError(t, err)

	tt.logger.Debug("MintServiceV2.Mint response:\n", protoMessageToJSON(tt, resp))

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")

	// Check if the MintId is set
	require.NotEmpty(t, resp.MintId, "unexpected empty response MintId")
	require.NotEmpty(t, resp.MintId.Value, "unexpected empty response MintId.Value")

	// check if the transaction ids are set and return them for further tests
	require.NotEmpty(t, resp.MintTransactionId, "unexpected empty response MintTransactionId")
	require.NotEmpty(t, resp.BuyTransactionId, "unexpected empty response BuyTransactionId")

	return resp.BookingTokenId, resp.Price
}

func testActivityV2VerifyBlockchainState(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	tokenID uint64,
	tokenPrice *typesv2.Price,
) {
	bigTokenID := big.NewInt(0).SetUint64(tokenID)
	callOpts := &bind.CallOpts{Context: ctx}

	require.Equal(t, booking.NativePaymentToken, getPaymentTokenFromPriceV2(t, tokenPrice))
	expectedReservationPrice, err := price.ToBigInt(tokenPrice.Value, tokenPrice.Decimals, price.NativeTokenDecimals)
	require.NoError(t, err)

	reservationPrice, err := tt.caminoNetwork.Client.BookingToken.GetReservationPrice(callOpts, bigTokenID)
	require.NoError(t, err)
	require.Equal(t, booking.NativePaymentToken, reservationPrice.PaymentToken)
	require.Equal(t, expectedReservationPrice, reservationPrice.Price)

	ownerAddr, err := tt.caminoNetwork.Client.BookingToken.OwnerOf(callOpts, bigTokenID)
	require.NoError(t, err)
	require.Equal(t, distributorBot.CMAccountAddress(), ownerAddr)

	tokenStatus, err := tt.caminoNetwork.Client.BookingToken.GetBookingStatus(callOpts, bigTokenID)
	require.NoError(t, err)
	require.Equal(t, booking.StatusBought, booking.Status(tokenStatus))
}

func TestActivityV2(t *testing.T, tt *Test) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()
	_, supplierBot, distributorBot := testActivityV2Setup(ctx, t, tt)

	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the activities
		testActivityV2ProductListService(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Product list with filter", func(t *testing.T) {
		// Happy path: will return only one activity
		testActivityV2ProductListServiceWithFilter(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Product info", func(t *testing.T) {
		// Happy path: will return only one activity
		testActivityV2ProductInfoService(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Product Search without currency", func(t *testing.T) {
		testActivityV2SearchServiceWithoutCurrency(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Product Search without travel period", func(t *testing.T) {
		testActivityV2SearchServiceWithoutTravelPeriod(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Product Search travel period out of bounds", func(t *testing.T) {
		testActivityV2SearchServiceTravelPeriodOutOfBounds(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Product Search travel period reversed", func(t *testing.T) {
		testActivityV2SearchServiceTravelPeriodReversed(ctx, t, tt, distributorBot, supplierBot)
	})

	t.Run("Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		searchID, resultID, totalPrice := testActivityV2SearchServiceWithTravelPeriod(ctx, t, tt, distributorBot, supplierBot)
		validationID := testActivityV2ValidateV2(ctx, t, tt, distributorBot, supplierBot, searchID, resultID, totalPrice)
		tokenID, price := testActivityV2MintV2(ctx, t, tt, distributorBot, supplierBot, validationID)
		testActivityV2VerifyBlockchainState(ctx, t, tt, distributorBot, tokenID, price)
	})
}

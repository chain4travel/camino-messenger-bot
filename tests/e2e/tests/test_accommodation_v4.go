// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/conversion"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/suite"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestAccommodationV4)(nil)

func init() {
	Tests["AccommodationV4"] = &TestAccommodationV4{}
}

type TestAccommodationV4 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestAccommodationV4) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestAccommodationV4) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("Product short list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testAccommodationV4ProductShortListService(ctx, t)
	})
	t.Run("Product short list with filter", func(t *testing.T) {
		// Happy path: will return only one property
		tt.testAccommodationV4ProductShortListServiceWithFilter(ctx, t)
	})
	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testAccommodationV4ProductListService(ctx, t)
	})
	t.Run("Product info", func(t *testing.T) {
		// Happy path: will return the detailed info of a property
		tt.testAccommodationV4ProductInfoService(ctx, t)
	})
	t.Run("Search with travel period oob", func(t *testing.T) {
		// ERROR path: with travel period outside of allowed constraints it should return an error
		tt.testAccommodationV4SearchServiceTravelPeriodOutOfBounds(ctx, t)
	})
	t.Run("Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		searchID, resultID, totalPrice := testAccommodationV4SearchService(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)
		validationID := testValidateV4(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		balanceBefore := tt.Environment.Balance(ctx, t, tt.distributorBot)
		tokenID, _, mintRespPrice := testMintV4(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID, common.BookingTokenPriceV4)
		verifyBookingTokenStateBoughtWithPriceV4(ctx, t, tt.Environment, tt.distributorBot, tokenID, mintRespPrice, balanceBefore)
	})
}

func (tt *TestAccommodationV4) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationProductShortListServiceV4,
		botGenerated.AccommodationProductListServiceV4,
		botGenerated.AccommodationProductInfoServiceV4,
		botGenerated.AccommodationSearchServiceV4,
		botGenerated.ValidationServiceV4,
		botGenerated.MintServiceV4,
	))

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, false, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.AccommodationProductShortListServiceV4, Fee: 100},
			{Name: botGenerated.AccommodationProductListServiceV4, Fee: 110},
			{Name: botGenerated.AccommodationProductInfoServiceV4, Fee: 120},
			{Name: botGenerated.AccommodationSearchServiceV4, Fee: 130},
			{Name: botGenerated.ValidationServiceV4, Fee: 140},
			{Name: botGenerated.MintServiceV4, Fee: 150},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

// Simple product list request which shall return all properties. Checking if all are present
func (tt *TestAccommodationV4) testAccommodationV4ProductShortListService(ctx context.Context, t *testing.T) {
	req := &accommodationv4.AccommodationProductShortListRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
	}
	resp, err := tt.distributorBot.AccommodationProductShortListServiceV4.AccommodationProductShortList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	require.Len(t, resp.PropertyShortListItems, len(mockdata.PropertiesV4), "unexpected number of properties in response")

	expected := make([]*accommodationv4.PropertyShortListItem, 0, len(mockdata.PropertiesV4))
	for _, prop := range mockdata.PropertiesV4 {
		expected = append(expected, &accommodationv4.PropertyShortListItem{
			SupplierCode: prop.Property.SupplierCode,
			Status:       prop.Property.Status,
		})
	}

	requireProtoSlicesElementsMatch(t, expected, resp.PropertyShortListItems)
}

// Product list request with a modification filter set. It should only return one fitting result.
func (tt *TestAccommodationV4) testAccommodationV4ProductShortListServiceWithFilter(ctx context.Context, t *testing.T) {
	modifiedAfter := time.Unix(1710547200, 0)
	var expected []*accommodationv4.PropertyShortListItem
	for _, prop := range mockdata.PropertiesV4 {
		if prop.Property.LastModified.AsTime().After(modifiedAfter) {
			expected = append(expected, &accommodationv4.PropertyShortListItem{
				SupplierCode: prop.Property.SupplierCode,
				Status:       prop.Property.Status,
			})
		}
	}
	require.Len(t, expected, 1, "test setup error: expected exactly one property to match the modifiedAfter filter")

	req := &accommodationv4.AccommodationProductShortListRequest{
		Header:        &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		ModifiedAfter: timestamppb.New(modifiedAfter),
	}
	resp, err := tt.distributorBot.AccommodationProductShortListServiceV4.AccommodationProductShortList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	require.Len(t, resp.PropertyShortListItems, len(expected), "unexpected number of properties in response")

	requireProtoSlicesElementsMatch(t, expected, resp.PropertyShortListItems)
}

// Product list request with a filter set. It should only return one fitting result.
func (tt *TestAccommodationV4) testAccommodationV4ProductListService(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL567890"

	req := &accommodationv4.AccommodationProductListRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SupplierCodes: []*typesv4.SupplierProductCode{
			{Code: hotelCode},
		},
	}
	resp, err := tt.distributorBot.AccommodationProductListServiceV4.AccommodationProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	require.Len(t, resp.Properties, 1, "unexpected number of properties in response")

	require.Equal(t, hotelCode, resp.Properties[0].SupplierCode.Code, "unexpected response properties[0].SupplierCode.Code")
}

// Get detailed accommodation information for a specific hotel code (supplier code).
func (tt *TestAccommodationV4) testAccommodationV4ProductInfoService(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL789012"
	const language = typesv1.Language_LANGUAGE_EN

	var expectedProperty *accommodationv4.PropertyExtendedInfo
	for _, prop := range mockdata.PropertiesV4 {
		if prop.Property.SupplierCode.Code == hotelCode {
			expectedProperty = prop
			break
		}
	}
	require.NotNil(t, expectedProperty, "test setup error: no property with supplier code %s found in mock data", hotelCode)

	req := &accommodationv4.AccommodationProductInfoRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SupplierCodes: []*typesv4.SupplierProductCode{
			{Code: hotelCode},
		},
		Languages: []typesv1.Language{language},
	}
	resp, err := tt.distributorBot.AccommodationProductInfoServiceV4.AccommodationProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	require.Len(t, resp.Properties, 1, "unexpected number of properties in response")

	require.Len(t, resp.Properties[0].LocalizedDescriptions, 1, "unexpected number of localized descriptions in response")
	require.Equal(t, language, resp.Properties[0].LocalizedDescriptions[0].Language, "unexpected language in section name")
	resp.Properties[0].LocalizedDescriptions = nil

	expectedProperty = common.CloneProto(expectedProperty)
	expectedProperty.LocalizedDescriptions = nil

	require.True(t, proto.Equal(resp.Properties[0], expectedProperty), "unexpected response property")
}

// Test search with wrong travel periods given: travel period outside of allowed constraints. Expect errors to be returned.
func (tt *TestAccommodationV4) testAccommodationV4SearchServiceTravelPeriodOutOfBounds(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL345678"

	const nights = 12                                 // 12 nights
	startDate := time.Now().Add(time.Hour * 24 * 100) // in 100 days, outside of allowed travel period
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &accommodationv4.AccommodationSearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{Currency: &typesv4.Currency_NativeToken{}},
			Language: typesv1.Language_LANGUAGE_EN,
		},
		Queries: []*accommodationv4.AccommodationSearchQuery{{
			SearchParametersAccommodation: &accommodationv4.AccommodationSearchParameters{
				SupplierCodes: []*typesv4.SupplierProductCode{
					{Code: hotelCode},
				},
			},
			TravelPeriod: &typesv4.TravelPeriod{
				StartDate: common.TimeToDateV4(startDate),
				EndDate:   common.TimeToDateV4(endDate),
			},
			Travellers:   []*typesv4.BasicTraveller{{Type: typesv4.TravellerType_TRAVELLER_TYPE_ADULT}},
			PropertyType: accommodationv4.PropertyType_PROPERTY_TYPE_HOTEL,
		}},
	}
	resp, err := tt.distributorBot.AccommodationSearchServiceV4.AccommodationSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv4.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
	require.NotEmpty(t, resp.Header.Alerts, "expected response alerts but got none")
}

// Test search with a valid travel period. Expect valid search results.
func testAccommodationV4SearchService(
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
	const hotelCode1 = "HOTEL345678"
	const hotelCode2 = "HOTEL789012"
	const nights = 12                           // 12 nights
	startDate := time.Now().Add(time.Hour * 24) // tomorrow
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))
	currency := &typesv4.Currency{Currency: &typesv4.Currency_NativeToken{}}

	req := &accommodationv4.AccommodationSearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: currency,
			Language: typesv1.Language_LANGUAGE_EN,
		},
		Queries: []*accommodationv4.AccommodationSearchQuery{{
			SearchParametersAccommodation: &accommodationv4.AccommodationSearchParameters{
				SupplierCodes: []*typesv4.SupplierProductCode{
					{Code: hotelCode1},
					{Code: hotelCode2},
				},
			},
			TravelPeriod: &typesv4.TravelPeriod{
				StartDate: common.TimeToDateV4(startDate),
				EndDate:   common.TimeToDateV4(endDate),
			},
			Travellers:   []*typesv4.BasicTraveller{{Type: typesv4.TravellerType_TRAVELLER_TYPE_ADULT}},
			PropertyType: accommodationv4.PropertyType_PROPERTY_TYPE_HOTEL,
		}},
	}
	resp, err := distributorBot.AccommodationSearchServiceV4.AccommodationSearch(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	require.Len(t, resp.Results, 2, "unexpected number of results in response")

	if resp.Results[1].Units[0].SupplierCode.Code == hotelCode1 {
		require.Equal(t, hotelCode2, resp.Results[0].Units[0].SupplierCode.Code)
	} else {
		require.Equal(t, hotelCode1, resp.Results[0].Units[0].SupplierCode.Code)
		require.Equal(t, hotelCode2, resp.Results[1].Units[0].SupplierCode.Code)
	}

	for i, result := range resp.Results {
		require.Equal(t, conversion.MustIntToUInt32(i), result.ResultId, "unexpected response Results[%d].ResultId", i)
	}

	// We expect 2 results - let's check for the 2nd one

	require.Len(t, resp.Results[1].Units, 1, "unexpected empty response Results[1].Units")

	expectedPrice := &typesv4.Price{
		Value:    fmt.Sprintf("%d", common.DefaultPricePerNight*nights),
		Decimals: 0,
		Currency: currency,
	}
	require.True(t, proto.Equal(expectedPrice, resp.Results[1].Units[0].PriceDetail.Price), "unexpected response Results[1].Units[0].PriceDetail.Price: got %+v, want %+v", resp.Results[1].Units[0].PriceDetail.Price, expectedPrice)

	// just one unit, total price is the same as unit price
	require.True(t, proto.Equal(expectedPrice, resp.Results[1].TotalPrice.Value), "unexpected response Results[1].TotalPrice.Value: got %+v, want %+v", resp.Results[1].TotalPrice.Value, expectedPrice)

	return resp.SearchId.Id.Value, resp.Results[1].ResultId, expectedPrice
}

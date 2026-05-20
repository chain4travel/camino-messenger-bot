// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	accommodationv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v5"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	typesv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v5"
	"buf.build/go/protovalidate"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v13/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/conversion"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/services/data"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/suite"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestAccommodationV5)(nil)

func init() {
	Tests["AccommodationV5"] = &TestAccommodationV5{}
}

type TestAccommodationV5 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestAccommodationV5) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestAccommodationV5) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("Product short list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testAccommodationV5ProductShortListService(ctx, t)
	})
	t.Run("Product short list with filter", func(t *testing.T) {
		// Happy path: will return only one property
		tt.testAccommodationV5ProductShortListServiceWithFilter(ctx, t)
	})
	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testAccommodationV5ProductListService(ctx, t)
	})
	t.Run("Product info", func(t *testing.T) {
		// Happy path: will return the detailed info of a property
		tt.testAccommodationV5ProductInfoService(ctx, t)
	})
	t.Run("Search with travel period oob", func(t *testing.T) {
		// ERROR path: with travel period outside of allowed constraints it should return an error
		tt.testAccommodationV5SearchServiceTravelPeriodOutOfBounds(ctx, t)
	})
	t.Run("Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		searchID, resultID, totalPrice := testAccommodationV5SearchService(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)
		validationID := testValidateV5(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		balanceBefore := tt.Balance(ctx, t, tt.distributorBot)
		tokenID, _, mintRespPrice := testMintV5(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID, common.BookingTokenPriceV5)
		verifyBookingTokenStateBoughtWithPriceV5(ctx, t, tt.Environment, tt.distributorBot, tokenID, mintRespPrice, balanceBefore)
	})
}

func (tt *TestAccommodationV5) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationProductShortListServiceV5,
		botGenerated.AccommodationProductListServiceV5,
		botGenerated.AccommodationProductInfoServiceV5,
		botGenerated.AccommodationSearchServiceV5,
		botGenerated.ValidationServiceV5,
		botGenerated.MintServiceV5,
	))

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, false, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.AccommodationProductShortListServiceV5, Fee: 100},
			{Name: botGenerated.AccommodationProductListServiceV5, Fee: 110},
			{Name: botGenerated.AccommodationProductInfoServiceV5, Fee: 120},
			{Name: botGenerated.AccommodationSearchServiceV5, Fee: 130},
			{Name: botGenerated.ValidationServiceV5, Fee: 140},
			{Name: botGenerated.MintServiceV5, Fee: 150},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

// Simple product list request which shall return all properties. Checking if all are present
func (tt *TestAccommodationV5) testAccommodationV5ProductShortListService(ctx context.Context, t *testing.T) {
	req := &accommodationv5.AccommodationProductShortListRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
	}
	resp, err := tt.distributorBot.AccommodationProductShortListServiceV5.AccommodationProductShortList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")
	require.Len(t, successResp.PropertyShortListItems, len(mockdata.PropertiesV5), "unexpected number of properties in response")

	expected := make([]*accommodationv5.PropertyShortListItem, 0, len(mockdata.PropertiesV5))
	for _, prop := range mockdata.PropertiesV5 {
		expected = append(expected, &accommodationv5.PropertyShortListItem{
			SupplierCode: prop.Property.SupplierCode,
			Status:       prop.Property.Status,
		})
	}

	requireProtoSlicesElementsMatch(t, expected, successResp.PropertyShortListItems)
}

// Product list request with a modification filter set. It should only return one fitting result.
func (tt *TestAccommodationV5) testAccommodationV5ProductShortListServiceWithFilter(ctx context.Context, t *testing.T) {
	modifiedAfter := time.Unix(1710547200, 0)
	var expected []*accommodationv5.PropertyShortListItem
	for _, prop := range mockdata.PropertiesV5 {
		if prop.Property.LastModified.AsTime().After(modifiedAfter) {
			expected = append(expected, &accommodationv5.PropertyShortListItem{
				SupplierCode: prop.Property.SupplierCode,
				Status:       prop.Property.Status,
			})
		}
	}
	require.Len(t, expected, 1, "test setup error: expected exactly one property to match the modifiedAfter filter")

	req := &accommodationv5.AccommodationProductShortListRequest{
		Header:        &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		ModifiedAfter: timestamppb.New(modifiedAfter),
	}
	resp, err := tt.distributorBot.AccommodationProductShortListServiceV5.AccommodationProductShortList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")

	require.Len(t, successResp.PropertyShortListItems, len(expected), "unexpected number of properties in response")

	requireProtoSlicesElementsMatch(t, expected, successResp.PropertyShortListItems)
}

// Product list request with a filter set. It should only return one fitting result.
func (tt *TestAccommodationV5) testAccommodationV5ProductListService(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL567890"

	req := &accommodationv5.AccommodationProductListRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SupplierCodes: []*typesv4.SupplierProductCode{
			{Code: hotelCode},
		},
	}
	resp, err := tt.distributorBot.AccommodationProductListServiceV5.AccommodationProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")

	require.Len(t, successResp.Properties, 1, "unexpected number of properties in response")

	require.Equal(t, hotelCode, successResp.Properties[0].SupplierCode.Code, "unexpected response properties[0].SupplierCode.Code")
}

// Get detailed accommodation information for a specific hotel code (supplier code).
func (tt *TestAccommodationV5) testAccommodationV5ProductInfoService(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL789012"
	const language = typesv1.Language_LANGUAGE_EN

	var expectedProperty *accommodationv5.PropertyExtendedInfo
	for _, prop := range mockdata.PropertiesV5 {
		if prop.Property.SupplierCode.Code == hotelCode {
			expectedProperty = prop
			break
		}
	}
	require.NotNil(t, expectedProperty, "test setup error: no property with supplier code %s found in mock data", hotelCode)

	req := &accommodationv5.AccommodationProductInfoRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SupplierCodes: []*typesv4.SupplierProductCode{
			{Code: hotelCode},
		},
		Languages: []typesv1.Language{language},
	}
	resp, err := tt.distributorBot.AccommodationProductInfoServiceV5.AccommodationProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")

	require.Len(t, successResp.Properties, 1, "unexpected number of properties in response")

	require.Len(t, successResp.Properties[0].LocalizedDescriptions, 1, "unexpected number of localized descriptions in response")
	require.Equal(t, language, successResp.Properties[0].LocalizedDescriptions[0].Language, "unexpected language in section name")
	successResp.Properties[0].LocalizedDescriptions = nil

	expectedProperty = common.CloneProto(expectedProperty)
	expectedProperty.LocalizedDescriptions = nil

	require.True(t, proto.Equal(successResp.Properties[0], expectedProperty), "unexpected response property")
}

// Test search with wrong travel periods given: travel period outside of allowed constraints. Expect errors to be returned.
func (tt *TestAccommodationV5) testAccommodationV5SearchServiceTravelPeriodOutOfBounds(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL345678"

	const nights = 12                                                                                              // 12 nights
	startDate := time.Now().Add(common.TravelPeriodMinStartOffset + common.TravelPeriodMaxDuration + 24*time.Hour) // outside of allowed travel period
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &accommodationv5.AccommodationSearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{Currency: &typesv4.Currency_NativeToken{}},
			Language: typesv1.Language_LANGUAGE_EN,
		},
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
		PropertyType: accommodationv5.PropertyType_PROPERTY_TYPE_HOTEL,
	}
	resp, err := tt.distributorBot.AccommodationSearchServiceV5.AccommodationSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	require.True(t, resp.HasErrorResponse(), "unexpected response status")
}

// Test search with a valid travel period. Expect valid search results.
func testAccommodationV5SearchService(
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
	const hotelCode1 = "HOTEL345678"
	const hotelCode2 = "HOTEL789012"
	const nights = 12                                              // 12 nights
	startDate := time.Now().Add(common.TravelPeriodMinStartOffset) // tomorrow
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))
	currency := &typesv4.Currency{Currency: &typesv4.Currency_NativeToken{}}

	req := &accommodationv5.AccommodationSearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: currency,
			Language: typesv1.Language_LANGUAGE_EN,
		},
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
		PropertyType: accommodationv5.PropertyType_PROPERTY_TYPE_HOTEL,
	}
	resp, err := distributorBot.AccommodationSearchServiceV5.AccommodationSearch(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")

	require.Len(t, successResp.Results, 2, "unexpected number of results in response")

	if successResp.Results[1].Unit.SupplierCode.Code == hotelCode1 {
		require.Equal(t, hotelCode2, successResp.Results[0].Unit.SupplierCode.Code)
	} else {
		require.Equal(t, hotelCode1, successResp.Results[0].Unit.SupplierCode.Code)
		require.Equal(t, hotelCode2, successResp.Results[1].Unit.SupplierCode.Code)
	}

	for i, result := range successResp.Results {
		require.Equal(t, conversion.MustIntToUInt32(i), result.ResultId, "unexpected response Results[%d].ResultId", i)
	}

	// We expect 2 results - let's check for the 2nd one

	expectedPrice := &typesv5.Price{
		Value:    fmt.Sprintf("%d", common.DefaultPricePerNight*nights),
		Decimals: 0,
		Currency: currency,
	}
	require.True(t, proto.Equal(expectedPrice, successResp.Results[1].Unit.PriceDetail.Price), "unexpected response Results[1].Unit.PriceDetail.Price: got %+v, want %+v", successResp.Results[1].Unit.PriceDetail.Price, expectedPrice)

	// just one unit, total price is the same as unit price
	require.True(t, proto.Equal(expectedPrice, successResp.Results[1].TotalPrice.Value), "unexpected response Results[1].TotalPrice.Value: got %+v, want %+v", successResp.Results[1].TotalPrice.Value, expectedPrice)

	return successResp.SearchId.Id.Value, successResp.Results[1].ResultId, expectedPrice
}

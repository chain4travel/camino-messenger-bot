// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/suite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

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

	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testAccommodationV4ProductListService(ctx, t)
	})
	t.Run("Product list with filter", func(t *testing.T) {
		// Happy path: will return only one property
		tt.testAccommodationV4ProductListServiceWithFilter(ctx, t)
	})
	t.Run("Product info", func(t *testing.T) {
		// Happy path: will return the detailed info of a property
		tt.testAccommodationV4ProductInfoService(ctx, t)
	})
	t.Run("Search w/o travel period", func(t *testing.T) {
		// ERROR path: without travel period it should return an error
		tt.testAccommodationV4SearchServiceWithoutTravelPeriod(ctx, t)
	})
	t.Run("Search with travel period oob", func(t *testing.T) {
		// ERROR path: with travel period outside of allowed constraints it should return an error
		tt.testAccommodationV4SearchServiceTravelPeriodOutOfBounds(ctx, t)
	})
	t.Run("Search with travel period reversed", func(t *testing.T) {
		// ERROR path: with travel period reversed it should return an error
		tt.testAccommodationV4SearchServiceTravelPeriodReversed(ctx, t)
	})
	t.Run("Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		searchID, resultID, totalPrice := testAccommodationV4SearchServiceWithTravelPeriod(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)
		validationID := testValidateV4(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		tokenID, price, _ := testMintV4(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID)
		verifyBookingTokenStateWithPriceV4(ctx, t, tt.Environment, tt.distributorBot, tokenID, price)
	})
}

func (tt *TestAccommodationV4) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationProductListServiceV4,
		botGenerated.AccommodationProductInfoServiceV4,
		botGenerated.AccommodationSearchServiceV4,
		botGenerated.ValidationServiceV4,
		botGenerated.MintServiceV4,
	))

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.AccommodationProductListServiceV4, Fee: 100},
			{Name: botGenerated.AccommodationProductInfoServiceV4, Fee: 110},
			{Name: botGenerated.AccommodationSearchServiceV4, Fee: 120},
			{Name: botGenerated.ValidationServiceV4, Fee: 130},
			{Name: botGenerated.MintServiceV4, Fee: 140},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

// Simple product list request which shall return all properties. Checking if all are present
func (tt *TestAccommodationV4) testAccommodationV4ProductListService(ctx context.Context, t *testing.T) {
	hotelCodes := []string{
		"HOTEL123456",
		"HOTEL789012",
		"HOTEL345678",
		"HOTEL901234",
		"HOTEL567890",
	}

	req := &accommodationv4.AccommodationProductListRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
	}
	resp, err := tt.distributorBot.AccommodationProductListServiceV4.AccommodationProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain all properties defined by the pp-mock (defined by hotelCodes/expectedTotalResults)
	require.Len(t, resp.Properties, len(mockdata.PropertiesV4), "unexpected number of properties in response")

	for i := range hotelCodes {
		require.NotEmpty(t, resp.Properties[i].SupplierCode, "unexpected empty response properties[%d].SupplierCode", i)
		require.NotEmpty(t, resp.Properties[i].SupplierCode.Code, "unexpected empty response properties[%d].SupplierCode.Code", i)
		require.Contains(t, hotelCodes, resp.Properties[i].SupplierCode.Code, "unexpected response properties[%d].SupplierCode.Code", i)
	}
}

// Product list request with a modification filter set. It should only return one fitting result.
func (tt *TestAccommodationV4) testAccommodationV4ProductListServiceWithFilter(ctx context.Context, t *testing.T) {
	// Modification timestamp which should exactly return one result (see hotelCode).
	// See the properties.json file in the pp-mock for more info
	const modifiedAfterSecs int64 = 1710489050
	const hotelCode = "HOTEL567890"

	req := &accommodationv4.AccommodationProductListRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		ModifiedAfter: &timestamppb.Timestamp{
			Seconds: modifiedAfterSecs,
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

	// The response should contain only one property as only one is modified after the given timestamp
	require.Len(t, resp.Properties, 1, "unexpected number of properties in response")

	require.NotEmpty(t, resp.Properties[0].SupplierCode, "unexpected empty response properties[0].SupplierCode")
	require.NotEmpty(t, resp.Properties[0].SupplierCode.Code, "unexpected empty response properties[0].SupplierCode.Code")
	require.Equal(t, hotelCode, resp.Properties[0].SupplierCode.Code, "unexpected response properties[0].SupplierCode.Code")
}

// Get detailed accommodation information for a specific hotel code (supplier code).
func (tt *TestAccommodationV4) testAccommodationV4ProductInfoService(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL789012"

	req := &accommodationv4.AccommodationProductInfoRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SupplierCodes: []*typesv4.SupplierProductCode{
			{Code: hotelCode},
		},
		Languages: []typesv1.Language{typesv1.Language_LANGUAGE_EN},
	}
	resp, err := tt.distributorBot.AccommodationProductInfoServiceV4.AccommodationProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain only the one property filtered in the request
	require.Len(t, resp.Properties, 1, "unexpected number of properties in response")

	require.NotEmpty(t, resp.Properties[0].Property, "unexpected empty response properties[0].Property")
	require.NotEmpty(t, resp.Properties[0].Property.SupplierCode, "unexpected empty response properties[0].SupplierCode")
	require.NotEmpty(t, resp.Properties[0].Property.SupplierCode.Code, "unexpected empty response properties[0].SupplierCode.Code")
	require.Equal(t, hotelCode, resp.Properties[0].Property.SupplierCode.Code, "unexpected response properties[0].SupplierCode.Code")

	// Let's also check for some other properties of the response
	require.NotEmpty(t, resp.Properties[0].Images, "unexpected empty response properties[0].Images")
	require.Len(t, resp.Properties[0].Images, 1, "unexpected number of images in response")
	require.Equal(t, resp.Properties[0].Images[0].File.Name, "Beach House", "unexpected image name")

	require.NotEmpty(t, resp.Properties[0].Videos, "unexpected empty response properties[0].Videos")
	require.Len(t, resp.Properties[0].Videos, 1, "unexpected number of videos in response")
	require.Equal(t, resp.Properties[0].Videos[0].File.Uri, "https://example.com/videos/resort-tour.mp4", "unexpected video url")

	require.NotEmpty(t, resp.Properties[0].Rooms, "unexpected empty response properties[0].Rooms")
	require.Len(t, resp.Properties[0].Rooms, 1, "unexpected number of rooms in response")
	require.Equal(t, resp.Properties[0].Rooms[0].SupplierCode, "DBL-MTN", "unexpected room code")
	require.Equal(t, resp.Properties[0].Rooms[0].TotalOccupancy.MinGuests, uint32(1), "unexpected min guests")
	require.Equal(t, resp.Properties[0].Rooms[0].TotalOccupancy.MaxGuests, uint32(3), "unexpected max guests")
	require.Equal(t, resp.Properties[0].Rooms[0].TotalOccupancy.StandardOccupancy, uint32(2), "unexpected standard occupancy")
}

// - metadata: value is required [required]
// - search_parameters: value is required [required]
// - queries[0].travel_period: value is required [required]
// - queries[0].travellers: value must contain at least 1 item(s) [repeated.min_items]
// - queries[0].unit_type: value must not be in list [0] [enum.not_in]

// Test search with wrong travel periods given: travel period outside of allowed constraints. Expect errors to be returned.
func (tt *TestAccommodationV4) testAccommodationV4SearchServiceTravelPeriodOutOfBounds(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL345678"

	const nights = 12                                 // 12 nights
	startDate := time.Now().Add(time.Hour * 24 * 100) // in 100 days, outside of allowed travel period
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &accommodationv4.AccommodationSearchRequest{
		Header:   &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		Metadata: &typesv4.SearchRequestMetadata{RequestId: &typesv4.UUID{Value: uuid.NewString()}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{Currency: &typesv4.Currency_NativeToken{}},
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
		}},
	}
	resp, err := tt.distributorBot.AccommodationSearchServiceV4.AccommodationSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv4.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search with wrong travel periods given: start date after end date. Expect errors to be returned.
func (tt *TestAccommodationV4) testAccommodationV4SearchServiceTravelPeriodReversed(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL345678"

	const nights = 12                                                // 12 nights
	endDate := time.Now().Add(time.Hour * 24)                        // tomorrow
	startDate := endDate.Add(time.Hour * 24 * time.Duration(nights)) // start date after end date

	req := &accommodationv4.AccommodationSearchRequest{
		Header:   &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		Metadata: &typesv4.SearchRequestMetadata{RequestId: &typesv4.UUID{Value: uuid.NewString()}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{Currency: &typesv4.Currency_NativeToken{}},
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
		}},
	}
	resp, err := tt.distributorBot.AccommodationSearchServiceV4.AccommodationSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv4.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search with a valid travel period. Expect valid search results.
func testAccommodationV4SearchServiceWithTravelPeriod(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) (
	searchID string,
	resultID uint32,
	totalPrice *big.Int,
) {
	const nights = 12                           // 12 nights
	startDate := time.Now().Add(time.Hour * 24) // tomorrow
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &accommodationv4.AccommodationSearchRequest{
		Header:   &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		Metadata: &typesv4.SearchRequestMetadata{RequestId: &typesv4.UUID{Value: uuid.NewString()}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{Currency: &typesv4.Currency_NativeToken{}},
		},
		Queries: []*accommodationv4.AccommodationSearchQuery{{
			SearchParametersAccommodation: &accommodationv4.AccommodationSearchParameters{
				SupplierCodes: []*typesv4.SupplierProductCode{
					{Code: "HOTEL345678"},
					{Code: "HOTEL789012"},
				},
			},
			TravelPeriod: &typesv4.TravelPeriod{
				StartDate: common.TimeToDateV4(startDate),
				EndDate:   common.TimeToDateV4(endDate),
			},
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

	// We expect 2 results - let's check for the 2nd one
	require.Len(t, resp.Results, 2, "unexpected number of results in response")

	// Let's check if result is as expected
	require.NotEmpty(t, resp.Results[1].Units, "unexpected empty response Results[1].Units")
	require.Equal(t, resp.Results[1].Units[0].SupplierCode.Code, "HOTEL345678", "unexpected response Results[1].Units[0].SupplierCode.Code")

	// Check if the price per night is set correctly
	pricePerNight := priceBigV4(t, resp.Results[1].Units[0].PriceDetail.Price)
	require.True(t, pricePerNight.Cmp(common.DefaultPricePerNightNativeTokenBig) == 0, "unexpected price per night: got %s, expected %s", pricePerNight.String(), common.DefaultPricePerNightNativeTokenBig.String())

	// Extract the total price from the response
	totalPrice = priceBigV4(t, resp.Results[1].TotalPrice.Value)

	// Check if this adds up with the total price of the unit
	expectedTotalPrice := big.NewInt(0).Mul(common.DefaultPricePerNightNativeTokenBig, big.NewInt(nights))
	require.True(t, totalPrice.Cmp(expectedTotalPrice) == 0, "unexpected total price: got %s, expected %s", totalPrice.String(), expectedTotalPrice.String())

	// Now extract all the values needed for the validate step which comes next
	require.NotEmpty(t, resp.Metadata, "unexpected empty response Metadata")
	require.NotEmpty(t, resp.Metadata.SearchId, "unexpected empty response Metadata.SearchId")
	require.NotEmpty(t, resp.Metadata.SearchId.Value, "unexpected empty response Metadata.SearchId.Value")

	require.NotEmpty(t, resp.Results[1].ResultId, "unexpected empty response Results[1].ResultId")

	return resp.Metadata.SearchId.Value, resp.Results[1].ResultId, totalPrice
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/suite"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestAccommodationV2)(nil)

func init() {
	Tests["AccommodationV2"] = &TestAccommodationV2{}
}

type TestAccommodationV2 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestAccommodationV2) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestAccommodationV2) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testAccommodationV2ProductListService(ctx, t)
	})
	t.Run("Product list with filter", func(t *testing.T) {
		// Happy path: will return only one property
		tt.testAccommodationV2ProductListServiceWithFilter(ctx, t)
	})
	t.Run("Product info", func(t *testing.T) {
		// Happy path: will return the detailed info of a property
		tt.testAccommodationV2ProductInfoService(ctx, t)
	})
	t.Run("Search w/o currency", func(t *testing.T) {
		// ERROR path: without currency it should return an error
		tt.testAccommodationV2SearchServiceWithoutCurrency(ctx, t)
	})
	t.Run("Search w/o travel period", func(t *testing.T) {
		// ERROR path: without travel period it should return an error
		tt.testAccommodationV2SearchServiceWithoutTravelPeriod(ctx, t)
	})
	t.Run("Search with travel period oob", func(t *testing.T) {
		// ERROR path: with travel period outside of allowed constraints it should return an error
		tt.testAccommodationV2SearchServiceTravelPeriodOutOfBounds(ctx, t)
	})
	t.Run("Search with travel period reversed", func(t *testing.T) {
		// ERROR path: with travel period reversed it should return an error
		tt.testAccommodationV2SearchServiceTravelPeriodReversed(ctx, t)
	})
	t.Run("Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		searchID, resultID, totalPrice := testAccommodationV2SearchServiceWithTravelPeriod(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)
		validationID := testValidateV2(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		balanceBefore := tt.Environment.Balance(ctx, t, tt.distributorBot)
		tokenID, _, mintRespPrice := testMintV2(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID)
		verifyBookingTokenStateBoughtWithPriceV2(ctx, t, tt.Environment, tt.distributorBot, tokenID, mintRespPrice, balanceBefore)
	})
}

func (tt *TestAccommodationV2) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationProductListServiceV2,
		botGenerated.AccommodationProductInfoServiceV2,
		botGenerated.AccommodationSearchServiceV2,
		botGenerated.ValidationServiceV2,
		botGenerated.MintServiceV2,
	))

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.AccommodationProductListServiceV2, Fee: 100},
			{Name: botGenerated.AccommodationProductInfoServiceV2, Fee: 110},
			{Name: botGenerated.AccommodationSearchServiceV2, Fee: 120},
			{Name: botGenerated.ValidationServiceV2, Fee: 130},
			{Name: botGenerated.MintServiceV2, Fee: 140},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

// Simple product list request which shall return all properties. Checking if all are present
func (tt *TestAccommodationV2) testAccommodationV2ProductListService(ctx context.Context, t *testing.T) {
	hotelCodes := []string{
		"HOTEL123456",
		"HOTEL789012",
		"HOTEL345678",
		"HOTEL901234",
		"HOTEL567890",
	}

	expectedTotalResults := len(hotelCodes)

	req := &accommodationv2.AccommodationProductListRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
	}
	resp, err := tt.distributorBot.AccommodationProductListServiceV2.AccommodationProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain all properties defined by the pp-mock (defined by hotelCodes/expectedTotalResults)
	require.Len(t, resp.Properties, expectedTotalResults, "unexpected number of properties in response")

	for i := range hotelCodes {
		require.NotEmpty(t, resp.Properties[i].SupplierCode, "unexpected empty response properties[%d].SupplierCode", i)
		require.NotEmpty(t, resp.Properties[i].SupplierCode.SupplierCode, "unexpected empty response properties[%d].SupplierCode.SupplierCode", i)
		require.Contains(t, hotelCodes, resp.Properties[i].SupplierCode.SupplierCode, "unexpected response properties[%d].SupplierCode.SupplierCode", i)
	}
}

// Product list request with a modification filter set. It should only return one fitting result.
func (tt *TestAccommodationV2) testAccommodationV2ProductListServiceWithFilter(ctx context.Context, t *testing.T) {
	// Modification timestamp which should exactly return one result (see hotelCode).
	// See the properties.json file in the pp-mock for more info
	const modifiedAfterSecs int64 = 1710489050
	const hotelCode = "HOTEL567890"

	req := &accommodationv2.AccommodationProductListRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		ModifiedAfter: &timestamppb.Timestamp{
			Seconds: modifiedAfterSecs,
		},
	}
	resp, err := tt.distributorBot.AccommodationProductListServiceV2.AccommodationProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain only one property as only one is modified after the given timestamp
	require.Len(t, resp.Properties, 1, "unexpected number of properties in response")

	require.NotEmpty(t, resp.Properties[0].SupplierCode, "unexpected empty response properties[0].SupplierCode")
	require.NotEmpty(t, resp.Properties[0].SupplierCode.SupplierCode, "unexpected empty response properties[0].SupplierCode.SupplierCode")
	require.Equal(t, hotelCode, resp.Properties[0].SupplierCode.SupplierCode, "unexpected response properties[0].SupplierCode.SupplierCode")
}

// Get detailed accommodation information for a specific hotel code (supplier code).
func (tt *TestAccommodationV2) testAccommodationV2ProductInfoService(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL789012"

	req := &accommodationv2.AccommodationProductInfoRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SupplierCodes: []*typesv2.SupplierProductCode{
			{SupplierCode: hotelCode},
		},
	}
	resp, err := tt.distributorBot.AccommodationProductInfoServiceV2.AccommodationProductInfo(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain only the one property filtered in the request
	require.Len(t, resp.Properties, 1, "unexpected number of properties in response")

	require.NotEmpty(t, resp.Properties[0].Property, "unexpected empty response properties[0].Property")
	require.NotEmpty(t, resp.Properties[0].Property.SupplierCode, "unexpected empty response properties[0].SupplierCode")
	require.NotEmpty(t, resp.Properties[0].Property.SupplierCode.SupplierCode, "unexpected empty response properties[0].SupplierCode.SupplierCode")
	require.Equal(t, hotelCode, resp.Properties[0].Property.SupplierCode.SupplierCode, "unexpected response properties[0].SupplierCode.SupplierCode")

	// Let's also check for some other properties of the response
	require.NotEmpty(t, resp.Properties[0].Images, "unexpected empty response properties[0].Images")
	require.Len(t, resp.Properties[0].Images, 1, "unexpected number of images in response")
	require.Equal(t, resp.Properties[0].Images[0].File.Name, "Beach House", "unexpected image name")

	require.NotEmpty(t, resp.Properties[0].Videos, "unexpected empty response properties[0].Videos")
	require.Len(t, resp.Properties[0].Videos, 1, "unexpected number of videos in response")
	require.Equal(t, resp.Properties[0].Videos[0].File.Url, "https://example.com/videos/resort-tour.mp4", "unexpected video url")

	require.NotEmpty(t, resp.Properties[0].Rooms, "unexpected empty response properties[0].Rooms")
	require.Len(t, resp.Properties[0].Rooms, 1, "unexpected number of rooms in response")
	require.Equal(t, resp.Properties[0].Rooms[0].SupplierCode, "DBL-MTN", "unexpected room code")
	require.Equal(t, resp.Properties[0].Rooms[0].TotalOccupancy.MinGuests, int32(1), "unexpected min guests")
	require.Equal(t, resp.Properties[0].Rooms[0].TotalOccupancy.MaxGuests, int32(3), "unexpected max guests")
	require.Equal(t, resp.Properties[0].Rooms[0].TotalOccupancy.StandardOccupancy, int32(2), "unexpected standard occupancy")
	require.Equal(t, resp.Properties[0].Rooms[0].TotalOccupancy.FullPayers, int32(2), "unexpected full payers")
}

func (tt *TestAccommodationV2) testAccommodationV2SearchServiceWithoutCurrency(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL345678"

	req := &accommodationv2.AccommodationSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Queries: []*accommodationv2.AccommodationSearchQuery{{
			SearchParametersAccommodation: &accommodationv2.AccommodationSearchParameters{
				SupplierCodes: []*typesv2.SupplierProductCode{
					{SupplierCode: hotelCode},
				},
			},
		}},
	}
	resp, err := tt.distributorBot.AccommodationSearchServiceV2.AccommodationSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search without the mandatory travel period given. Expect an error to be returned back.
func (tt *TestAccommodationV2) testAccommodationV2SearchServiceWithoutTravelPeriod(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL345678"

	req := &accommodationv2.AccommodationSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParametersGeneric: &typesv2.SearchParameters{
			Currency: &typesv2.Currency{Currency: &typesv2.Currency_NativeToken{}},
		},
		Queries: []*accommodationv2.AccommodationSearchQuery{{
			SearchParametersAccommodation: &accommodationv2.AccommodationSearchParameters{
				SupplierCodes: []*typesv2.SupplierProductCode{
					{SupplierCode: hotelCode},
				},
			},
		}},
	}
	resp, err := tt.distributorBot.AccommodationSearchServiceV2.AccommodationSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search with wrong travel periods given: travel period outside of allowed constraints. Expect errors to be returned.
func (tt *TestAccommodationV2) testAccommodationV2SearchServiceTravelPeriodOutOfBounds(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL345678"

	const nights = 12                                 // 12 nights
	startDate := time.Now().Add(time.Hour * 24 * 100) // in 100 days, outside of allowed travel period
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &accommodationv2.AccommodationSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParametersGeneric: &typesv2.SearchParameters{
			Currency: &typesv2.Currency{Currency: &typesv2.Currency_NativeToken{}},
		},
		Queries: []*accommodationv2.AccommodationSearchQuery{{
			SearchParametersAccommodation: &accommodationv2.AccommodationSearchParameters{
				SupplierCodes: []*typesv2.SupplierProductCode{
					{SupplierCode: hotelCode},
				},
			},
			TravelPeriod: &typesv1.TravelPeriod{
				StartDate: common.TimeToDateV1(startDate),
				EndDate:   common.TimeToDateV1(endDate),
			},
		}},
	}
	resp, err := tt.distributorBot.AccommodationSearchServiceV2.AccommodationSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search with wrong travel periods given: start date after end date. Expect errors to be returned.
func (tt *TestAccommodationV2) testAccommodationV2SearchServiceTravelPeriodReversed(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL345678"

	const nights = 12                                                // 12 nights
	endDate := time.Now().Add(time.Hour * 24)                        // tomorrow
	startDate := endDate.Add(time.Hour * 24 * time.Duration(nights)) // start date after end date

	req := &accommodationv2.AccommodationSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParametersGeneric: &typesv2.SearchParameters{
			Currency: &typesv2.Currency{Currency: &typesv2.Currency_NativeToken{}},
		},
		Queries: []*accommodationv2.AccommodationSearchQuery{{
			SearchParametersAccommodation: &accommodationv2.AccommodationSearchParameters{
				SupplierCodes: []*typesv2.SupplierProductCode{
					{SupplierCode: hotelCode},
				},
			},
			TravelPeriod: &typesv1.TravelPeriod{
				StartDate: common.TimeToDateV1(startDate),
				EndDate:   common.TimeToDateV1(endDate),
			},
		}},
	}
	resp, err := tt.distributorBot.AccommodationSearchServiceV2.AccommodationSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search with a valid travel period. Expect valid search results.
func testAccommodationV2SearchServiceWithTravelPeriod(
	ctx context.Context,
	t *testing.T,
	tt *suite.Environment,
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

	req := &accommodationv2.AccommodationSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParametersGeneric: &typesv2.SearchParameters{
			Currency: &typesv2.Currency{Currency: &typesv2.Currency_NativeToken{}},
		},
		Queries: []*accommodationv2.AccommodationSearchQuery{{
			SearchParametersAccommodation: &accommodationv2.AccommodationSearchParameters{
				SupplierCodes: []*typesv2.SupplierProductCode{
					{SupplierCode: "HOTEL345678"},
					{SupplierCode: "HOTEL789012"},
				},
			},
			TravelPeriod: &typesv1.TravelPeriod{
				StartDate: common.TimeToDateV1(startDate),
				EndDate:   common.TimeToDateV1(endDate),
			},
		}},
	}
	resp, err := distributorBot.AccommodationSearchServiceV2.AccommodationSearch(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// We expect 2 results - let's check for the 2nd one
	require.Len(t, resp.Results, 2, "unexpected number of results in response")

	// Let's check if result is as expected
	require.NotEmpty(t, resp.Results[1].Units, "unexpected empty response Results[1].Units")
	require.Equal(t, resp.Results[1].Units[0].SupplierCode.SupplierCode, "HOTEL345678", "unexpected response Results[1].Units[0].SupplierCode.SupplierCode")

	// Check if the price per night is set correctly
	pricePerNight := protoPriceBigV2(t, resp.Results[1].Units[0].PriceDetail.Price)
	require.True(t, pricePerNight.Cmp(common.DefaultPricePerNightNativeTokenBig) == 0, "unexpected price per night: got %s, expected %s", pricePerNight.String(), common.DefaultPricePerNightNativeTokenBig.String())

	// Extract the total price from the response
	totalPrice = protoPriceBigV2(t, resp.Results[1].TotalPriceDetail.Price)

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

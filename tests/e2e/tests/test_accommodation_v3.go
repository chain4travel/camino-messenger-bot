// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	accommodationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v13/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/suite"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestAccommodationV3)(nil)

func init() {
	Tests["AccommodationV3"] = &TestAccommodationV3{}
}

type TestAccommodationV3 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestAccommodationV3) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestAccommodationV3) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the properties
		tt.testAccommodationV3ProductListService(ctx, t)
	})
	t.Run("Product list with filter", func(t *testing.T) {
		// Happy path: will return only one property
		tt.testAccommodationV3ProductListServiceWithFilter(ctx, t)
	})
	t.Run("Product info", func(t *testing.T) {
		// Happy path: will return the detailed info of a property
		tt.testAccommodationV3ProductInfoService(ctx, t)
	})
	t.Run("Search w/o currency", func(t *testing.T) {
		// ERROR path: without currency it should return an error
		tt.testAccommodationV3SearchServiceWithoutCurrency(ctx, t)
	})
	t.Run("Search w/o travel period", func(t *testing.T) {
		// ERROR path: without travel period it should return an error
		tt.testAccommodationV3SearchServiceWithoutTravelPeriod(ctx, t)
	})
	t.Run("Search with travel period oob", func(t *testing.T) {
		// ERROR path: with travel period outside of allowed constraints it should return an error
		tt.testAccommodationV3SearchServiceTravelPeriodOutOfBounds(ctx, t)
	})
	t.Run("Search with travel period reversed", func(t *testing.T) {
		// ERROR path: with travel period reversed it should return an error
		tt.testAccommodationV3SearchServiceTravelPeriodReversed(ctx, t)
	})
	t.Run("Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		searchID, resultID, totalPrice := testAccommodationV3SearchServiceWithTravelPeriod(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)
		validationID := testValidateV3(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		balanceBefore := tt.Balance(ctx, t, tt.distributorBot)
		tokenID, _, mintRespPrice := testMintV3(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID)
		verifyBookingTokenStateBoughtWithPriceV3(ctx, t, tt.Environment, tt.distributorBot, tokenID, mintRespPrice, balanceBefore)
	})
}

func (tt *TestAccommodationV3) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationProductListServiceV3,
		botGenerated.AccommodationProductInfoServiceV3,
		botGenerated.AccommodationSearchServiceV3,
		botGenerated.ValidationServiceV3,
		botGenerated.MintServiceV3,
	))

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.AccommodationProductListServiceV3, Fee: 100},
			{Name: botGenerated.AccommodationProductInfoServiceV3, Fee: 110},
			{Name: botGenerated.AccommodationSearchServiceV3, Fee: 120},
			{Name: botGenerated.ValidationServiceV3, Fee: 130},
			{Name: botGenerated.MintServiceV3, Fee: 140},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

// Simple product list request which shall return all properties. Checking if all are present
func (tt *TestAccommodationV3) testAccommodationV3ProductListService(ctx context.Context, t *testing.T) {
	hotelCodes := []string{
		"HOTEL123456",
		"HOTEL789012",
		"HOTEL345678",
		"HOTEL901234",
		"HOTEL567890",
	}

	expectedTotalResults := len(hotelCodes)

	req := &accommodationv3.AccommodationProductListRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
	}
	resp, err := tt.distributorBot.AccommodationProductListServiceV3.AccommodationProductList(
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
func (tt *TestAccommodationV3) testAccommodationV3ProductListServiceWithFilter(ctx context.Context, t *testing.T) {
	// Modification timestamp which should exactly return one result (see hotelCode).
	// See the properties.json file in the pp-mock for more info
	const modifiedAfterSecs int64 = 1710489050
	const hotelCode = "HOTEL567890"

	req := &accommodationv3.AccommodationProductListRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		ModifiedAfter: &timestamppb.Timestamp{
			Seconds: modifiedAfterSecs,
		},
	}
	resp, err := tt.distributorBot.AccommodationProductListServiceV3.AccommodationProductList(
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
func (tt *TestAccommodationV3) testAccommodationV3ProductInfoService(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL789012"

	req := &accommodationv3.AccommodationProductInfoRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SupplierCodes: []*typesv2.SupplierProductCode{
			{SupplierCode: hotelCode},
		},
	}
	resp, err := tt.distributorBot.AccommodationProductInfoServiceV3.AccommodationProductInfo(
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

func (tt *TestAccommodationV3) testAccommodationV3SearchServiceWithoutCurrency(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL345678"

	req := &accommodationv3.AccommodationSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Queries: []*accommodationv3.AccommodationSearchQuery{{
			SearchParametersAccommodation: &accommodationv2.AccommodationSearchParameters{
				SupplierCodes: []*typesv2.SupplierProductCode{
					{SupplierCode: hotelCode},
				},
			},
		}},
	}
	resp, err := tt.distributorBot.AccommodationSearchServiceV3.AccommodationSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search without the mandatory travel period given. Expect an error to be returned back.
func (tt *TestAccommodationV3) testAccommodationV3SearchServiceWithoutTravelPeriod(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL345678"

	req := &accommodationv3.AccommodationSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParametersGeneric: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{Currency: &typesv3.Currency_NativeToken{}},
		},
		Queries: []*accommodationv3.AccommodationSearchQuery{{
			SearchParametersAccommodation: &accommodationv2.AccommodationSearchParameters{
				SupplierCodes: []*typesv2.SupplierProductCode{
					{SupplierCode: hotelCode},
				},
			},
		}},
	}
	resp, err := tt.distributorBot.AccommodationSearchServiceV3.AccommodationSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search with wrong travel periods given: travel period outside of allowed constraints. Expect errors to be returned.
func (tt *TestAccommodationV3) testAccommodationV3SearchServiceTravelPeriodOutOfBounds(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL345678"

	startDate := time.Now().Add(common.TravelPeriodMinStartOffset + common.TravelPeriodMaxDuration + 24*time.Hour) // outside of allowed travel period
	endDate := startDate.Add(time.Hour * 24)

	req := &accommodationv3.AccommodationSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParametersGeneric: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{Currency: &typesv3.Currency_NativeToken{}},
		},
		Queries: []*accommodationv3.AccommodationSearchQuery{{
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
	resp, err := tt.distributorBot.AccommodationSearchServiceV3.AccommodationSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search with wrong travel periods given: start date after end date. Expect errors to be returned.
func (tt *TestAccommodationV3) testAccommodationV3SearchServiceTravelPeriodReversed(ctx context.Context, t *testing.T) {
	const hotelCode = "HOTEL345678"

	const nights = 12                                              // 12 nights
	startDate := time.Now().Add(common.TravelPeriodMinStartOffset) // tomorrow
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &accommodationv3.AccommodationSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParametersGeneric: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{Currency: &typesv3.Currency_NativeToken{}},
		},
		Queries: []*accommodationv3.AccommodationSearchQuery{{
			SearchParametersAccommodation: &accommodationv2.AccommodationSearchParameters{
				SupplierCodes: []*typesv2.SupplierProductCode{
					{SupplierCode: hotelCode},
				},
			},
			TravelPeriod: &typesv1.TravelPeriod{
				StartDate: common.TimeToDateV1(endDate),   // End date used as start
				EndDate:   common.TimeToDateV1(startDate), // Start date used as end
			},
		}},
	}
	resp, err := tt.distributorBot.AccommodationSearchServiceV3.AccommodationSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search with a valid travel period. Expect valid search results.
func testAccommodationV3SearchServiceWithTravelPeriod(
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
	const nights = 12                                              // 12 nights
	startDate := time.Now().Add(common.TravelPeriodMinStartOffset) // tomorrow
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &accommodationv3.AccommodationSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParametersGeneric: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{Currency: &typesv3.Currency_NativeToken{}},
		},
		Queries: []*accommodationv3.AccommodationSearchQuery{{
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
	resp, err := distributorBot.AccommodationSearchServiceV3.AccommodationSearch(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// We expect 2 results - let's check for the 2nd one
	require.Len(t, resp.Results, 2, "unexpected number of results in response")

	// Let's check if result is as expected
	require.NotEmpty(t, resp.Results[1].Units, "unexpected empty response Results[1].Units")
	require.Equal(t, resp.Results[1].Units[0].SupplierCode.SupplierCode, "HOTEL345678", "unexpected response Results[1].Units[0].SupplierCode.SupplierCode")

	// Check if the price per night is set correctly
	pricePerNight := protoPriceBigV3(t, resp.Results[1].Units[0].PriceDetail.Price)
	require.True(t, pricePerNight.Cmp(common.DefaultPricePerNightNativeTokenBig) == 0, "unexpected price per night: got %s, expected %s", pricePerNight.String(), common.DefaultPricePerNightNativeTokenBig.String())

	// Extract the total price from the response
	totalPrice = protoPriceBigV3(t, resp.Results[1].TotalPriceDetail.Price)

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

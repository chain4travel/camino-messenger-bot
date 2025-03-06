// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"strconv"
	"testing"
	"time"

	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	botGenerated "github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	"github.com/chain4travel/camino-messenger-bot/pkg/price"
	common "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/tests/e2e/partner_plugin"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Setting up the basic applications and services used in all sub-test-cases
func testAccommodationV2Setup(
	ctx context.Context,
	t *testing.T,
	tt *Test,
) (
	supplierPartnerPlugin *partnerplugin.PartnerPlugin,
	supplierBot *bot.Bot,
	distributorBot *bot.Bot,
) {
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationProductListServiceV2,
		botGenerated.AccommodationProductInfoServiceV2,
		botGenerated.AccommodationSearchServiceV2,
		botGenerated.ValidationServiceV2,
		botGenerated.MintServiceV2,
	))
	supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	supplierBot = tt.CreateBot(ctx, t, false, supplierPartnerPlugin, []bot.CMService{
		{Name: botGenerated.AccommodationProductListServiceV2, Fee: 100},
		{Name: botGenerated.AccommodationProductInfoServiceV2, Fee: 110},
		{Name: botGenerated.AccommodationSearchServiceV2, Fee: 120},
		{Name: botGenerated.ValidationServiceV2, Fee: 130},
		{Name: botGenerated.MintServiceV2, Fee: 140},
	})

	// bot without partnerPlugin and with rpc server (distributor)
	distributorBot = tt.CreateBot(ctx, t, true, nil, nil)

	return supplierPartnerPlugin, supplierBot, distributorBot
}

// Simple product list request which shall return all properties. Checking if all are present
func testAccommodationV2ProductListService(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
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
	resp, err := distributorBot.AccommodationProductListServiceV2.AccommodationProductList(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)

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
func testAccommodationV2ProductListServiceWithFilter(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
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
	resp, err := distributorBot.AccommodationProductListServiceV2.AccommodationProductList(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain only one property as only one is modified after the given timestamp
	require.Len(t, resp.Properties, 1, "unexpected number of properties in response")

	require.NotEmpty(t, resp.Properties[0].SupplierCode, "unexpected empty response properties[0].SupplierCode")
	require.NotEmpty(t, resp.Properties[0].SupplierCode.SupplierCode, "unexpected empty response properties[0].SupplierCode.SupplierCode")
	require.Equal(t, hotelCode, resp.Properties[0].SupplierCode.SupplierCode, "unexpected response properties[0].SupplierCode.SupplierCode")
}

// Get detailed accommodation information for a specific hotel code (supplier code).
func testAccommodationV2ProductInfoService(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
	const hotelCode = "HOTEL789012"

	req := &accommodationv2.AccommodationProductInfoRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SupplierCodes: []*typesv2.SupplierProductCode{
			{SupplierCode: hotelCode},
		},
	}
	resp, err := distributorBot.AccommodationProductInfoServiceV2.AccommodationProductInfo(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)

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

func testAccommodationV2SearchServiceWithoutCurrency(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
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
	resp, err := distributorBot.AccommodationSearchServiceV2.AccommodationSearch(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search without the mandatory travel period given. Expect an error to be returned back.
func testAccommodationV2SearchServiceWithoutTravelPeriod(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
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
	resp, err := distributorBot.AccommodationSearchServiceV2.AccommodationSearch(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search with wrong travel periods given: travel period outside of allowed constraints. Expect errors to be returned.
func testAccommodationV2SearchServiceTravelPeriodOutOfBounds(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
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
	resp, err := distributorBot.AccommodationSearchServiceV2.AccommodationSearch(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search with wrong travel periods given: start date after end date. Expect errors to be returned.
func testAccommodationV2SearchServiceTravelPeriodReversed(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
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
	resp, err := distributorBot.AccommodationSearchServiceV2.AccommodationSearch(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test search with a valid travel period. Expect valid search results.
func testAccommodationV2SearchServiceWithTravelPeriod(
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
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// We expect 2 results - let's check for the 2nd one
	require.Len(t, resp.Results, 2, "unexpected number of results in response")

	// Let's check if result is as expected
	require.NotEmpty(t, resp.Results[1].Units, "unexpected empty response Results[1].Units")
	require.Equal(t, resp.Results[1].Units[0].SupplierCode.SupplierCode, "HOTEL345678", "unexpected response Results[1].Units[0].SupplierCode.SupplierCode")

	// Check if the price per night is set correctly
	resultPricePerNight, err := strconv.ParseFloat(resp.Results[1].Units[0].PriceDetail.Price.Value, 64)
	require.NoError(t, err)
	require.Equal(t, common.DefaultPricePerNight*100, resultPricePerNight, "unexpected price per night")

	// Extract the total price from the response
	totalPrice, err = strconv.ParseFloat(resp.Results[1].TotalPriceDetail.Price.Value, 64)
	require.NoError(t, err)

	// Check if this adds up with the total price of the unit
	require.NoError(t, err)
	require.Equal(t, common.DefaultPricePerNight*100*float64(nights), totalPrice, "unexpected total price")

	// Now extract all the values needed for the validate step which comes next
	require.NotEmpty(t, resp.Metadata, "unexpected empty response Metadata")
	require.NotEmpty(t, resp.Metadata.SearchId, "unexpected empty response Metadata.SearchId")
	require.NotEmpty(t, resp.Metadata.SearchId.Value, "unexpected empty response Metadata.SearchId.Value")

	require.NotEmpty(t, resp.Results[1].ResultId, "unexpected empty response Results[1].ResultId")

	return resp.Metadata.SearchId.Value, resp.Results[1].ResultId, totalPrice
}

// Let's test the validation step with the values extracted from the search request
func testAccommodationV2ValidateV2(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	searchID string,
	resultID int32,
	expectedTotalPrice float64,
) (validateID string) {
	req := &bookv2.ValidationRequest{
		ValidationObject: &bookv2.ValidationObject{
			SearchIdentifier: &typesv2.SearchIdentifier{
				SearchId: &typesv1.UUID{Value: searchID},
				ResultId: resultID,
			},
		},
	}
	resp, err := distributorBot.ValidationServiceV2.Validation(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// Check if the validationObject is correct in the response
	require.NotEmpty(t, resp.ValidationObject, "unexpected empty response ValidationObject")
	require.NotEmpty(t, resp.ValidationObject.SearchIdentifier, "unexpected empty response ValidationObject.SearchIdentifier")
	require.NotEmpty(t, resp.ValidationObject.SearchIdentifier.SearchId, "unexpected empty response ValidationObject.SearchIdentifier.SearchId")
	require.NotEmpty(t, resp.ValidationObject.SearchIdentifier.SearchId.Value, "unexpected empty response ValidationObject.SearchIdentifier.SearchId.Value")
	require.Equal(t, searchID, resp.ValidationObject.SearchIdentifier.SearchId.Value, "unexpected searchID in response")
	require.Equal(t, resultID, resp.ValidationObject.SearchIdentifier.ResultId, "unexpected resultID in response")

	// Check if the price per night is as expected
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
func testAccommodationV2MintV2(
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
	req := &bookv2.MintRequest{
		Header:       &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		ValidationId: &typesv1.UUID{Value: validationID},
	}
	resp, err := distributorBot.MintServiceV2.Mint(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")

	// Check if the MintId is set
	require.NotEmpty(t, resp.MintId, "unexpected empty response MintId")
	require.NotEmpty(t, resp.MintId.Value, "unexpected empty response MintId.Value")

	// check if the transaction ids are set and return them for further tests
	require.NotEmpty(t, resp.MintTransactionId, "unexpected empty response MintTransactionId")
	require.NotEmpty(t, resp.BuyTransactionId, "unexpected empty response BuyTransactionId")

	return resp.BookingTokenId, resp.Price
}

func testAccommodationV2VerifyBlockchainState(
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
	require.Equal(t, booking.BookingStatusBought, tokenStatus)
}

func TestAccommodationV2(t *testing.T, tt *Test) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()
	var supplierBot *bot.Bot
	var distributorBot *bot.Bot

	t.Run("Setup", func(t *testing.T) {
		_, supplierBot, distributorBot = testAccommodationV2Setup(ctx, t, tt)
	})
	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the properties
		testAccommodationV2ProductListService(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Product list with filter", func(t *testing.T) {
		// Happy path: will return only one property
		testAccommodationV2ProductListServiceWithFilter(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Product info", func(t *testing.T) {
		// Happy path: will return the detailed info of a property
		testAccommodationV2ProductInfoService(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Search w/o currency", func(t *testing.T) {
		// ERROR path: without currency it should return an error
		testAccommodationV2SearchServiceWithoutCurrency(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Search w/o travel period", func(t *testing.T) {
		// ERROR path: without travel period it should return an error
		testAccommodationV2SearchServiceWithoutTravelPeriod(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Search with travel period oob", func(t *testing.T) {
		// ERROR path: with travel period outside of allowed constraints it should return an error
		testAccommodationV2SearchServiceTravelPeriodOutOfBounds(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Search with travel period reversed", func(t *testing.T) {
		// ERROR path: with travel period reversed it should return an error
		testAccommodationV2SearchServiceTravelPeriodReversed(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		searchID, resultID, totalPrice := testAccommodationV2SearchServiceWithTravelPeriod(ctx, t, tt, distributorBot, supplierBot)
		validationID := testAccommodationV2ValidateV2(ctx, t, tt, distributorBot, supplierBot, searchID, resultID, totalPrice)
		tokenID, price := testAccommodationV2MintV2(ctx, t, tt, distributorBot, supplierBot, validationID)
		testAccommodationV2VerifyBlockchainState(ctx, t, tt, distributorBot, tokenID, price)
	})
}

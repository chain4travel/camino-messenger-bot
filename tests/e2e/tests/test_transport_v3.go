// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"
	"time"

	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	botGenerated "github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/tests/e2e/partner_plugin"
	"github.com/stretchr/testify/require"
)

// Setting up the basic applications and services used in all sub-test-cases
func testTransportV3Setup(
	ctx context.Context,
	t *testing.T,
	tt *Test,
) (
	supplierPartnerPlugin *partnerplugin.PartnerPlugin,
	supplierBot *bot.Bot,
	distributorBot *bot.Bot,
) {
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.TransportProductListServiceV3,
		botGenerated.TransportSearchServiceV3,
		botGenerated.ValidationServiceV3,
		botGenerated.MintServiceV3,
	))
	supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	supplierBot = tt.CreateBot(ctx, t, false, supplierPartnerPlugin, []bot.CMService{
		{Name: botGenerated.TransportProductListServiceV3, Fee: 100},
		{Name: botGenerated.TransportSearchServiceV3, Fee: 120},
		{Name: botGenerated.ValidationServiceV3, Fee: 130},
		{Name: botGenerated.MintServiceV3, Fee: 140},
	})

	// bot without partnerPlugin and with rpc server (distributor)
	distributorBot = tt.CreateBot(ctx, t, true, nil, nil)

	return supplierPartnerPlugin, supplierBot, distributorBot
}

// Simple product list request which shall return all properties. Checking if all are present
func TestTransportProductListServiceV3(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
	/*
		hotelCodes := []string{
			"HOTEL123456",
			"HOTEL789012",
			"HOTEL345678",
			"HOTEL901234",
			"HOTEL567890",
		}
	*/
	//expectedTotalResults := len(hotelCodes)

	resp, err := distributorBot.TransportProductListServiceV3.TransportProductList(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		&transportv3.TransportProductListRequest{
			Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		},
	)
	require.NoError(t, err)

	tt.logger.Debug("TransportProductListServiceV3.TransportProductList response:\n", protoMessageToJSON(tt, resp))
	/*
		require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
		require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

		// The response should contain all properties defined by the pp-mock (defined by hotelCodes/expectedTotalResults)
		require.Len(t, resp.Properties, expectedTotalResults, "unexpected number of properties in response")

		for i := range hotelCodes {
			require.NotEmpty(t, resp.Properties[i].SupplierCode, "unexpected empty response properties[%d].SupplierCode", i)
			require.NotEmpty(t, resp.Properties[i].SupplierCode.SupplierCode, "unexpected empty response properties[%d].SupplierCode.SupplierCode", i)
			require.Contains(t, hotelCodes, resp.Properties[i].SupplierCode.SupplierCode, "unexpected response properties[%d].SupplierCode.SupplierCode", i)
		}
	*/
}

/*
// Product list request with a modification filter set. It should only return one fitting result.
func TestTransportProductListServiceV3WithFilter(
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

	resp, err := distributorBot.TransportProductListServiceV3.TransportProductList(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		&transportv3.TransportProductListRequest{
			Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
			ModifiedAfter: &timestamppb.Timestamp{
				Seconds: modifiedAfterSecs,
			},
		},
	)
	require.NoError(t, err)

	tt.logger.Debug("TransportProductListServiceV3.TransportProductList response:\n", protoMessageToJSON(tt, resp))

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain only one property as only one is modified after the given timestamp
	require.Len(t, resp.Properties, 1, "unexpected number of properties in response")

	require.NotEmpty(t, resp.Properties[0].SupplierCode, "unexpected empty response properties[0].SupplierCode")
	require.NotEmpty(t, resp.Properties[0].SupplierCode.SupplierCode, "unexpected empty response properties[0].SupplierCode.SupplierCode")
	require.Equal(t, hotelCode, resp.Properties[0].SupplierCode.SupplierCode, "unexpected response properties[0].SupplierCode.SupplierCode")
}
*/

// Test product search without the mandatory query. Expect an error to be returned back.
func TestTransportProductSearchServiceV3WithoutQuery(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
	resp, err := distributorBot.TransportSearchServiceV3.TransportSearch(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		&transportv3.TransportSearchRequest{
			Header:  &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
			Queries: []*transportv3.TransportSearchQuery{},
		},
	)
	require.NoError(t, err)

	tt.logger.Debug("TransportSearchServiceV3.TransportSearch response:\n", protoMessageToJSON(tt, resp))
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

/*
// Test product search with wrong travel periods given: travel period outside of allowed constraints. Expect errors to be returned.
func TestTransportProductSearchServiceV3TravelPeriodOutOfBounds(
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

	resp, err := distributorBot.TransportSearchServiceV3.TransportSearch(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		&transportv3.TransportSearchRequest{
			Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
			Queries: []*transportv3.TransportSearchQuery{
				{
					SearchParametersTransport: &transportv3.TransportSearchParameters{
						SupplierCodes: []*typesv2.SupplierProductCode{
							{SupplierCode: hotelCode},
						},
					},
					TravelPeriod: &typesv1.TravelPeriod{
						StartDate: &typesv1.Date{
							Year:  int32(startDate.Year()),  //nolint:gosec
							Month: int32(startDate.Month()), //nolint:gosec
							Day:   int32(startDate.Day()),   //nolint:gosec
						},
						EndDate: &typesv1.Date{
							Year:  int32(endDate.Year()),  //nolint:gosec
							Month: int32(endDate.Month()), //nolint:gosec
							Day:   int32(endDate.Day()),   //nolint:gosec
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)

	tt.logger.Debug("TransportSearchServiceV3.TransportSearch response:\n", protoMessageToJSON(tt, resp))
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}
*/

// Test product search with wrong travel periods given: start date after end date. Expect errors to be returned.
func TestTransportProductSearchServiceV3TravelPeriodReversed(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) {
	const nights = 12                                                // 12 nights
	endDate := time.Now().Add(time.Hour * 24)                        // tomorrow
	startDate := endDate.Add(time.Hour * 24 * time.Duration(nights)) // start date after end date

	resp, err := distributorBot.TransportSearchServiceV3.TransportSearch(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		&transportv3.TransportSearchRequest{
			Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
			Queries: []*transportv3.TransportSearchQuery{
				{
					Travellers: []*typesv3.BasicTraveller{
						{
							TravellerId: 0,
							Type:        typesv3.TravellerType_TRAVELLER_TYPE_ADULT,
							Birthdate: &typesv1.Date{
								Year:  1980, //nolint:gosec
								Month: 1,    //nolint:gosec
								Day:   1,    //nolint:gosec
							},
							Nationality: typesv2.Country_COUNTRY_DE,
						},
						{
							TravellerId: 1,
							Type:        typesv3.TravellerType_TRAVELLER_TYPE_ADULT,
							Birthdate: &typesv1.Date{
								Year:  1980, //nolint:gosec
								Month: 1,    //nolint:gosec
								Day:   2,    //nolint:gosec
							},
							Nationality: typesv2.Country_COUNTRY_IT,
						},
					},
					Trips: []*transportv3.QueryTrip{
						{
							Departure: &transportv3.QueryTransitEvent{
								Date: &typesv1.Date{
									Year:  int32(startDate.Year()),  //nolint:gosec
									Month: int32(startDate.Month()), //nolint:gosec
									Day:   int32(startDate.Day()),   //nolint:gosec
								},
								LocationCode: &typesv2.LocationCode{
									Code: "PMI",
									Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_IATA_CODE,
								},
							},
							Arrival: &transportv3.QueryTransitEvent{
								Date: &typesv1.Date{
									Year:  int32(endDate.Year()),  //nolint:gosec
									Month: int32(endDate.Month()), //nolint:gosec
									Day:   int32(endDate.Day()),   //nolint:gosec
								},
								LocationCode: &typesv2.LocationCode{
									Code: "BCN",
									Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_IATA_CODE,
								},
							},
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)

	tt.logger.Debug("TransportSearchServiceV3.TransportSearch response:\n", protoMessageToJSON(tt, resp))
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

/*
// Test product search with a valid travel period. Expect valid search results.
func TestTransportProductSearchServiceV3WithTravelPeriod(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) (
	searchID string,
	resultID int32,
	pricePerNight float64,
) {
	const nights = 12                           // 12 nights
	startDate := time.Now().Add(time.Hour * 24) // tomorrow
	endDate := startDate.Add(time.Hour * 24 * time.Duration(nights))

	req := &transportv3.TransportSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Queries: []*transportv3.TransportSearchQuery{
			{
				SearchParametersTransport: &transportv3.TransportSearchParameters{
					SupplierCodes: []*typesv2.SupplierProductCode{
						{SupplierCode: "HOTEL345678"},
						{SupplierCode: "HOTEL789012"},
					},
				},
				TravelPeriod: &typesv1.TravelPeriod{
					StartDate: &typesv1.Date{
						Year:  int32(startDate.Year()),  //nolint:gosec
						Month: int32(startDate.Month()), //nolint:gosec
						Day:   int32(startDate.Day()),   //nolint:gosec
					},
					EndDate: &typesv1.Date{
						Year:  int32(endDate.Year()),  //nolint:gosec
						Month: int32(endDate.Month()), //nolint:gosec
						Day:   int32(endDate.Day()),   //nolint:gosec
					},
				},
			},
		},
	}

	tt.logger.Debug("TransportSearchServiceV3.TransportSearch request:\n", protoMessageToJSON(tt, req))

	resp, err := distributorBot.TransportSearchServiceV3.TransportSearch(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)

	tt.logger.Debug("TransportSearchServiceV3.TransportSearch response:\n", protoMessageToJSON(tt, resp))

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// We expect 2 results - let's check for the 2nd one
	require.Len(t, resp.Results, 2, "unexpected number of results in response")

	// Let's check if result is as expected
	require.NotEmpty(t, resp.Results[1].Units, "unexpected empty response Results[1].Units")
	require.Equal(t, resp.Results[1].Units[0].SupplierCode.SupplierCode, "HOTEL345678", "unexpected response Results[1].Units[0].SupplierCode.SupplierCode")

	// Extract the price per night from the response
	pricePerNight, err = strconv.ParseFloat(resp.Results[1].Units[0].PriceDetail.Price.Value, 64)
	require.NoError(t, err)

	// Check if this adds up with the total price of the unit
	totalPrice, err := strconv.ParseFloat(resp.Results[1].TotalPriceDetail.Price.Value, 64)
	require.NoError(t, err)
	require.Equal(t, pricePerNight*float64(nights), totalPrice, "unexpected total price")

	// Now extract all the values needed for the validate step which comes next
	require.NotEmpty(t, resp.Metadata, "unexpected empty response Metadata")
	require.NotEmpty(t, resp.Metadata.SearchId, "unexpected empty response Metadata.SearchId")
	require.NotEmpty(t, resp.Metadata.SearchId.Value, "unexpected empty response Metadata.SearchId.Value")

	require.NotEmpty(t, resp.Results[1].ResultId, "unexpected empty response Results[1].ResultId")

	return resp.Metadata.SearchId.Value, resp.Results[1].ResultId, pricePerNight
}

// Let's test the validation step with the values extracted from the search request
func TestTransportValidateV3(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	searchID string,
	resultID int32,
	pricePerNight float64,
) (validateID string) {
	resp, err := distributorBot.ValidationServiceV3.Validation(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		&bookv3.ValidationRequest{
			ValidationObject: &bookv3.ValidationObject{
				SearchIdentifier: &typesv2.SearchIdentifier{
					SearchId: &typesv1.UUID{Value: searchID},
					ResultId: resultID,
				},
			},
		},
	)
	require.NoError(t, err)

	tt.logger.Debug("ValidationServiceV3.Validation response:\n", protoMessageToJSON(tt, resp))

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
	pricePerNightResponse, err := strconv.ParseFloat(resp.PriceDetail.Price.Value, 64)
	require.NoError(t, err)
	require.Equal(t, pricePerNight, pricePerNightResponse, "unexpected price per night")

	// Last check if the validationID is set and if yes extract it and pass it back for the mint step
	require.NotEmpty(t, resp.ValidationId, "unexpected empty response validationID")
	require.NotEmpty(t, resp.ValidationId.Value, "unexpected empty response validationID.Value")
	return resp.ValidationId.Value
}

// Lastly we do the mint request based on the validation id
func TestTransportMintV3(
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
	resp, err := distributorBot.MintServiceV3.Mint(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		&bookv3.MintRequest{
			Header:       &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
			ValidationId: &typesv1.UUID{Value: validationID},
		},
	)
	require.NoError(t, err)

	tt.logger.Debug("MintServiceV3.Mint response:\n", protoMessageToJSON(tt, resp))

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// Check if the MintId is set
	require.NotEmpty(t, resp.MintId, "unexpected empty response MintId")
	require.NotEmpty(t, resp.MintId.Value, "unexpected empty response MintId.Value")

	// check if the transaction ids are set and return them for further tests
	require.NotEmpty(t, resp.MintTransactionId, "unexpected empty response MintTransactionId")
	require.NotEmpty(t, resp.BuyTransactionId, "unexpected empty response BuyTransactionId")

	return resp.BookingTokenId, resp.Price
}

func VerifyTransportBlockchainState(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	tokenID uint64,
	price *typesv2.Price,
) {
	bigTokenID := big.NewInt(0).SetUint64(tokenID)
	callOpts := &bind.CallOpts{Context: ctx}

	require.Equal(t, booking.NativePaymentToken, getPaymentTokenFromPriceV2(t, price))
	expectedReservationPrice, err := booking.ConvertPriceToBigInt(price.Value, price.Decimals, booking.NativeTokenDecimals)
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
*/

func TestTransportV3(t *testing.T, tt *Test) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()
	_, supplierBot, distributorBot := testTransportV3Setup(ctx, t, tt)

	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the products
		TestTransportProductListServiceV3(ctx, t, tt, distributorBot, supplierBot)
	})
	/*
		// TODO @Noctunus: Implement as soon as the pp-mock is ready
		t.Run("Product list with filter", func(t *testing.T) {
			// Happy path: will return only one property
			TestTransportProductListServiceV3WithFilter(ctx, t, tt, distributorBot, supplierBot)
		})
	*/
	t.Run("Product search w/o query", func(t *testing.T) {
		// ERROR path: without query it should return an error
		TestTransportProductSearchServiceV3WithoutQuery(ctx, t, tt, distributorBot, supplierBot)
	})
	t.Run("Product search with departure / arrival dates reversed", func(t *testing.T) {
		// ERROR path: with travel period reversed it should return an error
		TestTransportProductSearchServiceV3TravelPeriodReversed(ctx, t, tt, distributorBot, supplierBot)
	})
	/*
		t.Run("Product search with travel period oob", func(t *testing.T) {
			// ERROR path: with travel period outside of allowed constraints it should return an error
			TestTransportProductSearchServiceV3TravelPeriodOutOfBounds(ctx, t, tt, distributorBot, supplierBot)
		})

		t.Run("Search->Validate->Mint->Verify", func(t *testing.T) {
			searchID, resultID, pricePerNight := TestTransportProductSearchServiceV3WithTravelPeriod(ctx, t, tt, distributorBot, supplierBot)
			validationID := TestTransportValidateV3(ctx, t, tt, distributorBot, supplierBot, searchID, resultID, pricePerNight)
			tokenID, price := TestTransportMintV3(ctx, t, tt, distributorBot, supplierBot, validationID)
			VerifyTransportBlockchainState(ctx, t, tt, distributorBot, tokenID, price)
		})
	*/
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/suite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

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

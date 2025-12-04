// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/suite"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestTransportV3)(nil)

func init() {
	Tests["TransportV3"] = &TestTransportV3{}
}

type TestTransportV3 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestTransportV3) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestTransportV3) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)
	var productListResponse *transportv3.TransportProductListResponse

	t.Run("Product list", func(t *testing.T) {
		// Happy path: will just return all the products
		productListResponse = testTransportV3ProductListService(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)
	})
	t.Run("Product list with filter", func(t *testing.T) {
		// Happy path: will return only one property
		tt.testTransportV3ProductListServiceWithFilter(ctx, t)
	})
	t.Run("Product search w/o query", func(t *testing.T) {
		// ERROR path: without query it should return an error
		tt.testTransportV3SearchServiceWithoutQuery(ctx, t)
	})
	t.Run("Product search with departure / arrival dates reversed", func(t *testing.T) {
		// ERROR path: with travel period reversed it should return an error
		tt.testTransportV3SearchServiceTravelDatesReversed(ctx, t)
	})
	t.Run("Product search without arrival date", func(t *testing.T) {
		// Happy path: will return one result
		tt.testTransportV3SearchServiceTravelWithoutArrivalDate(ctx, t, productListResponse)
	})
	t.Run("Product search without arrival location", func(t *testing.T) {
		// ERROR path: without arrival location it should return an error
		tt.testTransportV3SearchServiceWithoutArrivalLocation(ctx, t, productListResponse)
	})
	t.Run("Product search with wrong travel dates", func(t *testing.T) {
		// ERROR path: with travel period outside of allowed constraints it should return an error
		tt.testTransportV3SearchServiceTravelDatesWrong(ctx, t)
	})
	t.Run("ProductList->Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		productListResponse := testTransportV3ProductListService(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)
		searchID, resultID, totalPrice := testTransportV3SearchServiceWithFilters(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, productListResponse)
		validationID := testValidateV2(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		balanceBefore := tt.Balance(ctx, t, tt.distributorBot)
		tokenID, _, mintRespPrice := testMintV2(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID)
		verifyBookingTokenStateBoughtWithPriceV2(ctx, t, tt.Environment, tt.distributorBot, tokenID, mintRespPrice, balanceBefore)
	})
}

func (tt *TestTransportV3) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.TransportProductListServiceV3,
		botGenerated.TransportSearchServiceV3,
		botGenerated.ValidationServiceV2,
		botGenerated.MintServiceV2,
	))

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.TransportProductListServiceV3, Fee: 100},
			{Name: botGenerated.TransportSearchServiceV3, Fee: 120},
			{Name: botGenerated.ValidationServiceV2, Fee: 130},
			{Name: botGenerated.MintServiceV2, Fee: 140},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

// Product list request with a modification filter set. It should only return one fitting result.
func (tt *TestTransportV3) testTransportV3ProductListServiceWithFilter(ctx context.Context, t *testing.T) {
	productCodes := []*typesv2.SupplierProductCode{
		{
			SupplierCode:   "AB",
			SupplierNumber: 4567,
		},
	}
	expectedTotalResults := len(productCodes)
	modifiedAfter := 1740500000

	req := &transportv3.TransportProductListRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		ModifiedAfter: &timestamppb.Timestamp{
			Seconds: int64(modifiedAfter),
		},
	}
	resp, err := tt.distributorBot.TransportProductListServiceV3.TransportProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain all products defined by the pp-mock (defined by productCodes/expectedTotalResults)
	require.Len(t, resp.Trips, expectedTotalResults, "unexpected number of products in response")

	// iterate over the trips in the result and check if the supplier product code matches the product code definition
	// note that the order might be different, so we need to check all of them
	for _, trip := range resp.Trips {
		found := false
		for i := range productCodes {
			if proto.Equal(trip.SupplierCode, productCodes[i]) {
				found = true
				break
			}
		}
		require.True(t, found, "unexpected response products")
	}
}

// Test product search without the mandatory query. Expect an error to be returned back.
func (tt *TestTransportV3) testTransportV3SearchServiceWithoutQuery(ctx context.Context, t *testing.T) {
	req := &transportv3.TransportSearchRequest{
		Header:  &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Queries: []*transportv3.TransportSearchQuery{},
	}
	resp, err := tt.distributorBot.TransportSearchServiceV3.TransportSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test transport search with wrong travel periods given: start date after end date. Expect errors to be returned.
func (tt *TestTransportV3) testTransportV3SearchServiceTravelDatesReversed(ctx context.Context, t *testing.T) {
	const nights = 12                                                // 12 nights
	endDate := time.Now().Add(time.Hour * 24)                        // tomorrow
	startDate := endDate.Add(time.Hour * 24 * time.Duration(nights)) // start date after end date

	req := &transportv3.TransportSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParameters: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{
				Currency: &typesv3.Currency_IsoCurrency{IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR},
			},
		},
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
							Date: common.TimeToDateV1(startDate),
							Location: &transportv3.QueryTransitEventLocation{
								Location: &transportv3.QueryTransitEventLocation_LocationCodes{
									LocationCodes: &typesv2.LocationCodes{
										Codes: []*typesv2.LocationCode{{
											Code: "PMI",
											Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_IATA_CODE,
										}},
									},
								},
							},
						},
						Arrival: &transportv3.QueryTransitEvent{
							Date: common.TimeToDateV1(endDate),
							Location: &transportv3.QueryTransitEventLocation{
								Location: &transportv3.QueryTransitEventLocation_LocationCodes{
									LocationCodes: &typesv2.LocationCodes{
										Codes: []*typesv2.LocationCode{{
											Code: "BCN",
											Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_IATA_CODE,
										}},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	resp, err := tt.distributorBot.TransportSearchServiceV3.TransportSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Test transport search with wrong travel periods given: travel period outside of allowed constraints. Expect errors to be returned.
func (tt *TestTransportV3) testTransportV3SearchServiceTravelDatesWrong(ctx context.Context, t *testing.T) {
	departureDate := time.Unix(1741959420, 0) // 14. May 2025 -- Not in mock data
	arrivalDate := time.Unix(1742045820, 0)   // 15. May 2025 -- In mock data

	req := &transportv3.TransportSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParameters: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{
				Currency: &typesv3.Currency_IsoCurrency{IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR},
			},
		},
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
							Date: common.TimeToDateV1(departureDate),
							Location: &transportv3.QueryTransitEventLocation{
								Location: &transportv3.QueryTransitEventLocation_LocationCodes{
									LocationCodes: &typesv2.LocationCodes{
										Codes: []*typesv2.LocationCode{{
											Code: "PMI",
											Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_IATA_CODE,
										}},
									},
								},
							},
						},
						Arrival: &transportv3.QueryTransitEvent{
							Date: common.TimeToDateV1(arrivalDate),
							Location: &transportv3.QueryTransitEventLocation{
								Location: &transportv3.QueryTransitEventLocation_LocationCodes{
									LocationCodes: &typesv2.LocationCodes{
										Codes: []*typesv2.LocationCode{{
											Code: "BCN",
											Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_IATA_CODE,
										}},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	resp, err := tt.distributorBot.TransportSearchServiceV3.TransportSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	// Note: an empty result is still a success as the request was valid
	// There is just no result for the given filters
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Equal(t, 0, len(resp.Results), "unexpected number of results in response")
}

func (tt *TestTransportV3) testTransportV3SearchServiceTravelWithoutArrivalDate(
	ctx context.Context,
	t *testing.T,
	productListResponse *transportv3.TransportProductListResponse,
) {
	// Extract the filters from the product list response which double also
	// as the expected results later
	// The product list request has already made sure that there are 2 results
	// And that the 2nd result has 2 segments. So just extract the values here
	firstSegmentDeparture := productListResponse.Trips[2].Segments[0].Departure
	lastSegmentArrival := productListResponse.Trips[2].Segments[1].Arrival

	departureDate := time.Unix(firstSegmentDeparture.DateTime.Seconds, 0)
	departureLocationCode := firstSegmentDeparture.Location.GetLocationCode()
	arrivalLocationCode := lastSegmentArrival.Location.GetLocationCode()

	req := &transportv3.TransportSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParameters: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{
				Currency: &typesv3.Currency_IsoCurrency{IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR},
			},
		},
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
							Date: common.TimeToDateV1(departureDate),
							Location: &transportv3.QueryTransitEventLocation{
								Location: &transportv3.QueryTransitEventLocation_LocationCodes{
									LocationCodes: &typesv2.LocationCodes{
										Codes: []*typesv2.LocationCode{departureLocationCode},
									},
								},
							},
						},
						Arrival: &transportv3.QueryTransitEvent{
							Location: &transportv3.QueryTransitEventLocation{
								Location: &transportv3.QueryTransitEventLocation_LocationCodes{
									LocationCodes: &typesv2.LocationCodes{
										Codes: []*typesv2.LocationCode{arrivalLocationCode},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	resp, err := tt.distributorBot.TransportSearchServiceV3.TransportSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	// We expect 1 result
	require.Len(t, resp.Results, 1, "unexpected number of results in response")

	// Let's check if result is as expected
	require.NotEmpty(t, resp.Results[0].TravellingTrips, "unexpected empty response Results[0].TravellingTrips")

	// We expect 2 segments in the trip
	require.Len(t, resp.Results[0].TravellingTrips[0].Segments, 2, "unexpected number of segments in response")

	// Check if the departure of the first segment is right
	require.NotEmpty(t, resp.Results[0].TravellingTrips[0].Segments[0].Info.Departure, "unexpected empty response Results[0].TravellingTrips[0].Segments[0].Info.Departure")

	require.True(t, proto.Equal(departureLocationCode, resp.Results[0].TravellingTrips[0].Segments[0].Info.Departure.Location.GetLocationCode()), "unexpected departure location code")
	// Now extract all the values needed for the validate step which comes next
	require.NotEmpty(t, resp.Metadata, "unexpected empty response Metadata")
	require.NotEmpty(t, resp.Metadata.SearchId, "unexpected empty response Metadata.SearchId")
	require.NotEmpty(t, resp.Metadata.SearchId.Value, "unexpected empty response Metadata.SearchId.Value")
	require.NotEmpty(t, resp.Results[0].ResultId, "unexpected empty response Results[1].ResultId")
}

func (tt *TestTransportV3) testTransportV3SearchServiceWithoutArrivalLocation(
	ctx context.Context,
	t *testing.T,
	productListResponse *transportv3.TransportProductListResponse,
) {
	// Extract the filters from the product list response which double also
	// as the expected results later
	// The product list request has already made sure that there are 2 results
	// And that the 2nd result has 2 segments. So just extract the values here
	firstSegmentDeparture := productListResponse.Trips[2].Segments[0].Departure
	lastSegmentArrival := productListResponse.Trips[2].Segments[1].Arrival

	departureDate := time.Unix(firstSegmentDeparture.DateTime.Seconds, 0)
	departureLocationCode := firstSegmentDeparture.Location.GetLocationCode()
	arrivalDate := time.Unix(lastSegmentArrival.DateTime.Seconds, 0)

	req := &transportv3.TransportSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParameters: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{
				Currency: &typesv3.Currency_IsoCurrency{
					IsoCurrency: *typesv3.IsoCurrency_ISO_CURRENCY_EUR.Enum(),
				},
			},
		},
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
							Date: common.TimeToDateV1(departureDate),
							Location: &transportv3.QueryTransitEventLocation{
								Location: &transportv3.QueryTransitEventLocation_LocationCodes{
									LocationCodes: &typesv2.LocationCodes{
										Codes: []*typesv2.LocationCode{departureLocationCode},
									},
								},
							},
						},
						Arrival: &transportv3.QueryTransitEvent{
							Date: common.TimeToDateV1(arrivalDate),
						},
					},
				},
			},
		},
	}
	resp, err := tt.distributorBot.TransportSearchServiceV3.TransportSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

// Simple product list request which shall return all properties. Checking if all are present
func testTransportV3ProductListService(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
) *transportv3.TransportProductListResponse {
	productCodes := []*typesv2.SupplierProductCode{
		{
			SupplierCode:   "AB",
			SupplierNumber: 4567,
		},
		{
			SupplierCode:   "LH",
			SupplierNumber: 7453,
		},
		{
			SupplierCode:   "DB",
			SupplierNumber: 5483,
		},
	}
	expectedTotalResults := len(productCodes)

	req := &transportv3.TransportProductListRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
	}
	resp, err := distributorBot.TransportProductListServiceV3.TransportProductList(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// The response should contain all products defined by the pp-mock (defined by productCodes/expectedTotalResults)
	require.Len(t, resp.Trips, expectedTotalResults, "unexpected number of products in response")

	// iterate over the trips in the result and check if the supplier product code matches the product code definition
	// note that the order might be different, so we need to check all of them
	for _, trip := range resp.Trips {
		found := false
		for i := range productCodes {
			if proto.Equal(trip.SupplierCode, productCodes[i]) {
				found = true
				break
			}
		}
		require.True(t, found, "unexpected response products")
	}

	// Verify that the 2nd result has 2 segments and that the departure and arrival locations are set
	require.Len(t, resp.Trips[1].Segments, 2, "unexpected number of segments in response")
	require.NotEmpty(t, resp.Trips[1].Segments[0].Departure, "unexpected empty response Trips[1].Segments[0].Info.Departure")
	require.NotEmpty(t, resp.Trips[1].Segments[1].Arrival, "unexpected empty response Trips[1].Segments[1].Info.Arrival")

	return resp
}

// Test product search with a valid query. Expect a valid response with results.
func testTransportV3SearchServiceWithFilters(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	productListResponse *transportv3.TransportProductListResponse,
) (
	searchID string,
	resultID int32,
	totalPrice *big.Int,
) {
	// Extract the filters from the product list response which double also
	// as the expected results later
	// The product list request has already made sure that there are 2 results
	// And that the 2nd result has 2 segments. So just extract the values here
	firstSegmentDeparture := productListResponse.Trips[1].Segments[0].Departure
	lastSegmentArrival := productListResponse.Trips[1].Segments[1].Arrival

	departureDate := time.Unix(firstSegmentDeparture.DateTime.Seconds, 0)
	arrivalDate := time.Unix(lastSegmentArrival.DateTime.Seconds, 0)
	departureLocationCode := firstSegmentDeparture.Location.GetLocationCode()
	arrivalLocationCode := lastSegmentArrival.Location.GetLocationCode()
	expectedTotalPrice, err := price.ToBigInt("750", 0, price.ISODecimals)
	require.NoError(t, err)

	req := &transportv3.TransportSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParameters: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{
				Currency: &typesv3.Currency_IsoCurrency{IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR},
			},
		},
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
							Date: common.TimeToDateV1(departureDate),
							Location: &transportv3.QueryTransitEventLocation{
								Location: &transportv3.QueryTransitEventLocation_LocationCodes{
									LocationCodes: &typesv2.LocationCodes{
										Codes: []*typesv2.LocationCode{departureLocationCode},
									},
								},
							},
						},
						Arrival: &transportv3.QueryTransitEvent{
							Date: common.TimeToDateV1(arrivalDate),
							Location: &transportv3.QueryTransitEventLocation{
								Location: &transportv3.QueryTransitEventLocation_LocationCodes{
									LocationCodes: &typesv2.LocationCodes{
										Codes: []*typesv2.LocationCode{arrivalLocationCode},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	resp, err := distributorBot.TransportSearchServiceV3.TransportSearch(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	// We expect 1 result
	require.Len(t, resp.Results, 1, "unexpected number of results in response")

	// Let's check if result is as expected
	require.NotEmpty(t, resp.Results[0].TravellingTrips, "unexpected empty response Results[0].TravellingTrips")

	// We expect 2 segments in the trip
	require.Len(t, resp.Results[0].TravellingTrips[0].Segments, 2, "unexpected number of segments in response")

	// Check if the departure of the first segment and the arrival of the last segment is right
	require.NotEmpty(t, resp.Results[0].TravellingTrips[0].Segments[0].Info.Departure, "unexpected empty response Results[0].TravellingTrips[0].Segments[0].Info.Departure")
	require.NotEmpty(t, resp.Results[0].TravellingTrips[0].Segments[1].Info.Arrival, "unexpected empty response Results[0].TravellingTrips[0].Segments[1].Info.Arrival")

	require.True(t, proto.Equal(departureLocationCode, resp.Results[0].TravellingTrips[0].Segments[0].Info.Departure.Location.GetLocationCode()), "unexpected departure location code")
	require.True(t, proto.Equal(arrivalLocationCode, resp.Results[0].TravellingTrips[0].Segments[1].Info.Arrival.Location.GetLocationCode()), "unexpected arrival location code")

	// Extract the price from the response
	totalPrice = protoPriceBigV3(t, resp.Results[0].TotalPrice.Price)
	require.True(t, totalPrice.Cmp(expectedTotalPrice) == 0, "unexpected total price: got %s, expected %s", totalPrice.String(), expectedTotalPrice.String())

	// Now extract all the values needed for the validate step which comes next
	require.NotEmpty(t, resp.Metadata, "unexpected empty response Metadata")
	require.NotEmpty(t, resp.Metadata.SearchId, "unexpected empty response Metadata.SearchId")
	require.NotEmpty(t, resp.Metadata.SearchId.Value, "unexpected empty response Metadata.SearchId.Value")
	require.NotEmpty(t, resp.Results[0].ResultId, "unexpected empty response Results[1].ResultId")

	return resp.Metadata.SearchId.Value, resp.Results[0].ResultId, totalPrice
}

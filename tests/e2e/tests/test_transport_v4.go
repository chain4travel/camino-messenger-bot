// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"
	"time"

	transportv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v4"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/suite"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestTransportV4)(nil)

func init() {
	Tests["TransportV4"] = &TestTransportV4{}
}

type TestTransportV4 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestTransportV4) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestTransportV4) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)
	t.Run("Product list with filter", func(t *testing.T) {
		tt.testTransportV4ProductListServiceWithFilter(ctx, t)
	})
	t.Run("Product search with wrong travel dates", func(t *testing.T) {
		tt.testTransportV4SearchServiceTravelDatesWrong(ctx, t)
	})
	t.Run("ProductList->Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		productListResponse := tt.testTransportV4ProductListService(ctx, t)
		searchID, resultID, totalPrice := testTransportV4SearchService(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, productListResponse)
		validationID := testValidateV4(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		balanceBefore := tt.Environment.Balance(ctx, t, tt.distributorBot)
		tokenID, _, mintRespPrice := testMintV4(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID, common.BookingTokenPriceV4)
		verifyBookingTokenStateBoughtWithPriceV4(ctx, t, tt.Environment, tt.distributorBot, tokenID, mintRespPrice, balanceBefore)
	})
}

func (tt *TestTransportV4) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.TransportProductListServiceV4,
		botGenerated.TransportSearchServiceV4,
		botGenerated.ValidationServiceV4,
		botGenerated.MintServiceV4,
	))

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.TransportProductListServiceV4, Fee: 100},
			{Name: botGenerated.TransportSearchServiceV4, Fee: 120},
			{Name: botGenerated.ValidationServiceV4, Fee: 130},
			{Name: botGenerated.MintServiceV4, Fee: 140},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

// Simple product list request which shall return all properties. Checking if all are present
func (tt *TestTransportV4) testTransportV4ProductListService(ctx context.Context, t *testing.T) *transportv4.TransportProductListResponse {
	req := &transportv4.TransportProductListRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
	}
	resp, err := tt.distributorBot.TransportProductListServiceV4.TransportProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	requireProtoSlicesElementsMatch(t, mockdata.TripsBasicV4, resp.Trips)

	return resp
}

// Product list request with a modification filter set. It should only return one fitting result.
func (tt *TestTransportV4) testTransportV4ProductListServiceWithFilter(ctx context.Context, t *testing.T) {
	modifiedAfter := time.Unix(1740500000, 0)
	expectedTrips := []*transportv4.TripBasic{}
	for _, trip := range mockdata.TripsBasicV4 {
		if trip.LastModified.AsTime().After(modifiedAfter) {
			expectedTrips = append(expectedTrips, trip)
			break
		}
	}

	req := &transportv4.TransportProductListRequest{
		Header:        &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		ModifiedAfter: timestamppb.New(modifiedAfter),
	}
	resp, err := tt.distributorBot.TransportProductListServiceV4.TransportProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	requireProtoSlicesElementsMatch(t, expectedTrips, resp.Trips)
}

// Test transport search with wrong travel periods given: travel period outside of allowed constraints. Expect errors to be returned.
func (tt *TestTransportV4) testTransportV4SearchServiceTravelDatesWrong(ctx context.Context, t *testing.T) {
	departureDate := time.Unix(1741959420, 0) // 14. May 2025 -- Not in mock data
	arrivalDate := time.Unix(1742045820, 0)   // 15. May 2025 -- In mock data

	req := &transportv4.TransportSearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{
				Currency: &typesv4.Currency_IsoCurrency{IsoCurrency: typesv4.IsoCurrency_ISO_CURRENCY_EUR},
			},
			Language: typesv1.Language_LANGUAGE_EN,
		},
		Queries: []*transportv4.TransportSearchQuery{
			{
				Travellers: []*typesv4.BasicTraveller{{Type: typesv4.TravellerType_TRAVELLER_TYPE_ADULT}},
				Trips: []*transportv4.QueryTrip{
					{
						Departure: &transportv4.QueryTransitEvent{
							Date: common.TimeToDateV4(departureDate),
							Location: &transportv4.QueryTransitEventLocation{
								Location: &transportv4.QueryTransitEventLocation_LocationCodes{
									LocationCodes: &typesv4.LocationCodes{
										Codes: []*typesv4.LocationCode{{
											Code: "PMI",
											Type: typesv4.LocationCodeType_LOCATION_CODE_TYPE_IATA_CODE,
										}},
									},
								},
							},
						},
						Arrival: &transportv4.QueryTransitEvent{
							Date: common.TimeToDateV4(arrivalDate),
							Location: &transportv4.QueryTransitEventLocation{
								Location: &transportv4.QueryTransitEventLocation_LocationCodes{
									LocationCodes: &typesv4.LocationCodes{
										Codes: []*typesv4.LocationCode{{
											Code: "BCN",
											Type: typesv4.LocationCodeType_LOCATION_CODE_TYPE_IATA_CODE,
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
	resp, err := tt.distributorBot.TransportSearchServiceV4.TransportSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Results, "expected no results in response")
}

// Test product search with a valid query. Expect a valid response with results.
func testTransportV4SearchService(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	productListResponse *transportv4.TransportProductListResponse,
) (
	searchID string,
	resultID uint32,
	totalPrice *typesv4.Price,
) {
	firstSegmentDeparture := productListResponse.Trips[1].Segments[0].Departure
	departureTime := firstSegmentDeparture.DateTime.AsTime()
	departureLocationCode := firstSegmentDeparture.Location.GetLocationCode()

	lastSegmentArrival := productListResponse.Trips[1].Segments[1].Arrival // expecting 2 segments
	arrivalTime := lastSegmentArrival.DateTime.AsTime()
	arrivalLocationCode := lastSegmentArrival.Location.GetLocationCode()

	expectedTrips := []*transportv4.TripExtended{
		tripExtendedWithSupplierCode(mockdata.TripsExtendedV4, productListResponse.Trips[1].SupplierCode),
	}

	expectedTotalPrice := &typesv4.Price{
		Value:    "750000000",
		Decimals: uint32(price.ISODecimals),
		Currency: &typesv4.Currency{
			Currency: &typesv4.Currency_IsoCurrency{IsoCurrency: typesv4.IsoCurrency_ISO_CURRENCY_EUR},
		},
	}

	req := &transportv4.TransportSearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{
				Currency: &typesv4.Currency_IsoCurrency{IsoCurrency: typesv4.IsoCurrency_ISO_CURRENCY_EUR},
			},
			Language: typesv1.Language_LANGUAGE_EN,
		},
		Queries: []*transportv4.TransportSearchQuery{{
			Travellers: []*typesv4.BasicTraveller{
				{
					TravellerId: 0,
					Type:        typesv4.TravellerType_TRAVELLER_TYPE_ADULT,
					Birthdate: &typesv4.Date{
						Year:  1980,
						Month: 1,
						Day:   1,
					},
					Nationality: typesv2.Country_COUNTRY_DE,
				},
				{
					TravellerId: 1,
					Type:        typesv4.TravellerType_TRAVELLER_TYPE_ADULT,
					Birthdate: &typesv4.Date{
						Year:  1980,
						Month: 1,
						Day:   2,
					},
					Nationality: typesv2.Country_COUNTRY_IT,
				},
			},
			Trips: []*transportv4.QueryTrip{{
				Departure: &transportv4.QueryTransitEvent{
					Date: common.TimeToDateV4(departureTime),
					Location: &transportv4.QueryTransitEventLocation{
						Location: &transportv4.QueryTransitEventLocation_LocationCodes{
							LocationCodes: &typesv4.LocationCodes{
								Codes: []*typesv4.LocationCode{departureLocationCode},
							},
						},
					},
				},
				Arrival: &transportv4.QueryTransitEvent{
					Date: common.TimeToDateV4(arrivalTime),
					Location: &transportv4.QueryTransitEventLocation{
						Location: &transportv4.QueryTransitEventLocation_LocationCodes{
							LocationCodes: &typesv4.LocationCodes{
								Codes: []*typesv4.LocationCode{arrivalLocationCode},
							},
						},
					},
				},
			}},
		}},
	}
	resp, err := distributorBot.TransportSearchServiceV4.TransportSearch(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	require.Len(t, resp.Results, 1, "unexpected number of results in response")
	require.True(t, proto.Equal(expectedTotalPrice, resp.Results[0].TotalPrice.Value), "unexpected total price: expected %v, got %v", expectedTotalPrice, resp.Results[0].TotalPrice.Value)
	requireProtoSlicesElementsMatch(t, expectedTrips, resp.Results[0].TravellingTrips)

	return resp.SearchId.Id.Value, resp.Results[0].ResultId, resp.Results[0].TotalPrice.Value
}

func tripExtendedWithSupplierCode(trips []*transportv4.TripExtended, supplierCode *typesv4.SupplierProductCode) *transportv4.TripExtended {
	for _, trip := range trips {
		if proto.Equal(trip.SupplierCode, supplierCode) {
			return common.CloneProto(trip)
		}
	}
	return nil
}

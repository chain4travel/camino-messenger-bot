// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"
	"time"

	transportv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v5"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	typesv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v5"
	"buf.build/go/protovalidate"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v13/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
	mockdata "github.com/chain4travel/camino-messenger-bot/v13/pp-mock/services/data"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v13/tests/e2e/suite"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestTransportV5)(nil)

func init() {
	Tests["TransportV5"] = &TestTransportV5{}
}

type TestTransportV5 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestTransportV5) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestTransportV5) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)
	t.Run("Product list with filter", func(t *testing.T) {
		tt.testTransportV5ProductListServiceWithFilter(ctx, t)
	})
	t.Run("Product search with wrong travel dates", func(t *testing.T) {
		tt.testTransportV5SearchServiceTravelDatesWrong(ctx, t)
	})
	t.Run("ProductList->Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		productListSuccessResponse := tt.testTransportV5ProductListService(ctx, t)
		searchID, resultID, totalPrice := testTransportV5SearchService(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, productListSuccessResponse)
		validationID := testValidateV5(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, searchID, resultID, totalPrice)
		balanceBefore := tt.Balance(ctx, t, tt.distributorBot)
		tokenID, _, mintRespPrice := testMintV5(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, validationID, common.BookingTokenPriceV5)
		verifyBookingTokenStateBoughtWithPriceV5(ctx, t, tt.Environment, tt.distributorBot, tokenID, mintRespPrice, balanceBefore)
	})
}

func (tt *TestTransportV5) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.TransportProductListServiceV5,
		botGenerated.TransportSearchServiceV5,
		botGenerated.ValidationServiceV5,
		botGenerated.MintServiceV5,
	))

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.TransportProductListServiceV5, Fee: 100},
			{Name: botGenerated.TransportSearchServiceV5, Fee: 120},
			{Name: botGenerated.ValidationServiceV5, Fee: 130},
			{Name: botGenerated.MintServiceV5, Fee: 140},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

// Simple product list request which shall return all properties. Checking if all are present
func (tt *TestTransportV5) testTransportV5ProductListService(ctx context.Context, t *testing.T) *transportv5.TransportProductListSuccessResponse {
	req := &transportv5.TransportProductListRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
	}
	resp, err := tt.distributorBot.TransportProductListServiceV5.TransportProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")

	requireProtoSlicesElementsMatch(t, mockdata.TripsBasicV5, successResp.Trips)

	return successResp
}

// Product list request with a modification filter set. It should only return one fitting result.
func (tt *TestTransportV5) testTransportV5ProductListServiceWithFilter(ctx context.Context, t *testing.T) {
	modifiedAfter := time.Unix(1740500000, 0)
	expectedTrips := []*transportv5.TripBasic{}
	for _, trip := range mockdata.TripsBasicV5 {
		if trip.LastModified.AsTime().After(modifiedAfter) {
			expectedTrips = append(expectedTrips, trip)
			break
		}
	}

	req := &transportv5.TransportProductListRequest{
		Header:        &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		ModifiedAfter: timestamppb.New(modifiedAfter),
	}
	resp, err := tt.distributorBot.TransportProductListServiceV5.TransportProductList(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")

	requireProtoSlicesElementsMatch(t, expectedTrips, successResp.Trips)
}

// Test transport search with wrong travel periods given: travel period outside of allowed constraints. Expect errors to be returned.
func (tt *TestTransportV5) testTransportV5SearchServiceTravelDatesWrong(ctx context.Context, t *testing.T) {
	departureDate := time.Unix(1741959420, 0) // 14. May 2025 -- Not in mock data
	arrivalDate := time.Unix(1742045820, 0)   // 15. May 2025 -- In mock data

	req := &transportv5.TransportSearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{
				Currency: &typesv4.Currency_IsoCurrency{IsoCurrency: typesv4.IsoCurrency_ISO_CURRENCY_EUR},
			},
			Language: typesv1.Language_LANGUAGE_EN,
		},
		Queries: []*transportv5.TransportSearchQuery{
			{
				Travellers: []*typesv4.BasicTraveller{{Type: typesv4.TravellerType_TRAVELLER_TYPE_ADULT}},
				Trips: []*transportv5.QueryTrip{
					{
						Departure: &transportv5.QueryTransitEvent{
							Date: common.TimeToDateV4(departureDate),
							Location: &transportv5.QueryTransitEventLocation{
								Location: &transportv5.QueryTransitEventLocation_LocationCodes{
									LocationCodes: &typesv4.LocationCodes{
										Codes: []*typesv4.LocationCode{{
											Code: "PMI",
											Type: typesv4.LocationCodeType_LOCATION_CODE_TYPE_IATA_CODE,
										}},
									},
								},
							},
						},
						Arrival: &transportv5.QueryTransitEvent{
							Date: common.TimeToDateV4(arrivalDate),
							Location: &transportv5.QueryTransitEventLocation{
								Location: &transportv5.QueryTransitEventLocation_LocationCodes{
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
	resp, err := tt.distributorBot.TransportSearchServiceV5.TransportSearch(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Len(t, successResp.Header.Alerts, 1, "expected one alert in response header")
	require.Empty(t, successResp.Results, "expected no results in response")
}

// Test product search with a valid query. Expect a valid response with results.
func testTransportV5SearchService(
	ctx context.Context,
	t *testing.T,
	e *suite.Environment,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	productListSuccessResponse *transportv5.TransportProductListSuccessResponse,
) (
	searchID string,
	resultID uint32,
	totalPrice *typesv5.Price,
) {
	firstSegmentDeparture := productListSuccessResponse.Trips[1].Segments[0].Departure
	departureTime := firstSegmentDeparture.DateTime.AsTime()
	departureLocationCode := firstSegmentDeparture.Location.GetLocationCode()

	lastSegmentArrival := productListSuccessResponse.Trips[1].Segments[1].Arrival // expecting 2 segments
	arrivalTime := lastSegmentArrival.DateTime.AsTime()
	arrivalLocationCode := lastSegmentArrival.Location.GetLocationCode()

	expectedTrips := []*transportv5.TripExtended{
		tripExtendedV5WithSupplierCode(mockdata.TripsExtendedV5, productListSuccessResponse.Trips[1].SupplierCode),
	}

	expectedTotalPrice := &typesv5.Price{
		Value:    "750000000",
		Decimals: uint32(price.ISODecimals),
		Currency: &typesv4.Currency{
			Currency: &typesv4.Currency_IsoCurrency{IsoCurrency: typesv4.IsoCurrency_ISO_CURRENCY_EUR},
		},
	}

	req := &transportv5.TransportSearchRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		SearchParameters: &typesv4.SearchParameters{
			Currency: &typesv4.Currency{
				Currency: &typesv4.Currency_IsoCurrency{IsoCurrency: typesv4.IsoCurrency_ISO_CURRENCY_EUR},
			},
			Language: typesv1.Language_LANGUAGE_EN,
		},
		Queries: []*transportv5.TransportSearchQuery{{
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
			Trips: []*transportv5.QueryTrip{{
				Departure: &transportv5.QueryTransitEvent{
					Date: common.TimeToDateV4(departureTime),
					Location: &transportv5.QueryTransitEventLocation{
						Location: &transportv5.QueryTransitEventLocation_LocationCodes{
							LocationCodes: &typesv4.LocationCodes{
								Codes: []*typesv4.LocationCode{departureLocationCode},
							},
						},
					},
				},
				Arrival: &transportv5.QueryTransitEvent{
					Date: common.TimeToDateV4(arrivalTime),
					Location: &transportv5.QueryTransitEventLocation{
						Location: &transportv5.QueryTransitEventLocation_LocationCodes{
							LocationCodes: &typesv4.LocationCodes{
								Codes: []*typesv4.LocationCode{arrivalLocationCode},
							},
						},
					},
				},
			}},
		}},
	}
	resp, err := distributorBot.TransportSearchServiceV5.TransportSearch(
		requestContext(ctx, supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	e.DebugPrintRequestResponse(req, resp)
	require.NoError(t, protovalidate.Validate(resp))

	successResp := resp.GetSuccessResponse()
	require.NotNil(t, successResp, "unexpected response status")
	require.Empty(t, successResp.Header.Alerts, "unexpected response alerts")

	require.Len(t, successResp.Results, 1, "unexpected number of results in response")
	require.True(t, proto.Equal(expectedTotalPrice, successResp.Results[0].TotalPrice.Value), "unexpected total price: expected %v, got %v", expectedTotalPrice, successResp.Results[0].TotalPrice.Value)
	requireProtoSlicesElementsMatch(t, expectedTrips, successResp.Results[0].TravellingTrips)

	return successResp.SearchId.Id.Value, successResp.Results[0].ResultId, successResp.Results[0].TotalPrice.Value
}

func tripExtendedV5WithSupplierCode(trips []*transportv5.TripExtended, supplierCode *typesv4.SupplierProductCode) *transportv5.TripExtended {
	for _, trip := range trips {
		if proto.Equal(trip.SupplierCode, supplierCode) {
			return common.CloneProto(trip)
		}
	}
	return nil
}

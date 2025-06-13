package tests

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	accommodationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v3"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	carbonv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/carbon/v1"
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func testCarbonCompensateV1Setup(
	ctx context.Context,
	t *testing.T,
	tt *Test,
) (
	supplierPartnerPlugin *partnerplugin.PartnerPlugin,
	supplierBot *bot.Bot,
	distributorBot *bot.Bot,
) {
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationSearchServiceV3,
		botGenerated.ValidationServiceV2,
		botGenerated.MintServiceV2,
		botGenerated.CarbonCompensateServiceV1,
	))
	supplierPartnerPlugin = tt.createPartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	supplierBot = tt.createBot(ctx, t, false, supplierPartnerPlugin, []bot.CMService{
		{Name: botGenerated.AccommodationSearchServiceV3, Fee: 120},
		{Name: botGenerated.ValidationServiceV2, Fee: 130},
		{Name: botGenerated.MintServiceV2, Fee: 140},
		{Name: botGenerated.CarbonCompensateServiceV1, Fee: 100},
	})

	// bot without partnerPlugin and with rpc server (distributor)
	distributorBot = tt.createBot(ctx, t, true, nil, nil)

	return supplierPartnerPlugin, supplierBot, distributorBot
}

func testCarbonCompensateV1Search(
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

	req := &carbonv1.CarbonCompensateSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		SearchParametersGeneric: &typesv3.SearchParameters{
			Currency: &typesv3.Currency{
				Currency: &typesv3.Currency_NativeToken{}},
		},
		Queries: []*carbonv1.CarbonSearchQuery{{
			Accommodation: []*carbonv1.AccommodationCarbonSearchQuery{{
				Id:        1,
				Reference: "booking-reference-1",
				Location: &carbonv1.AccommodationCarbonSearchQuery_LocationCode{
					LocationCode: &typesv2.LocationCode{
						Code: "HAM",
					},
				},
				CategoryRating: accommodationv3.CategoryRating_CATEGORY_RATING_4_5,
				CategoryUnit:   accommodationv3.CategoryUnit_CATEGORY_UNIT_STARS,
				Period: &typesv1.DateTimeRange{
					StartDatetime: timestamppb.New(time.Date(2024, 1, 1, 16, 0, 0, 0, time.UTC)),
					EndDatetime:   timestamppb.New(time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)),
				},
			}},
			Transport: []*carbonv1.TransportCarbonSearchQuery{{
				Id:        2,
				Reference: "booking-reference-2",
				From: &transportv3.QueryTransitEventLocation{
					Location: &transportv3.QueryTransitEventLocation_LocationCodes{
						LocationCodes: &typesv2.LocationCodes{
							Codes: []*typesv2.LocationCode{
								{
									Code: "HAM",
								},
								{
									Code: "BER",
								},
							},
						},
					},
				},
				To: &transportv3.QueryTransitEventLocation{
					Location: &transportv3.QueryTransitEventLocation_LocationCodes{
						LocationCodes: &typesv2.LocationCodes{
							Codes: []*typesv2.LocationCode{
								{
									Code: "BER",
								},
								{
									Code: "HAM",
								},
							},
						},
					},
				},
				VehicleType: "plane",
			}},
			SearchParametersCarbon: &carbonv1.CarbonSearchParameters{
				CompensationType: carbonv1.CompensationType_COMPENSATION_TYPE_CO2_DEBITS,
			},
		}},
	}

	resp, err := distributorBot.CarbonCompensateServiceV1.CarbonCompensateSearch(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status)
	require.Empty(t, resp.Header.Alerts)

	// Extract the total price from the response
	totalPrice, err = strconv.ParseFloat(resp.Results[0].TotalPrice.Value, 64)
	require.NoError(t, err)

	return resp.Metadata.SearchId.Value, resp.Results[0].ResultId, totalPrice
}

func testCarbonCompensateV1ValidateV2(
	ctx context.Context,
	t *testing.T,
	tt *Test,
	distributorBot *bot.Bot,
	supplierBot *bot.Bot,
	searchID string,
	resultID int32,
	expectedTotalPrice float64,
) (ValidationId string) {
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
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)
	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	return resp.ValidationId.Value
}

func TestCarbonCompensateV1(
	t *testing.T,
	tt *Test,
) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	_, supplierBot, distributorBot := testCarbonCompensateV1Setup(ctx, t, tt)

	t.Run("Search", func(t *testing.T) {
		searchID, resultID, totalPrice := testCarbonCompensateV1Search(ctx, t, tt, distributorBot, supplierBot)
		validationId := testCarbonCompensateV1ValidateV2(ctx, t, tt, distributorBot, supplierBot, searchID, resultID, totalPrice)
		fmt.Println("validationId", validationId)
	})
}

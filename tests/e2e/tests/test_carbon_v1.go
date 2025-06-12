package tests

import (
	"context"
	"testing"

	carbonv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/carbon/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/stretchr/testify/require"
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
) {

	req := &carbonv1.CarbonCompensateSearchRequest{
		Header: &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		Queries: []*carbonv1.CarbonSearchQuery{{
			Accommodation: []*carbonv1.AccommodationCarbonSearchQuery{{
				Reference: "booking-reference",
				Location: &carbonv1.AccommodationCarbonSearchQuery_LocationCode{
					LocationCode: &typesv2.LocationCode{
						Code: "HAM",
					},
				},
			}},
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

}

func TestCarbonCompensateV1(
	t *testing.T,
	tt *Test,
) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	_, supplierBot, distributorBot := testCarbonCompensateV1Setup(ctx, t, tt)

	t.Run("Search", func(t *testing.T) {
		testCarbonCompensateV1Search(ctx, t, tt, distributorBot, supplierBot)
	})
}

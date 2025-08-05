// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"

	seatmapv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v4"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/proto/pb/events"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/suite"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var _ suite.Test = (*TestSeatMapV4)(nil)

func init() {
	Tests["SeatMapV4"] = &TestSeatMapV4{}
}

type TestSeatMapV4 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierPPEventStream events.EventsService_SubscribeClient
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestSeatMapV4) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestSeatMapV4) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	// TODO@ test not found case
	t.Run("SeatMap", func(t *testing.T) {
		_ = tt.testSeatMapV4(ctx, t)
	})
	// TODO@ test not found case
	t.Run("Search->SeatMapAvailability with searchID", func(t *testing.T) {
		searchID, _, _ := testAccommodationV3SearchServiceWithTravelPeriod(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot) // see test_accommodation_v3.go
		tt.testSeatMapAvailabilityV4WithSearchID(ctx, t, searchID)
	})
	// TODO@ test not found case
	t.Run("ProductList->Search->Validate->Mint->VerifyBlockchain", func(t *testing.T) {
		_, mintID, _ := mintBuyTokenV4(ctx, t, tt.Environment, tt.supplierPPEventStream, tt.distributorBot, tt.supplierBot)
		tt.testSeatMapAvailabilityV4WithMintID(ctx, t, mintID)
	})
}

func (tt *TestSeatMapV4) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.SeatMapServiceV4,
		botGenerated.SeatMapAvailabilityServiceV4,
	))

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.SeatMapServiceV4, Fee: 100},
			{Name: botGenerated.SeatMapAvailabilityServiceV4, Fee: 120},
		}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)

	var err error
	tt.supplierPPEventStream, err = tt.supplierPartnerPlugin.SubscribeForEvents(ctx)
	require.NoError(t, err)
}

func (tt *TestSeatMapV4) testSeatMapV4(ctx context.Context, t *testing.T) *seatmapv4.SeatMapResponse {
	req := &seatmapv4.SeatMapRequest{
		Header:    &typesv4.RequestHeader{BaseHeader: &typesv4.Header{}},
		MapId:     mockdata.SeatMapV4[0].Id,
		Languages: []typesv1.Language{}, // TODO@ languages
	}
	resp, err := tt.distributorBot.SeatMapServiceV4.SeatMap(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	expectedSeatMap := common.CloneProto(mockdata.SeatMapV4[0])
	// TODO@ check that seat map matches mock data with selected languages
	require.True(t, proto.Equal(expectedSeatMap, resp.SeatMap), "unexpected seat map data in response")

	return resp
}

func (tt *TestSeatMapV4) testSeatMapAvailabilityV4WithSearchID(ctx context.Context, t *testing.T, searchID string) {
	req := &seatmapv4.SeatMapAvailabilityRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{}},
		Identifier: &seatmapv4.SeatMapAvailabilityRequest_SearchIdentifier{
			SearchIdentifier: &typesv4.SearchIdentifier{
				SearchId: &typesv4.UUID{Value: searchID},
			},
		},
	}
	resp, err := tt.distributorBot.SeatMapAvailabilityServiceV4.SeatMapAvailability(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")
	// TODO@
}

func (tt *TestSeatMapV4) testSeatMapAvailabilityV4WithMintID(ctx context.Context, t *testing.T, mintID string) {
	req := &seatmapv4.SeatMapAvailabilityRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{}},
		Identifier: &seatmapv4.SeatMapAvailabilityRequest_MintId{
			MintId: &typesv4.UUID{Value: mintID},
		},
	}
	resp, err := tt.distributorBot.SeatMapAvailabilityServiceV4.SeatMapAvailability(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")
	// TODO@
}

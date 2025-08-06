// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"sync"
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
	"github.com/google/uuid"
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

	t.Run("Search->SeatMapAvailability with searchID", func(t *testing.T) {
		searchID, _, _ := testAccommodationV3SearchServiceWithTravelPeriod(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot) // see test_accommodation_v3.go
		tt.testSeatMapAvailabilityV4WithSearchID(ctx, t, searchID)
	})
	t.Run("SeatMapAvailability with non-existing searchID", func(t *testing.T) {
		tt.testSeatMapAvailabilityV4WithBadSearchID(ctx, t)
	})
	t.Run("Search->Validate->Mint->SeatMapAvailability with mintID", func(t *testing.T) {
		_, mintID, _ := mintBuyTokenV3(ctx, t, tt.Environment, tt.supplierPPEventStream, tt.distributorBot, tt.supplierBot)
		tt.testSeatMapAvailabilityV4WithMintID(ctx, t, mintID)
	})
	t.Run("SeatMapAvailability with non-existing mintID", func(t *testing.T) {
		tt.testSeatMapAvailabilityV4WithBadMintID(ctx, t)
	})
	// TODO@ test not found case
	t.Run("SeatMap", func(t *testing.T) {
		tt.testSeatMapV4(ctx, t)
	})
}

func (tt *TestSeatMapV4) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.AccommodationSearchServiceV3,
		botGenerated.ValidationServiceV3,
		botGenerated.MintServiceV3,
		botGenerated.SeatMapServiceV4,
		botGenerated.SeatMapAvailabilityServiceV4,
	))

	wg := sync.WaitGroup{}

	// bot with partnerPlugin and without rpc server (supplier)
	wg.Add(1)
	go func() {
		defer wg.Done()
		tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
		tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
			bot.WithServices([]bot.CMService{
				{Name: botGenerated.AccommodationSearchServiceV3, Fee: 100},
				{Name: botGenerated.ValidationServiceV3, Fee: 110},
				{Name: botGenerated.MintServiceV3, Fee: 120},
				{Name: botGenerated.SeatMapServiceV4, Fee: 130},
				{Name: botGenerated.SeatMapAvailabilityServiceV4, Fee: 140},
			}),
		)
		var err error
		tt.supplierPPEventStream, err = tt.supplierPartnerPlugin.SubscribeForEvents(ctx)
		require.NoError(t, err)
	}()

	// bot without partnerPlugin and with rpc server (distributor)
	wg.Add(1)
	go func() {
		defer wg.Done()
		tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
	}()

	wg.Wait()
}

func (tt *TestSeatMapV4) testSeatMapAvailabilityV4WithSearchID(ctx context.Context, t *testing.T, searchID string) {
	req := &seatmapv4.SeatMapAvailabilityRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
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

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")
	// TODO@ check that seat map availability matches mock data
}

func (tt *TestSeatMapV4) testSeatMapAvailabilityV4WithBadSearchID(ctx context.Context, t *testing.T) {
	req := &seatmapv4.SeatMapAvailabilityRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		Identifier: &seatmapv4.SeatMapAvailabilityRequest_SearchIdentifier{
			SearchIdentifier: &typesv4.SearchIdentifier{
				SearchId: &typesv4.UUID{Value: uuid.NewString()},
			},
		},
	}
	resp, err := tt.distributorBot.SeatMapAvailabilityServiceV4.SeatMapAvailability(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

func (tt *TestSeatMapV4) testSeatMapAvailabilityV4WithMintID(ctx context.Context, t *testing.T, mintID string) {
	req := &seatmapv4.SeatMapAvailabilityRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
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

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")
	// TODO@ check that seat map availability matches mock data
}

func (tt *TestSeatMapV4) testSeatMapAvailabilityV4WithBadMintID(ctx context.Context, t *testing.T) {
	req := &seatmapv4.SeatMapAvailabilityRequest{
		Header: &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		Identifier: &seatmapv4.SeatMapAvailabilityRequest_MintId{
			MintId: &typesv4.UUID{Value: uuid.NewString()},
		},
	}
	resp, err := tt.distributorBot.SeatMapAvailabilityServiceV4.SeatMapAvailability(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

func (tt *TestSeatMapV4) testSeatMapV4(ctx context.Context, t *testing.T) {
	req := &seatmapv4.SeatMapRequest{
		Header:    &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		MapId:     mockdata.SeatMapV4[0].Id,
		Languages: []typesv1.Language{}, // TODO@ languages
	}
	resp, err := tt.distributorBot.SeatMapServiceV4.SeatMap(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	expectedSeatMap := common.CloneProto(mockdata.SeatMapV4[0])
	// TODO@ check that seat map matches mock data with selected languages
	require.True(t, proto.Equal(expectedSeatMap, resp.SeatMap), "unexpected seat map data in response")
}

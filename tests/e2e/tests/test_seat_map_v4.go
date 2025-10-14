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

	t.Run("Transport List->Search->SeatMapAvailability with searchID", func(t *testing.T) {
		productListResp := testTransportV3ProductListService(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)                       // see test_transport_v3.go
		searchID, _, _ := testTransportV3SearchServiceWithFilters(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, productListResp) // see test_transport_v3.go
		tt.testSeatMapAvailabilityV4WithSearchID(ctx, t, searchID)
	})
	t.Run("SeatMapAvailability with non-existing searchID", func(t *testing.T) {
		tt.testSeatMapAvailabilityV4WithBadSearchID(ctx, t)
	})
	t.Run("Search->Validate->Mint->SeatMapAvailability with mintID", func(t *testing.T) {
		_, mintID, _ := mintBuyTransportTokenV3(ctx, t, tt.Environment, tt.supplierPPEventStream, tt.distributorBot, tt.supplierBot)
		tt.testSeatMapAvailabilityV4WithMintID(ctx, t, mintID)
	})
	t.Run("SeatMapAvailability with non-existing mintID", func(t *testing.T) {
		tt.testSeatMapAvailabilityV4WithBadMintID(ctx, t)
	})
	t.Run("SeatMap non-existing seatMap id", func(t *testing.T) {
		tt.testSeatMapV4BadID(ctx, t)
	})
	t.Run("SeatMap without requested language", func(t *testing.T) {
		tt.testSeatMapV4WithoutLocalization(ctx, t)
	})
	t.Run("SeatMap", func(t *testing.T) {
		tt.testSeatMapV4(ctx, t)
	})
}

func (tt *TestSeatMapV4) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.TransportProductListServiceV3,
		botGenerated.TransportSearchServiceV3,
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
				{Name: botGenerated.TransportProductListServiceV3, Fee: 100},
				{Name: botGenerated.TransportSearchServiceV3, Fee: 110},
				{Name: botGenerated.ValidationServiceV3, Fee: 120},
				{Name: botGenerated.MintServiceV3, Fee: 130},
				{Name: botGenerated.SeatMapServiceV4, Fee: 140},
				{Name: botGenerated.SeatMapAvailabilityServiceV4, Fee: 150},
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
	require.True(t, proto.Equal(mockdata.SeatMapAvailabilityV4[0], resp.SeatMap), "unexpected seat map availability data in response")
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
	require.True(t, proto.Equal(mockdata.SeatMapAvailabilityV4[0], resp.SeatMap), "unexpected seat map availability data in response")
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

func (tt *TestSeatMapV4) testSeatMapV4BadID(ctx context.Context, t *testing.T) {
	req := &seatmapv4.SeatMapRequest{
		Header:    &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		MapId:     "non-existing-id",
		Languages: []typesv1.Language{typesv1.Language_LANGUAGE_EN},
	}
	resp, err := tt.distributorBot.SeatMapServiceV4.SeatMap(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_FAILURE, resp.Header.Status, "unexpected response status")
}

func (tt *TestSeatMapV4) testSeatMapV4WithoutLocalization(ctx context.Context, t *testing.T) {
	req := &seatmapv4.SeatMapRequest{
		Header:    &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		MapId:     mockdata.SeatMapV4[0].Id.Id,
		Languages: []typesv1.Language{typesv1.Language_LANGUAGE_AA},
	}
	resp, err := tt.distributorBot.SeatMapServiceV4.SeatMap(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	// Check response header

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Len(t, resp.Header.Alerts, 1, "expected one alert in response header")
	require.Equal(t, typesv4.AlertType_ALERT_TYPE_WARNING, resp.Header.Alerts[0].Type, "unexpected alert type in response header")

	// Check seatMap description and section names/descriptions language

	for _, section := range resp.SeatMap.Sections {
		traverseSection(section, func(s *typesv4.Section) {
			require.Empty(t, s.Names, "expected no section names")

			seatList, ok := s.SeatInfo.(*typesv4.Section_SeatList)
			if !ok {
				return
			}

			for _, seat := range seatList.SeatList.Seats {
				require.Empty(t, seat.GetAttributes().Features, "expected no seat features")
			}
		})
	}

	// Check seatMap

	expectedSeatMap := common.CloneProto(mockdata.SeatMapV4[0])

	// Set all localized strings to nil for easier comparison
	for _, section := range expectedSeatMap.Sections {
		traverseSection(section, func(s *typesv4.Section) {
			s.Names = nil

			seatList, ok := s.SeatInfo.(*typesv4.Section_SeatList)
			if !ok {
				return
			}

			for _, seat := range seatList.SeatList.Seats {
				if seat.Attributes == nil {
					continue
				}
				seat.Attributes.Features = nil
			}
		})
	}

	require.True(t, proto.Equal(expectedSeatMap, resp.SeatMap), "unexpected seat map data in response")
}

func (tt *TestSeatMapV4) testSeatMapV4(ctx context.Context, t *testing.T) {
	expectedLang := typesv1.Language_LANGUAGE_EN
	req := &seatmapv4.SeatMapRequest{
		Header:    &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		MapId:     mockdata.SeatMapV4[0].Id.Id,
		Languages: []typesv1.Language{expectedLang},
	}
	resp, err := tt.distributorBot.SeatMapServiceV4.SeatMap(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)
	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)

	// Check response header

	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")

	// Check seatMap description and section names/descriptions language
	// Also set all localized strings to nil for easier comparison

	for _, section := range resp.SeatMap.Sections {
		traverseSection(section, func(s *typesv4.Section) {
			require.Len(t, s.Names, 1, "unexpected number of section names")
			require.Equal(t, expectedLang, s.Names[0].Language, "unexpected language in section name")
			s.Names = nil

			seatList, ok := s.SeatInfo.(*typesv4.Section_SeatList)
			if !ok {
				return
			}

			for _, seat := range seatList.SeatList.Seats {
				require.Len(t, seat.GetAttributes().Features, 1, "unexpected number of seat features")
				require.Equal(t, expectedLang, seat.Attributes.Features[0].Language, "unexpected language in seat feature")
				seat.Attributes.Features = nil
			}
		})
	}

	// Check seatMap

	expectedSeatMap := common.CloneProto(mockdata.SeatMapV4[0])

	// Set all localized strings to nil for easier comparison
	for _, section := range expectedSeatMap.Sections {
		traverseSection(section, func(s *typesv4.Section) {
			s.Names = nil

			seatList, ok := s.SeatInfo.(*typesv4.Section_SeatList)
			if !ok {
				return
			}

			for _, seat := range seatList.SeatList.Seats {
				if seat.Attributes == nil {
					continue
				}
				seat.Attributes.Features = nil
			}
		})
	}

	require.True(t, proto.Equal(expectedSeatMap, resp.SeatMap), "unexpected seat map data in response")
}

func traverseSection(section *typesv4.Section, f func(*typesv4.Section)) {
	if f != nil {
		f(section)
	}
	for _, section := range section.Sections {
		traverseSection(section, f)
	}
}

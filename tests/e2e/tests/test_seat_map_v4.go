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

	t.Run("SeatMapAvailability with non-existing searchID", func(t *testing.T) {
		tt.testSeatMapAvailabilityV4WithBadSearchID(ctx, t)
	})
	t.Run("SeatMapAvailability with non-existing mintID", func(t *testing.T) {
		tt.testSeatMapAvailabilityV4WithBadMintID(ctx, t)
	})
	t.Run("Transport List->Search->SeatMapAvailability with searchID", func(t *testing.T) {
		productListResp := testTransportV3ProductListService(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot)                       // see test_transport_v3.go
		searchID, _, _ := testTransportV3SearchServiceWithFilters(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot, productListResp) // see test_transport_v3.go
		tt.testSeatMapAvailabilityV4WithSearchID(ctx, t, searchID, mockdata.SeatMapAvailabilityV4[0])
	})
	t.Run("TransportSearch->Validate->Mint->SeatMapAvailability with mintID", func(t *testing.T) {
		_, mintID, _ := mintBuyTransportTokenV3(ctx, t, tt.Environment, tt.supplierPPEventStream, tt.distributorBot, tt.supplierBot)
		tt.testSeatMapAvailabilityV4WithMintID(ctx, t, mintID, mockdata.SeatMapAvailabilityV4[0])
	})
	t.Run("ActivitySearch->SeatMapAvailability with searchID", func(t *testing.T) {
		searchID, _, _ := testActivityV3SearchServiceWithTravelPeriod(ctx, t, tt.Environment, tt.distributorBot, tt.supplierBot) // see test_activity_v3.go
		tt.testSeatMapAvailabilityV4WithSearchID(ctx, t, searchID, mockdata.SeatMapAvailabilityV4[1])
	})
	t.Run("ActivitySearch->Validate->Mint->SeatMapAvailability with mintID", func(t *testing.T) {
		_, mintID, _ := mintBuyActivityTokenV3(ctx, t, tt.Environment, tt.supplierPPEventStream, tt.distributorBot, tt.supplierBot)
		tt.testSeatMapAvailabilityV4WithMintID(ctx, t, mintID, mockdata.SeatMapAvailabilityV4[1])
	})
	t.Run("SeatMap non-existing seatMap id", func(t *testing.T) {
		tt.testSeatMapV4BadID(ctx, t)
	})
	t.Run("SeatMap without requested language", func(t *testing.T) {
		tt.testSeatMapV4WithoutLocalization(ctx, t)
	})
	t.Run("SeatMap for transport", func(t *testing.T) {
		tt.testSeatMapV4(ctx, t, mockdata.SeatMapV4[0])
	})
	t.Run("SeatMap for activity", func(t *testing.T) {
		tt.testSeatMapV4(ctx, t, mockdata.SeatMapV4[1])
	})
}

func (tt *TestSeatMapV4) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx,
		botGenerated.TransportProductListServiceV3,
		botGenerated.TransportSearchServiceV3,
		botGenerated.ActivitySearchServiceV3,
		botGenerated.ValidationServiceV3,
		botGenerated.MintServiceV3,
		botGenerated.SeatMapServiceV4,
		botGenerated.SeatMapAvailabilityServiceV4,
	))

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{
			{Name: botGenerated.TransportProductListServiceV3, Fee: 100},
			{Name: botGenerated.TransportSearchServiceV3, Fee: 110},
			{Name: botGenerated.ActivitySearchServiceV3, Fee: 120},
			{Name: botGenerated.ValidationServiceV3, Fee: 130},
			{Name: botGenerated.MintServiceV3, Fee: 140},
			{Name: botGenerated.SeatMapServiceV4, Fee: 150},
			{Name: botGenerated.SeatMapAvailabilityServiceV4, Fee: 160},
		}),
	)
	var err error
	tt.supplierPPEventStream, err = tt.supplierPartnerPlugin.SubscribeForEvents(ctx)
	require.NoError(t, err)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

func (tt *TestSeatMapV4) testSeatMapAvailabilityV4WithSearchID(ctx context.Context, t *testing.T, searchID string, expectedSeatMapInventory *typesv4.SeatMapInventory) {
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
	require.True(t, proto.Equal(expectedSeatMapInventory, resp.SeatMap), "unexpected seat map availability data in response")
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

func (tt *TestSeatMapV4) testSeatMapAvailabilityV4WithMintID(
	ctx context.Context,
	t *testing.T,
	mintID string,
	expectedSeatMapInventory *typesv4.SeatMapInventory,
) {
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
	require.True(t, proto.Equal(expectedSeatMapInventory, resp.SeatMap), "unexpected seat map availability data in response")
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

func (tt *TestSeatMapV4) testSeatMapV4(ctx context.Context, t *testing.T, expectedSeatMap *typesv4.SeatMap) {
	expectedLang := typesv1.Language_LANGUAGE_EN
	req := &seatmapv4.SeatMapRequest{
		Header:    &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		MapId:     expectedSeatMap.Id.Id,
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

	// Compare seatMap with expectedSeatMap

	expectedSeatMap = common.CloneProto(expectedSeatMap)

	// Order sections by their traversal order
	orderedSections := []*typesv4.Section{}
	for _, section := range expectedSeatMap.Sections {
		traverseSection(section, func(s *typesv4.Section) {
			orderedSections = append(orderedSections, s)
		})
	}

	// Check seatMap localized strings and strip it and expected seatMap of localized strings for easier comparison

	checkAndStripAttributes := func(expectedAttributes, attributes *typesv4.SeatAttributes, expectedLang typesv1.Language) {
		require.NotNil(t, attributes)

		// features
		var expectedFeature *typesv4.LocalizedSeatAttributeSet
		for _, feature := range expectedAttributes.Features {
			if feature.Language == expectedLang {
				expectedFeature = feature
				break
			}
		}
		expectedAttributes.Features = nil

		if expectedFeature != nil {
			require.Len(t, attributes.Features, 1)
			require.True(t, proto.Equal(expectedFeature, attributes.Features[0]), "unexpected seat map feature")
			attributes.Features = nil
		} else {
			require.Nil(t, attributes.Features)
		}

		// descriptions
		var expectedDescription *typesv4.LocalizedDescriptionSet
		for _, description := range expectedAttributes.Descriptions {
			if description.Language == expectedLang {
				expectedDescription = description
				break
			}
		}
		expectedAttributes.Descriptions = nil

		if expectedDescription != nil {
			require.Len(t, attributes.Descriptions, 1)
			require.True(t, proto.Equal(expectedDescription, attributes.Descriptions[0]), "unexpected seat map description")
			attributes.Descriptions = nil
		} else {
			require.Nil(t, attributes.Descriptions)
		}

		// restrictions
		var expectedRestriction *typesv4.LocalizedSeatAttributeSet
		for _, restriction := range expectedAttributes.Restrictions {
			if restriction.Language == expectedLang {
				expectedRestriction = restriction
				break
			}
		}
		expectedAttributes.Restrictions = nil

		if expectedRestriction != nil {
			require.Len(t, attributes.Restrictions, 1)
			require.True(t, proto.Equal(expectedRestriction, attributes.Restrictions[0]), "unexpected seat map restriction")
			attributes.Restrictions = nil
		} else {
			require.Nil(t, attributes.Restrictions)
		}
	}

	sectionIndex := 0
	for _, section := range resp.SeatMap.Sections {
		traverseSection(section, func(traversedSection *typesv4.Section) {
			expectedSection := orderedSections[sectionIndex]
			sectionIndex++

			// check name
			var expectedName *typesv4.LocalizedString
			for _, name := range expectedSection.Names {
				if name.Language == expectedLang {
					expectedName = name
					break
				}
			}
			expectedSection.Names = nil

			require.Len(t, traversedSection.Names, 1, "unexpected number of section names")
			require.True(t, proto.Equal(expectedName, traversedSection.Names[0]), "unexpected section name")
			traversedSection.Names = nil

			// check attributes
			if expectedSection.Attributes == nil {
				require.Nil(t, traversedSection.Attributes)
			} else {
				checkAndStripAttributes(expectedSection.Attributes, traversedSection.Attributes, expectedLang)
			}

			// check seats' attributes
			expectedSeatList, expectSeatList := expectedSection.SeatInfo.(*typesv4.Section_SeatList)
			seatList, hasSeatList := traversedSection.SeatInfo.(*typesv4.Section_SeatList)

			require.Equal(t, expectSeatList, hasSeatList)
			if !expectSeatList {
				return
			}

			for i, traversedSeat := range seatList.SeatList.Seats {
				expectedSeat := expectedSeatList.SeatList.Seats[i]

				if expectedSeat.Attributes == nil {
					require.Nil(t, traversedSeat.Attributes)
				} else {
					checkAndStripAttributes(expectedSeat.Attributes, traversedSeat.Attributes, expectedLang)
				}
			}
		})
	}

	require.True(t, proto.Equal(expectedSeatMap, resp.SeatMap), "unexpected seat map data in response")
}

func traverseSection(section *typesv4.Section, f func(*typesv4.Section)) {
	if f != nil {
		f(section)
	}

	for _, section := range section.GetSubsections().GetSections() {
		traverseSection(section, f)
	}
}

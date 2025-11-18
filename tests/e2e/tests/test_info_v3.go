// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"
	"time"

	infov3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/info/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/suite"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestInfoV3)(nil)

func init() {
	Tests["InfoV3"] = &TestInfoV3{}
}

type TestInfoV3 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestInfoV3) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestInfoV3) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("CountryEntryRequirements", func(t *testing.T) {
		tt.testCountryEntryRequirements(ctx, t)
	})
}

func (tt *TestInfoV3) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx, botGenerated.CountryEntryRequirementsServiceV3))

	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{{Name: botGenerated.CountryEntryRequirementsServiceV3, Fee: 100}}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

func (tt *TestInfoV3) testCountryEntryRequirements(ctx context.Context, t *testing.T) {
	startTime := time.Now()

	req := &infov3.CountryEntryRequirementsRequest{
		Header:      &typesv4.RequestHeader{BaseHeader: &typesv4.Header{Version: &typesv4.Version{}}},
		Departure:   typesv2.Country_COUNTRY_DE,
		Destination: typesv2.Country_COUNTRY_FR,
		Citizenship: typesv2.Country_COUNTRY_DE,
		Residence:   typesv2.Country_COUNTRY_DE,
		DatetimeRange: &typesv4.DateTimeRange{
			Start: timestamppb.New(startTime),
			End:   timestamppb.New(startTime.Add(time.Hour)),
		},
		Languages: []typesv1.Language{typesv1.Language_LANGUAGE_EN},
	}
	resp, err := tt.distributorBot.CountryEntryRequirementsServiceV3.CountryEntryRequirements(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)

	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv4.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")
}

// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"fmt"
	"testing"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/bot"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/common"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/suite"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ suite.Test = (*TestPingV1)(nil)

func init() {
	Tests["PingV1"] = &TestPingV1{}
}

type TestPingV1 struct {
	*suite.Environment

	supplierPartnerPlugin *partnerplugin.PartnerPlugin
	supplierBot           *bot.Bot
	distributorBot        *bot.Bot
}

func (tt *TestPingV1) Setup(e *suite.Environment) {
	tt.Environment = e
}

func (tt *TestPingV1) Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	tt.prepare(ctx, t)

	t.Run("Ping", func(t *testing.T) {
		tt.testPingV1Service(ctx, t)
	})
}

func (tt *TestPingV1) prepare(ctx context.Context, t *testing.T) {
	require.NoError(t, tt.CaminoNetwork.Client.RegisterCMServices(ctx, botGenerated.PingServiceV1))

	// bot with partnerPlugin and without rpc server (supplier)
	tt.supplierPartnerPlugin = tt.CreatePartnerPlugin(ctx, t)
	tt.supplierBot = tt.CreateBot(ctx, t, true, tt.supplierPartnerPlugin,
		bot.WithServices([]bot.CMService{{Name: botGenerated.PingServiceV1, Fee: 100}}),
	)

	// bot without partnerPlugin and with rpc server (distributor)
	tt.distributorBot = tt.CreateBot(ctx, t, true, nil)
}

func (tt *TestPingV1) testPingV1Service(ctx context.Context, t *testing.T) {
	expectedResponseMessageSubString := fmt.Sprintf("Ping response to [%s] with request ID:", common.PingMessage)

	req := &pingv1.PingRequest{
		Header:      &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		PingMessage: common.PingMessage,
		Timestamp:   timestamppb.Now(),
	}
	resp, err := tt.distributorBot.PingServiceV1.Ping(
		requestContext(ctx, tt.supplierBot.CMAccountAddress()),
		req,
	)

	require.NoError(t, err)
	tt.DebugPrintRequestResponse(req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")
	require.Contains(t, resp.PingMessage, expectedResponseMessageSubString, "unexpected response message")
}

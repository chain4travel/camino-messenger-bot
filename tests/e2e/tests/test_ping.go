// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"fmt"
	"testing"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	botGenerated "github.com/chain4travel/camino-messenger-bot/v11/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func testPingV1Setup(ctx context.Context, t *testing.T, tt *Test) (*partnerplugin.PartnerPlugin, *bot.Bot, *bot.Bot) {
	// Register all the services needed for the tests
	require.NoError(t, tt.caminoNetwork.Client.RegisterCMServices(ctx, botGenerated.PingServiceV1))
	supplierPartnerPlugin := tt.createPartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	supplierBot := tt.createBot(ctx, t, false, supplierPartnerPlugin, []bot.CMService{
		{Name: botGenerated.PingServiceV1, Fee: 100},
	})

	// bot without partnerPlugin and with rpc server (distributor)
	distributorBot := tt.createBot(ctx, t, true, nil, nil)

	return supplierPartnerPlugin, supplierBot, distributorBot
}

func testPingV1Service(ctx context.Context, t *testing.T, tt *Test, distributorBot *bot.Bot, supplierBot *bot.Bot) {
	pingMessage := "ping"
	expectedResponseMessageSubString := fmt.Sprintf("Ping response to [%s] with request ID:", pingMessage)

	req := &pingv1.PingRequest{
		Header:      &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
		PingMessage: pingMessage,
		Timestamp:   timestamppb.Now(),
	}
	resp, err := distributorBot.PingServiceV1.Ping(
		requestContext(ctx, &metadata.Metadata{
			RecipientCMAccount: supplierBot.CMAccountAddress().Hex(),
		}),
		req,
	)

	require.NoError(t, err)
	debugPrintRequestResponse(tt, getCurrentFuncName(), req, resp)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")
	require.Contains(t, resp.PingMessage, expectedResponseMessageSubString, "unexpected response message")
}

func TestPingV1(t *testing.T, tt *Test) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	_, supplierBot, distributorBot := testPingV1Setup(ctx, t, tt)

	t.Run("Ping", func(t *testing.T) {
		testPingV1Service(ctx, t, tt, distributorBot, supplierBot)
	})
}

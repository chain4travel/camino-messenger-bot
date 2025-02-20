// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"fmt"
	"testing"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	botGenerated "github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/bot"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestPingV1(t *testing.T, tt *Test) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	require.NoError(t, tt.caminoNetwork.Client.RegisterCMService(ctx, botGenerated.PingServiceV1))

	supplierPartnerPlugin := tt.CreatePartnerPlugin(ctx, t)

	// bot with partnerPlugin and without rpc server (supplier)
	supplierBot := tt.CreateBot(ctx, t, false, supplierPartnerPlugin, []bot.CMService{
		{Name: botGenerated.PingServiceV1, Fee: 100},
	})

	// bot without partnerPlugin and with rpc server (distributor)
	distributorBot := tt.CreateBot(ctx, t, true, nil, nil)

	pingMessage := "ping"
	expectedResponceMessageSubString := fmt.Sprintf("Ping response to [%s] with request ID:", pingMessage)

	resp, err := distributorBot.PingServiceV1.Ping(
		requestContext(ctx, &metadata.Metadata{
			Recipient: supplierBot.CMAccountAddress().Hex(),
		}),
		&pingv1.PingRequest{
			Header:      &typesv1.RequestHeader{BaseHeader: &typesv1.Header{}},
			PingMessage: pingMessage,
			Timestamp:   timestamppb.Now(),
		},
	)

	require.NoError(t, err)
	require.Equal(t, typesv1.StatusType_STATUS_TYPE_SUCCESS, resp.Header.Status, "unexpected response status")
	require.Empty(t, resp.Header.Alerts, "unexpected response alerts")
	require.Contains(t, resp.PingMessage, expectedResponceMessageSubString, "unexpected response message")
}

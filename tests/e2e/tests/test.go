// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/blockchain"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/matrix"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/resources"
)

// Not safe for concurrent use.
type Test struct {
	logger                 *zap.SugaredLogger
	matrix                 *matrix.Server
	caminoNetwork          *blockchain.Network
	partnerPluginFactory   *partnerplugin.Factory
	botFactory             *bot.Factory
	networkFeeKey          *ecdsa.PrivateKey
	resourceManagerSession *resources.Session
}

func (tt *Test) CreateBot(
	ctx context.Context,
	t *testing.T,
	enableRPCServer bool,
	partnerPlugin *partnerplugin.PartnerPlugin,
	services []bot.CMService,
) *bot.Bot {
	t.Helper()
	bot, errChan, err := tt.botFactory.CreateBot(ctx, enableRPCServer, partnerPlugin, services, &bot.Skip{})
	require.NoError(t, err)
	expectNoErrorAsync(t, errChan)
	return bot
}

func (tt *Test) CreatePartnerPlugin(
	ctx context.Context,
	t *testing.T,
) *partnerplugin.PartnerPlugin {
	t.Helper()
	partnerPlugin, errChan, err := tt.partnerPluginFactory.CreatePartnerPlugin(ctx)
	require.NoError(t, err)
	expectNoErrorAsync(t, errChan)
	return partnerPlugin
}

func expectChannelErrorWithTimeout(t *testing.T, errChan chan error, errContent string, timeout time.Duration) {
	t.Helper()

	select {
	case err := <-errChan:
		require.Error(t, err)
		require.Contains(t, err.Error(), errContent)
	case <-time.After(timeout):
		require.Fail(t, "timeout waiting for channel error with content: "+errContent)
	}
}

func expectNoErrorAsync(t *testing.T, errChan chan error) {
	t.Helper()
	go func() {
		require.NoError(t, <-errChan)
	}()
}

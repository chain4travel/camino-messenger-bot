// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"crypto/ecdsa"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	grpcMetadata "google.golang.org/grpc/metadata"

	messageMetadata "github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/blockchain"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/bot"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/matrix"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/resources"
)

// Not safe for concurrent use.
type Test struct {
	logger                 *zap.SugaredLogger
	matrix                 *matrix.MatrixServer
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
	bot, errChan, err := tt.botFactory.CreateBot(ctx, enableRPCServer, partnerPlugin, services)
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

func expectNoErrorAsync(t *testing.T, errChan chan error) {
	t.Helper()
	go func() {
		require.NoError(t, <-errChan)
	}()
}

func requestContext(ctx context.Context, metadata *messageMetadata.Metadata) context.Context {
	return grpcMetadata.NewOutgoingContext(ctx, metadata.ToGrpcMD())
}

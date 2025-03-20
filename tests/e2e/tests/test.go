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
	"google.golang.org/grpc/metadata"

	"github.com/chain4travel/camino-messenger-bot/tests/e2e/blockchain"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/bot"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/common"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/matrix"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/resources"
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

// ValidateTimestamps validates that the response contains the expected timestamps
// It returns the parsed timestamps map for further validation if needed
func (tt *Test) ValidateTimestamps(t *testing.T, headers metadata.MD, clientTimestampKey string) map[string]int64 {
	validator := common.NewTimestampValidator(t, tt.logger, headers)
	timestamps := validator.ValidateTimestamps()

	if clientTimestampKey != "" {
		validator.ValidateClientTimestamp(clientTimestampKey, timestamps)
	}

	return timestamps
}

// ValidateTimestampsWithPatterns validates timestamps including specific patterns
// This is a more strict validation that should be used for services that are expected
// to have specific timestamp patterns (like accommodation search)
func (tt *Test) ValidateTimestampsWithPatterns(t *testing.T, headers metadata.MD, clientTimestampKey string) map[string]int64 {
	validator := common.NewTimestampValidator(t, tt.logger, headers)
	timestamps := validator.ValidateTimestampsWithPatterns()

	if clientTimestampKey != "" {
		validator.ValidateClientTimestamp(clientTimestampKey, timestamps)
	}

	return timestamps
}

// ValidateBasicTimestamps validates that the response contains timestamps
// This is a more lenient validation that should be used for simpler services like ping
func (tt *Test) ValidateBasicTimestamps(t *testing.T, headers metadata.MD) map[string]int64 {
	validator := common.NewTimestampValidator(t, tt.logger, headers)
	return validator.ValidateBasicTimestamps()
}

// LogTimestamps logs a map of timestamps with each key on a new line
func (tt *Test) LogTimestamps(title string, timestamps map[string]int64) {
	tt.logger.Infof("%s (%d entries)", title, len(timestamps))
	for key, value := range timestamps {
		// Convert Unix millisecond timestamp to human-readable format
		timeStr := time.UnixMilli(value).Format("2006-01-02 15:04:05.000")
		tt.logger.Infof("  %s: %d (%s)", key, value, timeStr)
	}
}

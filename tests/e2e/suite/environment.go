// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package suite

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/blockchain"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/bot"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/common"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/matrix"
	partnerplugin "github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/partner_plugin"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/resources"
)

type Environment struct {
	// used by tests; we don't bother to abstract it for safety, because its tests and we expect tests to not modify those fields
	Logger        *zap.SugaredLogger
	ASB           *matrix.AppService
	CaminoNetwork *blockchain.Network

	// those are not used by tests directly, so we can hide them for at least some safety
	matrix                 *matrix.ConduitServer
	partnerPluginFactory   *partnerplugin.Factory
	botFactory             *bot.Factory
	networkFeeKey          *ecdsa.PrivateKey
	resourceManagerSession *resources.Session

	// those are expected to be set by tests within Setup method
	ASBOptions []matrix.ASBOption
}

func (e *Environment) CreateBot(
	ctx context.Context,
	t *testing.T,
	enableRPCServer bool,
	partnerPlugin *partnerplugin.PartnerPlugin,
	opts ...bot.Option,
) *bot.Bot {
	t.Helper()

	bot, err := e.botFactory.CreateBot(ctx, enableRPCServer, partnerPlugin, opts...)
	require.NoError(t, err)

	errChan, err := bot.Start(ctx)
	require.NoError(t, err)

	common.ExpectNoErrorAsync(t, errChan)
	return bot
}

func (e *Environment) CreateBotWithError(
	ctx context.Context,
	enableRPCServer bool,
	partnerPlugin *partnerplugin.PartnerPlugin,
	opts ...bot.Option,
) error {
	bot, err := e.botFactory.CreateBot(ctx, enableRPCServer, partnerPlugin, opts...)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}
	if _, err := bot.Start(ctx); err != nil {
		return fmt.Errorf("failed to start bot: %w", err)
	}
	return nil
}

func (e *Environment) CreateBotAwaitError(
	ctx context.Context,
	t *testing.T,
	enableRPCServer bool,
	partnerPlugin *partnerplugin.PartnerPlugin,
	errorContains string,
	timeout time.Duration,
	opts ...bot.Option,
) {
	t.Helper()

	bot, err := e.botFactory.CreateBot(ctx, enableRPCServer, partnerPlugin, opts...)
	require.NoError(t, err)

	errChan, err := bot.Start(ctx)
	require.NoError(t, err)

	common.AwaitError(t, errChan, errorContains, timeout)
}

func (e *Environment) RestartBot(
	ctx context.Context,
	t *testing.T,
	bot *bot.Bot,
) *bot.Bot {
	t.Helper()
	errChan, err := bot.Restart(ctx)
	require.NoError(t, err)
	common.ExpectNoErrorAsync(t, errChan)
	return bot
}

func (e *Environment) CreatePartnerPlugin(
	ctx context.Context,
	t *testing.T,
) *partnerplugin.PartnerPlugin {
	t.Helper()

	partnerPlugin, err := e.partnerPluginFactory.CreatePartnerPlugin()
	require.NoError(t, err)

	errChan, err := partnerPlugin.Start(ctx)
	require.NoError(t, err)

	common.ExpectNoErrorAsync(t, errChan)
	return partnerPlugin
}

func (e *Environment) DebugPrintProtoMessage(message proto.Message) {
	// Skip the potentially expensive conversion to JSON if debug logging is disabled
	if e.Logger.Level().Enabled(zapcore.DebugLevel) {
		e.Logger.Debugf("%s:\n%s", getTypeInfo(message), e.protoMessageToJSON(message))
	}
}

// Debug print used in each test case to print the request and response as json
func (e *Environment) DebugPrintRequestResponse(request proto.Message, response proto.Message) {
	functionName := "unknown"
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		functionName = runtime.FuncForPC(pc).Name()
	}

	// Skip the potentially expensive conversion to JSON if debug logging is disabled
	if e.Logger.Level().Enabled(zapcore.DebugLevel) {
		e.Logger.Debugf("Function: %s", functionName)
		e.Logger.Debugf("Request (%s):\n%s", getTypeInfo(request), e.protoMessageToJSON(request))
		e.Logger.Debugf("Response (%s):\n%s", getTypeInfo(response), e.protoMessageToJSON(response))
	}
}

// Service function to convert the responses into pretty-printed JSON.
// Only used for debugging and test creation.
func (e *Environment) protoMessageToJSON(message proto.Message) string {
	// Pretty-print using protojson.MarshalOptions
	marshaler := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}
	jsonData, err := marshaler.Marshal(message)
	if err != nil {
		e.Logger.Errorf("Error marshalling: %v", err)
		return ""
	}
	return string(jsonData)
}

// Get printable type information including the package path
func getTypeInfo(v interface{}) (res string) {
	if v == nil {
		return "nil"
	}
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
		res += "*"
	}
	return fmt.Sprintf("%s%s [Package: %s]", res, t.String(), t.PkgPath())
}

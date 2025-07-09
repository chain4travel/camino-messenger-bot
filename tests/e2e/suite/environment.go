// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package suite

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"reflect"
	"testing"

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

// Not safe for concurrent use.
type Environment struct {
	Logger                 *zap.SugaredLogger
	matrix                 *matrix.ConduitServer
	ASB                    *matrix.AppService
	CaminoNetwork          *blockchain.Network
	partnerPluginFactory   *partnerplugin.Factory
	botFactory             *bot.Factory
	networkFeeKey          *ecdsa.PrivateKey
	resourceManagerSession *resources.Session

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
	bot, errChan, err := e.botFactory.CreateBot(ctx, enableRPCServer, partnerPlugin, opts...)
	require.NoError(t, err)
	common.ExpectNoErrorAsync(t, errChan)
	return bot
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
	partnerPlugin, errChan, err := e.partnerPluginFactory.CreatePartnerPlugin(ctx)
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
func (e *Environment) DebugPrintRequestResponse(functionName string, request proto.Message, response proto.Message) {
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

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"testing"

	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/booking"
	messageMetadata "github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	grpcMetadata "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func requestContext(ctx context.Context, metadata *messageMetadata.Metadata) context.Context {
	return grpcMetadata.NewOutgoingContext(ctx, metadata.ToGrpcMD())
}

// Gets the current function name including the whole package path
func getCurrentFuncName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	}
	return runtime.FuncForPC(pc).Name()
}

// Get printable type information including the package path
func getTypeInfo(myvar interface{}) (res string) {
	if myvar == nil {
		return "nil"
	}
	t := reflect.TypeOf(myvar)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
		res += "*"
	}
	return fmt.Sprintf("%s%s [Package: %s]", res, t.String(), t.PkgPath())
}

// Debug print used in each test case to print the request and response as json
func debugPrintRequestResponse(tt *Test, functionName string, request proto.Message, response proto.Message) {
	// Skip the potentially expensive conversion to JSON if debug logging is disabled
	if tt.logger.Level().Enabled(zapcore.DebugLevel) {
		tt.logger.Debugf("Function: %s", functionName)
		tt.logger.Debugf("Request (%s):\n%s", getTypeInfo(request), protoMessageToJSON(tt, request))
		tt.logger.Debugf("Response (%s):\n%s", getTypeInfo(response), protoMessageToJSON(tt, response))
	}
}

func debugPrintProtoMessage(tt *Test, message proto.Message) {
	// Skip the potentially expensive conversion to JSON if debug logging is disabled
	if tt.logger.Level().Enabled(zapcore.DebugLevel) {
		tt.logger.Debugf("%s:\n%s", getTypeInfo(message), protoMessageToJSON(tt, message))
	}
}

// Service function to convert the responses into pretty-printed JSON.
// Only used for debugging and test creation.
func protoMessageToJSON(tt *Test, message proto.Message) string {
	// Pretty-print using protojson.MarshalOptions
	marshaler := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}
	jsonData, err := marshaler.Marshal(message)
	if err != nil {
		tt.logger.Errorf("Error marshalling: %v", err)
		return ""
	}
	return string(jsonData)
}

func getPaymentTokenFromPriceV2(t *testing.T, price *typesv2.Price) common.Address {
	require.NotNil(t, price, "unexpected nil price")
	switch currency := price.GetCurrency().GetCurrency().(type) {
	case *typesv2.Currency_NativeToken:
		return booking.NativePaymentToken
	case *typesv2.Currency_IsoCurrency:
		return booking.ISOPaymentToken
	case *typesv2.Currency_TokenCurrency:
		return common.HexToAddress(currency.TokenCurrency.ContractAddress)
	}
	require.Fail(t, "unexpected currency type")
	return common.Address{}
}

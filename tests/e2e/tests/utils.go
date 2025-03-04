// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"testing"

	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	messageMetadata "github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	grpcMetadata "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func requestContext(ctx context.Context, metadata *messageMetadata.Metadata) context.Context {
	return grpcMetadata.NewOutgoingContext(ctx, metadata.ToGrpcMD())
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

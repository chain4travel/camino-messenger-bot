// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/booking"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/conversion"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/suite"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	grpcMetadata "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

var Tests = make(map[string]suite.Test)

const defaultTestTimeout = 120 * time.Second

type SupplierOrDistributor uint8

const (
	Supplier SupplierOrDistributor = iota
	Distributor
)

func requestContext(ctx context.Context, recipientCMAccount common.Address) context.Context {
	return grpcMetadata.NewOutgoingContext(ctx, grpcMetadata.Pairs(
		metadata.KeyRecipientCMAccount, recipientCMAccount.Hex(),
	))
}

func priceBigV4(t *testing.T, value string, decimals int32, currency *typesv4.Currency) *big.Int { //nolint:unused // will be used in following PRs
	priceBig, err := price.ToBigInt(value, decimals, currencyDecimalsV4(t, currency))
	require.NoError(t, err)
	return priceBig
}

func protoPriceBigV4(t *testing.T, protoPrice *typesv4.Price) *big.Int { //nolint:unused // will be used in following PRs
	priceBig, err := price.ToBigInt(protoPrice.Value, conversion.MustUInt32ToInt32(protoPrice.Decimals), currencyDecimalsV4(t, protoPrice.Currency))
	require.NoError(t, err)
	return priceBig
}

func currencyDecimalsV4(t *testing.T, currency *typesv4.Currency) int32 { //nolint:unused // will be used in following PRs
	require.NotNil(t, currency)
	switch currency.Currency.(type) {
	case *typesv4.Currency_IsoCurrency:
		return price.ISODecimals
	case *typesv4.Currency_NativeToken:
		return price.NativeTokenDecimals
	}
	require.FailNow(t, "unexpected currency type in price")
	return 0
}

func protoPriceBigV3(t *testing.T, protoPrice *typesv3.Price) *big.Int {
	require.NotNil(t, protoPrice)
	var priceBig *big.Int
	var err error
	switch protoPrice.Currency.Currency.(type) {
	case *typesv3.Currency_IsoCurrency:
		priceBig, err = price.ToBigInt(protoPrice.Value, protoPrice.Decimals, price.ISODecimals)
	case *typesv3.Currency_NativeToken:
		priceBig, err = price.ToBigInt(protoPrice.Value, protoPrice.Decimals, price.NativeTokenDecimals)
	default:
		require.FailNow(t, "unexpected currency type in price")
		return nil
	}
	require.NoError(t, err)
	return priceBig
}

func protoPriceBigV2(t *testing.T, protoPrice *typesv2.Price) *big.Int {
	require.NotNil(t, protoPrice)
	var priceBig *big.Int
	var err error
	switch protoPrice.Currency.Currency.(type) {
	case *typesv2.Currency_IsoCurrency:
		priceBig, err = price.ToBigInt(protoPrice.Value, protoPrice.Decimals, price.ISODecimals)
	case *typesv2.Currency_NativeToken:
		priceBig, err = price.ToBigInt(protoPrice.Value, protoPrice.Decimals, price.NativeTokenDecimals)
	default:
		require.FailNow(t, "unexpected currency type in price")
		return nil
	}
	require.NoError(t, err)
	return priceBig
}

func getPaymentTokenFromPriceV4(t *testing.T, price *typesv4.Price) common.Address { //nolint:unused // will be used in following PRs
	require.NotNil(t, price, "unexpected nil price")
	switch currency := price.GetCurrency().GetCurrency().(type) {
	case *typesv4.Currency_NativeToken:
		return booking.NativePaymentToken
	case *typesv4.Currency_IsoCurrency:
		return booking.ISOPaymentToken
	case *typesv4.Currency_TokenCurrency:
		return common.HexToAddress(currency.TokenCurrency.ContractAddress.Address)
	}
	require.Fail(t, "unexpected currency type")
	return common.Address{}
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

var (
	c4tFeeCutNominator   = big.NewInt(10) // 10% fee cut for C4T
	c4tFeeCutDenominator = big.NewInt(100)
)

func calculateCashIn(value *big.Int) (cashedIn *big.Int, c4tFeeCut *big.Int) { //nolint:unparam // c4tFeeCut is needed for logic clarity at least
	c4tFeeCut = big.NewInt(0).Mul(value, c4tFeeCutNominator)
	c4tFeeCut.Div(c4tFeeCut, c4tFeeCutDenominator)
	return big.NewInt(0).Sub(value, c4tFeeCut), c4tFeeCut
}

func requireProtoSlicesElementsMatch[T proto.Message](t *testing.T, expected, actual []T) { //nolint:unused // will be used in following PRs
	protoMarshal := proto.MarshalOptions{Deterministic: true}
	opts := []cmp.Option{
		cmpopts.SortSlices(func(x, y T) bool {
			xb, err := protoMarshal.Marshal(x)
			require.NoError(t, err)
			yb, err := protoMarshal.Marshal(y)
			require.NoError(t, err)
			return bytes.Compare(xb, yb) < 0
		}),
		protocmp.Transform(),
	}
	require.Truef(t,
		cmp.Equal(expected, actual, opts...),
		"Mismatch (-expected,+actual):\n%s",
		cmp.Diff(expected, actual, opts...),
	)
}

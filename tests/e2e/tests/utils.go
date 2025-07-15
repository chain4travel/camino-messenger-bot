// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/booking"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v11/tests/e2e/suite"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	grpcMetadata "google.golang.org/grpc/metadata"
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

func priceBigV3(t *testing.T, protoPrice *typesv3.Price) *big.Int {
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

func priceBigV2(t *testing.T, protoPrice *typesv2.Price) *big.Int {
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

func priceBigV1(t *testing.T, protoPrice *typesv1.Price) *big.Int {
	require.NotNil(t, protoPrice)
	var priceBig *big.Int
	var err error
	switch protoPrice.Currency.Currency.(type) {
	case *typesv1.Currency_IsoCurrency:
		priceBig, err = price.ToBigInt(protoPrice.Value, protoPrice.Decimals, price.ISODecimals)
	case *typesv1.Currency_NativeToken:
		priceBig, err = price.ToBigInt(protoPrice.Value, protoPrice.Decimals, price.NativeTokenDecimals)
	default:
		require.FailNow(t, "unexpected currency type in price")
		return nil
	}
	require.NoError(t, err)
	return priceBig
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

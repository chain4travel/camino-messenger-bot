// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package tests

import (
	"context"
	"math/big"
	"runtime"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/ethereum/go-ethereum/common"
	grpcMetadata "google.golang.org/grpc/metadata"
)

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

// Gets the current function name including the whole package path
func currentFuncName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	}
	return runtime.FuncForPC(pc).Name()
}

// func getPaymentTokenFromPriceV2(t *testing.T, price *typesv2.Price) common.Address {
// 	require.NotNil(t, price, "unexpected nil price")
// 	switch currency := price.GetCurrency().GetCurrency().(type) {
// 	case *typesv2.Currency_NativeToken:
// 		return booking.NativePaymentToken
// 	case *typesv2.Currency_IsoCurrency:
// 		return booking.ISOPaymentToken
// 	case *typesv2.Currency_TokenCurrency:
// 		return common.HexToAddress(currency.TokenCurrency.ContractAddress)
// 	}
// 	require.Fail(t, "unexpected currency type")
// 	return common.Address{}
// }

var (
	c4tFeeCutNominator   = big.NewInt(10) // 10% fee cut for C4T
	c4tFeeCutDenominator = big.NewInt(100)
)

func calculateCashIn(value *big.Int) (cashedIn *big.Int, c4tFeeCut *big.Int) { //nolint:unparam // c4tFeeCut is needed for logic clarity at least
	c4tFeeCut = big.NewInt(0).Mul(value, c4tFeeCutNominator)
	c4tFeeCut.Div(c4tFeeCut, c4tFeeCutDenominator)
	return big.NewInt(0).Sub(value, c4tFeeCut), c4tFeeCut
}

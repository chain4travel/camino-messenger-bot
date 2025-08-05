// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/conversion"
)

func (p *UnifiedPrice) ExtractCurrencyV2() *typesv2.Currency {
	switch {
	case p.IsNative:
		return &typesv2.Currency{
			Currency: &typesv2.Currency_NativeToken{},
		}
	case p.IsoCurrencyEnum != 0:
		return &typesv2.Currency{
			Currency: &typesv2.Currency_IsoCurrency{
				IsoCurrency: typesv2.IsoCurrency(p.IsoCurrencyEnum),
			},
		}
	case p.TokenContractAddress != "":
		return &typesv2.Currency{
			Currency: &typesv2.Currency_TokenCurrency{
				TokenCurrency: &typesv2.TokenCurrency{
					ContractAddress: p.TokenContractAddress,
				},
			},
		}
	}
	return nil
}

func (p *UnifiedPrice) ToPriceV2() *typesv2.Price {
	currency := p.ExtractCurrencyV2()
	return &typesv2.Price{
		Value:    p.Price,
		Decimals: conversion.MustUInt32ToInt32(p.Decimals),
		Currency: currency,
	}
}

func PriceV2ToUnifiedPrice(price *typesv2.Price) *UnifiedPrice {
	out := &UnifiedPrice{
		Price:    price.Value,
		Decimals: conversion.MustInt32ToUInt32(price.Decimals),
	}
	switch currency := price.Currency.Currency.(type) {
	case *typesv2.Currency_NativeToken:
		out.IsNative = true
	case *typesv2.Currency_IsoCurrency:
		out.IsoCurrencyEnum = int32(currency.IsoCurrency)
	case *typesv2.Currency_TokenCurrency:
		out.TokenContractAddress = currency.TokenCurrency.ContractAddress
	}
	return out
}

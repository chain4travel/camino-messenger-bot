// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"

func ExtractCurrencyV1FromUnifiedPrice(uPrice *UnifiedPrice) (*typesv1.Currency, bool) {
	currency := &typesv1.Currency{}

	switch {
	case uPrice.IsNative:
		currency.Currency = &typesv1.Currency_NativeToken{}
	case uPrice.IsoCurrencyEnum != 0:
		currency.Currency = &typesv1.Currency_IsoCurrency{
			IsoCurrency: typesv1.IsoCurrency(uPrice.IsoCurrencyEnum),
		}
	case uPrice.TokenContractAddress != "":
		currency.Currency = &typesv1.Currency_TokenCurrency{
			TokenCurrency: &typesv1.TokenCurrency{
				ContractAddress: uPrice.TokenContractAddress,
			},
		}
	default:
		return nil, false
	}
	return currency, true
}

func UnifiedPriceToProtoPriceV1(uPrice *UnifiedPrice) (*typesv1.Price, bool) {
	currency, ok := ExtractCurrencyV1FromUnifiedPrice(uPrice)
	if !ok {
		return nil, false
	}

	return &typesv1.Price{
		Value:    uPrice.Price,
		Decimals: uPrice.Decimals,
		Currency: currency,
	}, true
}

func ProtoPriceV1ToUnifiedPrice(price *typesv1.Price) (*UnifiedPrice, bool) {
	out := &UnifiedPrice{}
	out.Price = price.Value
	out.Decimals = price.Decimals

	switch currency := price.Currency.Currency.(type) {
	case *typesv1.Currency_NativeToken:
		out.IsNative = true
	case *typesv1.Currency_IsoCurrency:
		out.IsoCurrencyEnum = int32(currency.IsoCurrency)
	case *typesv1.Currency_TokenCurrency:
		out.TokenContractAddress = currency.TokenCurrency.ContractAddress
	default:
		return nil, false
	}
	return out, true
}

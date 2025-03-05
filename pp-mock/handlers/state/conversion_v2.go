// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"

func ExtractCurrencyV2FromUnifiedPrice(uPrice *UnifiedPrice) (*typesv2.Currency, bool) {
	currency := &typesv2.Currency{}

	switch {
	case uPrice.IsNative:
		currency.Currency = &typesv2.Currency_NativeToken{}
	case uPrice.IsoCurrencyEnum != 0:
		currency.Currency = &typesv2.Currency_IsoCurrency{
			IsoCurrency: typesv2.IsoCurrency(uPrice.IsoCurrencyEnum),
		}
	case uPrice.TokenContractAddress != "":
		currency.Currency = &typesv2.Currency_TokenCurrency{
			TokenCurrency: &typesv2.TokenCurrency{
				ContractAddress: uPrice.TokenContractAddress,
			},
		}
	default:
		return nil, false
	}
	return currency, true
}

func UnifiedPriceToPriceV2(uPrice *UnifiedPrice) (*typesv2.Price, bool) {
	currency, ok := ExtractCurrencyV2FromUnifiedPrice(uPrice)
	if !ok {
		return nil, false
	}

	return &typesv2.Price{
		Value:    uPrice.Price,
		Decimals: uPrice.Decimals,
		Currency: currency,
	}, true
}

func ProtoPriceV2ToUnifiedPrice(price *typesv2.Price) (*UnifiedPrice, bool) {
	out := &UnifiedPrice{}
	out.Price = price.Value
	out.Decimals = price.Decimals

	switch currency := price.Currency.Currency.(type) {
	case *typesv2.Currency_NativeToken:
		out.IsNative = true
	case *typesv2.Currency_IsoCurrency:
		out.IsoCurrencyEnum = int32(currency.IsoCurrency)
	case *typesv2.Currency_TokenCurrency:
		out.TokenContractAddress = currency.TokenCurrency.ContractAddress
	default:
		return nil, false
	}
	return out, true
}

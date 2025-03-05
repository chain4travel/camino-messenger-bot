// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"

func ExtractCurrencyV3FromUnifiedPrice(uPrice *UnifiedPrice) (*typesv3.Currency, bool) {
	currency := &typesv3.Currency{}

	switch {
	case uPrice.IsNative:
		currency.Currency = &typesv3.Currency_NativeToken{}
	case uPrice.IsoCurrencyEnum != 0:
		currency.Currency = &typesv3.Currency_IsoCurrency{
			IsoCurrency: typesv3.IsoCurrency(uPrice.IsoCurrencyEnum),
		}
	case uPrice.TokenContractAddress != "":
		currency.Currency = &typesv3.Currency_TokenCurrency{
			TokenCurrency: &typesv3.TokenCurrency{
				ContractAddress: &typesv3.EVMAddress{
					Address: uPrice.TokenContractAddress,
				},
			},
		}
	default:
		return nil, false
	}
	return currency, true
}

func UnifiedPriceToPriceV3(uPrice *UnifiedPrice) (*typesv3.Price, bool) {
	currency, ok := ExtractCurrencyV3FromUnifiedPrice(uPrice)
	if !ok {
		return nil, false
	}

	return &typesv3.Price{
		Value:    uPrice.Price,
		Decimals: uPrice.Decimals,
		Currency: currency,
	}, true
}

func ProtoPriceV3ToUnifiedPrice(price *typesv3.Price) (*UnifiedPrice, bool) {
	out := &UnifiedPrice{}
	out.Price = price.Value
	out.Decimals = price.Decimals

	switch currency := price.Currency.Currency.(type) {
	case *typesv3.Currency_NativeToken:
		out.IsNative = true
	case *typesv3.Currency_IsoCurrency:
		out.IsoCurrencyEnum = int32(currency.IsoCurrency)
	case *typesv3.Currency_TokenCurrency:
		out.TokenContractAddress = currency.TokenCurrency.ContractAddress.Address
	default:
		return nil, false
	}
	return out, true
}

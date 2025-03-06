// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"

func (p *UnifiedPrice) ExtractCurrencyV1() *typesv1.Currency {
	switch {
	case p.IsNative:
		return &typesv1.Currency{
			Currency: &typesv1.Currency_NativeToken{},
		}
	case p.IsoCurrencyEnum != 0:
		return &typesv1.Currency{
			Currency: &typesv1.Currency_IsoCurrency{
				IsoCurrency: typesv1.IsoCurrency(p.IsoCurrencyEnum),
			},
		}
	case p.TokenContractAddress != "":
		return &typesv1.Currency{
			Currency: &typesv1.Currency_TokenCurrency{
				TokenCurrency: &typesv1.TokenCurrency{
					ContractAddress: p.TokenContractAddress,
				},
			},
		}
	}
	return nil
}

func (p *UnifiedPrice) ToPriceV1() *typesv1.Price {
	currency := p.ExtractCurrencyV1()
	return &typesv1.Price{
		Value:    p.Price,
		Decimals: p.Decimals,
		Currency: currency,
	}
}

func PriceV1ToUnifiedPrice(price *typesv1.Price) *UnifiedPrice {
	out := &UnifiedPrice{
		Price:    price.Value,
		Decimals: price.Decimals,
	}
	switch currency := price.Currency.Currency.(type) {
	case *typesv1.Currency_NativeToken:
		out.IsNative = true
	case *typesv1.Currency_IsoCurrency:
		out.IsoCurrencyEnum = int32(currency.IsoCurrency)
	case *typesv1.Currency_TokenCurrency:
		out.TokenContractAddress = currency.TokenCurrency.ContractAddress
	}
	return out
}

// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	typesv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v5"
)

func (p *UnifiedPrice) ExtractCurrencyV5() *typesv4.Currency {
	switch {
	case p.IsNative:
		return &typesv4.Currency{
			Currency: &typesv4.Currency_NativeToken{},
		}
	case p.IsoCurrencyEnum != 0:
		return &typesv4.Currency{
			Currency: &typesv4.Currency_IsoCurrency{
				IsoCurrency: typesv4.IsoCurrency(p.IsoCurrencyEnum),
			},
		}
	case p.TokenContractAddress != "":
		return &typesv4.Currency{
			Currency: &typesv4.Currency_TokenCurrency{
				TokenCurrency: &typesv4.EVMAddress{
					Address: p.TokenContractAddress,
				},
			},
		}
	}
	return nil
}

func (p *UnifiedPrice) ToPriceV5() *typesv5.Price {
	currency := p.ExtractCurrencyV5()
	return &typesv5.Price{
		Value:    p.Price,
		Decimals: p.Decimals,
		Currency: currency,
	}
}

func PriceV5ToUnifiedPrice(price *typesv5.Price) *UnifiedPrice {
	out := &UnifiedPrice{
		Price:    price.Value,
		Decimals: price.Decimals,
	}
	switch currency := price.Currency.Currency.(type) {
	case *typesv4.Currency_NativeToken:
		out.IsNative = true
	case *typesv4.Currency_IsoCurrency:
		out.IsoCurrencyEnum = int32(currency.IsoCurrency)
	case *typesv4.Currency_TokenCurrency:
		out.TokenContractAddress = currency.TokenCurrency.Address
	}
	return out
}

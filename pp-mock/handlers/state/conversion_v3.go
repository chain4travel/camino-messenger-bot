// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"

func (p *UnifiedPrice) ExtractCurrencyV3() *typesv3.Currency {
	switch {
	case p.IsNative:
		return &typesv3.Currency{
			Currency: &typesv3.Currency_NativeToken{},
		}
	case p.IsoCurrencyEnum != 0:
		return &typesv3.Currency{
			Currency: &typesv3.Currency_IsoCurrency{
				IsoCurrency: typesv3.IsoCurrency(p.IsoCurrencyEnum),
			},
		}
	case p.TokenContractAddress != "":
		return &typesv3.Currency{
			Currency: &typesv3.Currency_TokenCurrency{
				TokenCurrency: &typesv3.TokenCurrency{
					ContractAddress: &typesv3.EVMAddress{
						Address: p.TokenContractAddress,
					},
				},
			},
		}
	}
	return nil
}

func (p *UnifiedPrice) ToPriceV3() *typesv3.Price {
	currency := p.ExtractCurrencyV3()
	return &typesv3.Price{
		Value:    p.Price,
		Decimals: p.Decimals,
		Currency: currency,
	}
}

func PriceV3ToUnifiedPrice(price *typesv3.Price) *UnifiedPrice {
	out := &UnifiedPrice{
		Price:    price.Value,
		Decimals: price.Decimals,
	}
	switch currency := price.Currency.Currency.(type) {
	case *typesv3.Currency_NativeToken:
		out.IsNative = true
	case *typesv3.Currency_IsoCurrency:
		out.IsoCurrencyEnum = int32(currency.IsoCurrency)
	case *typesv3.Currency_TokenCurrency:
		out.TokenContractAddress = currency.TokenCurrency.ContractAddress.Address
	}
	return out
}

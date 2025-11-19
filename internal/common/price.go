// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"context"
	"fmt"
	"math/big"

	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/booking"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/conversion"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/erc20"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/price"
	"github.com/ethereum/go-ethereum/common"
)

var (
	_ PriceHandler = (*priceHandler)(nil)

	errMissingPrice    = fmt.Errorf("missing price")
	errUnknownCurrency = fmt.Errorf("unknown currency")
)

type PriceHandler interface {
	GetPriceAndTokenV2(
		ctx context.Context,
		priceV2 *typesv2.Price,
	) (
		priceBigInt *big.Int,
		paymentToken common.Address,
		isoCurrency *big.Int,
		err error,
	)

	GetPriceAndTokenV3(
		ctx context.Context,
		priceV3 *typesv3.Price,
	) (
		priceBigInt *big.Int,
		paymentToken common.Address,
		isoCurrency *big.Int,
		err error,
	)

	GetPriceAndTokenV4(
		ctx context.Context,
		priceV4 *typesv4.Price,
	) (
		priceBigInt *big.Int,
		paymentToken common.Address,
		isoCurrency *big.Int,
		err error,
	)
}

func NewPriceHandler(erc20 erc20.Service) PriceHandler {
	return &priceHandler{
		erc20: erc20,
	}
}

type priceHandler struct {
	erc20 erc20.Service
}

func (p *priceHandler) GetPriceAndTokenV2(
	ctx context.Context,
	priceV2 *typesv2.Price,
) (
	priceBigInt *big.Int,
	paymentToken common.Address,
	isoCurrency *big.Int,
	err error,
) {
	if priceV2 == nil {
		return nil, common.Address{}, nil, errMissingPrice
	}

	isoCurrency = big.NewInt(0)
	paymentToken = booking.NativePaymentToken

	switch currency := priceV2.Currency.GetCurrency().(type) {
	case *typesv2.Currency_NativeToken:
		priceBigInt, err = price.ToBigInt(priceV2.Value, priceV2.Decimals, price.NativeTokenDecimals)
	case *typesv2.Currency_TokenCurrency:
		contractAddress := common.HexToAddress(currency.TokenCurrency.ContractAddress)
		// if contract address is invalid in any way, Decimals() will return an error
		tokenDecimals, decErr := p.erc20.Decimals(ctx, contractAddress)
		if decErr != nil {
			return nil, common.Address{}, nil, fmt.Errorf("failed to fetch token decimals: %w", decErr)
		}

		priceBigInt, err = price.ToBigInt(priceV2.Value, priceV2.Decimals, tokenDecimals)
		paymentToken = contractAddress
	case *typesv2.Currency_IsoCurrency:
		priceBigInt, err = price.ToBigInt(priceV2.Value, priceV2.Decimals, price.ISODecimals)
		paymentToken = booking.ISOPaymentToken
		isoCurrency = big.NewInt(int64(currency.IsoCurrency))
	default:
		return nil, common.Address{}, nil, fmt.Errorf("%w (%T)", errUnknownCurrency, currency)
	}

	if err != nil {
		return nil, common.Address{}, nil, fmt.Errorf("failed to convert price to big.Int: %w", err)
	}

	return priceBigInt, paymentToken, isoCurrency, nil
}

func (p *priceHandler) GetPriceAndTokenV3(
	ctx context.Context,
	priceV3 *typesv3.Price,
) (
	priceBigInt *big.Int,
	paymentToken common.Address,
	isoCurrency *big.Int,
	err error,
) {
	if priceV3 == nil {
		return nil, common.Address{}, nil, errMissingPrice
	}

	isoCurrency = big.NewInt(0)
	paymentToken = booking.NativePaymentToken

	switch currency := priceV3.Currency.GetCurrency().(type) {
	case *typesv3.Currency_NativeToken:
		priceBigInt, err = price.ToBigInt(priceV3.Value, priceV3.Decimals, price.NativeTokenDecimals)
	case *typesv3.Currency_TokenCurrency:
		contractAddress := common.HexToAddress(currency.TokenCurrency.ContractAddress.Address)
		// if contract address is invalid in any way, Decimals() will return an error
		tokenDecimals, decErr := p.erc20.Decimals(ctx, contractAddress)
		if decErr != nil {
			return nil, common.Address{}, nil, fmt.Errorf("failed to fetch token decimals: %w", decErr)
		}

		priceBigInt, err = price.ToBigInt(priceV3.Value, priceV3.Decimals, tokenDecimals)
		paymentToken = contractAddress
	case *typesv3.Currency_IsoCurrency:
		priceBigInt, err = price.ToBigInt(priceV3.Value, priceV3.Decimals, price.ISODecimals)
		paymentToken = booking.ISOPaymentToken
		isoCurrency = big.NewInt(int64(currency.IsoCurrency))
	default:
		return nil, common.Address{}, nil, fmt.Errorf("%w (%T)", errUnknownCurrency, currency)
	}

	if err != nil {
		return nil, common.Address{}, nil, fmt.Errorf("failed to convert price to big.Int: %w", err)
	}

	return priceBigInt, paymentToken, isoCurrency, nil
}

func (p *priceHandler) GetPriceAndTokenV4(
	ctx context.Context,
	priceV4 *typesv4.Price,
) (
	priceBigInt *big.Int,
	paymentToken common.Address,
	isoCurrency *big.Int,
	err error,
) {
	if priceV4 == nil {
		return nil, common.Address{}, nil, errMissingPrice
	}

	isoCurrency = big.NewInt(0)
	paymentToken = booking.NativePaymentToken

	switch currency := priceV4.Currency.GetCurrency().(type) {
	case *typesv4.Currency_NativeToken:
		priceBigInt, err = price.ToBigInt(priceV4.Value, conversion.MustUInt32ToInt32(priceV4.Decimals), price.NativeTokenDecimals)
	case *typesv4.Currency_TokenCurrency:
		contractAddress := common.HexToAddress(currency.TokenCurrency.Address)
		// if contract address is invalid in any way, Decimals() will return an error
		tokenDecimals, decErr := p.erc20.Decimals(ctx, contractAddress)
		if decErr != nil {
			return nil, common.Address{}, nil, fmt.Errorf("failed to fetch token decimals: %w", decErr)
		}

		priceBigInt, err = price.ToBigInt(priceV4.Value, conversion.MustUInt32ToInt32(priceV4.Decimals), tokenDecimals)
		paymentToken = contractAddress
	case *typesv4.Currency_IsoCurrency:
		priceBigInt, err = price.ToBigInt(priceV4.Value, conversion.MustUInt32ToInt32(priceV4.Decimals), price.ISODecimals)
		paymentToken = booking.ISOPaymentToken
		isoCurrency = big.NewInt(int64(currency.IsoCurrency))
	default:
		return nil, common.Address{}, nil, fmt.Errorf("%w (%T)", errUnknownCurrency, currency)
	}

	if err != nil {
		return nil, common.Address{}, nil, fmt.Errorf("failed to convert price to big.Int: %w", err)
	}

	return priceBigInt, paymentToken, isoCurrency, nil
}

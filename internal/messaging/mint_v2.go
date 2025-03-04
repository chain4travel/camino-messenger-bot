// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"fmt"
	"math/big"
	"time"

	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"

	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	"github.com/chain4travel/camino-messenger-bot/pkg/price"
	"github.com/ethereum/go-ethereum/common"
)

func (h *evmResponseHandler) prepareMintResponseV2(
	ctx context.Context,
	response *bookv2.MintResponse,
	request *bookv2.MintRequest,
) {
	if response.Header.Status != typesv1.StatusType_STATUS_TYPE_SUCCESS {
		return
	}
	// TODO: @VjeraTurk check if CMAccount exists
	// TODO if address is invalid and will just get zero addr
	if !common.IsHexAddress(request.BuyerAddress) {
		errMsg := fmt.Sprintf("Invalid BuyerAddress: %s", request.BuyerAddress)
		h.logger.Error(errMsg)
		h.AddErrorToResponseHeader(response, errMsg)
		return
	}

	buyerAddress := common.HexToAddress(request.BuyerAddress)

	if response.BookingTokenUri == "" {
		jsonPlain, tokenURI, err := createTokenURIforMintResponse(
			response.MintId.Value,
			request.BookingReference,
		)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to mint token: failed to generate tokenURI:  %s", err)
			h.logger.Error(errMsg)
			h.AddErrorToResponseHeader(response, errMsg)
			return
		}
		h.logger.Debugf("Token URI JSON: %s\n", jsonPlain)
		response.BookingTokenUri = tokenURI
	}
	h.logger.Debugf("Token URI: %s\n", response.BookingTokenUri)

	buyableUntil, err := verifyAndFixBuyableUntil(response.BuyableUntil, time.Now())
	if err != nil {
		h.logger.Error(err)
		h.AddErrorToResponseHeader(response, err.Error())
		return
	}
	response.BuyableUntil = buyableUntil

	price, paymentToken, isoCurrency, err := h.getPriceAndTokenV2(ctx, response.Price)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(response, errMessage)
		return
	}

	// MINT TOKEN
	txID, tokenID, err := h.mint(
		ctx,
		buyerAddress,
		response.BookingTokenUri,
		big.NewInt(response.BuyableUntil.Seconds),
		price,
		paymentToken,
		isoCurrency,
	)
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(response, errMessage)
		return
	}

	h.logger.Infof("NFT minted with txID: %s\n", txID)

	h.onBookingTokenMint(tokenID, response.MintId, response.BuyableUntil.AsTime())

	response.Header.Status = typesv1.StatusType_STATUS_TYPE_SUCCESS
	response.BookingTokenId = tokenID.Uint64()
	response.MintTransactionId = txID
}

func (h *evmResponseHandler) processMintResponseV2(ctx context.Context, response *bookv2.MintResponse) {
	if response.MintTransactionId == "" {
		h.logger.Error(errMissingMintTxID)
		h.AddErrorToResponseHeader(response, errMissingMintTxID.Error())
		return
	}

	tokenID := new(big.Int).SetUint64(response.BookingTokenId)

	price, paymentToken, _, err := h.getPriceAndTokenV2(ctx, response.Price)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(response, errMessage)
		return
	}

	receipt, err := h.bookingService.BuyBookingToken(ctx, tokenID, price, paymentToken)
	if err != nil {
		errMessage := fmt.Sprintf("error buying NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(response, errMessage)
		return
	}

	h.logger.Infof("Bought NFT (txID=%s) with ID: %s\n", receipt, response.MintTransactionId)
	response.BuyTransactionId = receipt.TxHash.Hex()
}

func (h *evmResponseHandler) getPriceAndTokenV2(ctx context.Context, priceV2 *typesv2.Price) (*big.Int, common.Address, *big.Int, error) {
	if priceV2 == nil {
		return nil, common.Address{}, nil, errMissingPrice
	}

	var priceBigInt *big.Int
	isoCurrency := big.NewInt(0)
	paymentToken := booking.NativePaymentToken
	var err error

	switch currency := priceV2.Currency.GetCurrency().(type) {
	case *typesv2.Currency_NativeToken:
		priceBigInt, err = price.ToBigInt(priceV2.Value, priceV2.Decimals, price.NativeTokenDecimals)
	case *typesv2.Currency_TokenCurrency:
		contractAddress := common.HexToAddress(currency.TokenCurrency.ContractAddress)
		// if contract address is invalid in any way, Decimals() will return an error
		tokenDecimals, decErr := h.erc20.Decimals(ctx, contractAddress)
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

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"fmt"
	"math/big"
	"time"

	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"

	"github.com/ethereum/go-ethereum/common"
)

func (h *evmResponseHandler) prepareMintResponseV1(
	ctx context.Context,
	response *bookv1.MintResponse,
	request *bookv1.MintRequest,
) {
	if response.Header.Status != typesv1.StatusType_STATUS_TYPE_SUCCESS {
		return
	}

	if !common.IsHexAddress(request.BuyerAddress) {
		errMsg := fmt.Sprintf("Invalid BuyerAddress: %s", request.BuyerAddress)
		h.logger.Error(errMsg)
		h.responseHeaderHandler.AddError(response, errMsg)
		return
	}
	buyerAddress := common.HexToAddress(request.BuyerAddress)

	// Get a Token URI for the token.
	jsonPlain, tokenURI, err := createTokenURIforMintResponse(
		response.MintId.Value,
		request.BookingReference,
	)
	if err != nil {
		errMsg := fmt.Sprintf("error creating token URI: %v", err)
		h.logger.Debugf(errMsg) // TODO: @VjeraTurk change to Error after we stop using mocked uri data
		h.responseHeaderHandler.AddError(response, errMsg)
		return
	}

	h.logger.Debugf("Token URI JSON: %s", jsonPlain)

	buyableUntil, err := h.verifyAndFixBuyableUntil(response.BuyableUntil, time.Now())
	if err != nil {
		h.logger.Error(err)
		h.responseHeaderHandler.AddError(response, err.Error())
		return
	}
	response.BuyableUntil = buyableUntil

	price, paymentToken, isoCurrency, err := h.priceHandler.GetPriceAndTokenV1(ctx, response.Price)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Errorf(errMessage)
		h.responseHeaderHandler.AddError(response, errMessage)
		return
	}

	receipt, tokenID, err := h.bookingService.MintBookingToken(
		ctx,
		buyerAddress,
		tokenURI,
		big.NewInt(response.BuyableUntil.Seconds),
		price,
		paymentToken,
		isoCurrency,
	)
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.responseHeaderHandler.AddError(response, errMessage)
		return
	}
	txID := receipt.TxHash.Hex()

	h.logger.Infof("NFT minted with txID: %s", txID)

	h.subscribeForTokenBoughtEvent(ctx, tokenID, response.MintId.Value, buyableUntil)

	// TODO @evlekht pp will not know if we failed to mint or setup notification

	response.Header.Status = typesv1.StatusType_STATUS_TYPE_SUCCESS
	response.BookingToken = &typesv1.BookingToken{TokenId: int32(tokenID.Int64())} //nolint:gosec
	response.MintTransactionId = txID
}

func (h *evmResponseHandler) processMintResponseV1(ctx context.Context, response *bookv1.MintResponse) {
	if response.MintTransactionId == "" {
		h.logger.Error(errMissingMintTxID)
		h.responseHeaderHandler.AddError(response, errMissingMintTxID.Error())
		return
	}

	value64 := uint64(response.BookingToken.TokenId)
	tokenID := new(big.Int).SetUint64(value64)

	price, paymentToken, _, err := h.priceHandler.GetPriceAndTokenV1(ctx, response.Price)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Errorf(errMessage)
		h.responseHeaderHandler.AddError(response, errMessage)
		return
	}

	receipt, err := h.bookingService.BuyBookingToken(ctx, tokenID, price, paymentToken)
	if err != nil {
		errMessage := fmt.Sprintf("error buying NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.responseHeaderHandler.AddError(response, errMessage)
		return
	}

	response.BuyTransactionId = receipt.TxHash.Hex()
	h.logger.Infof("Bought NFT: buy-tx %s, mint-tx %s", response.BuyTransactionId, response.MintTransactionId)
}

// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"fmt"
	"math/big"
	"time"

	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"

	"github.com/chain4travel/camino-messenger-bot/v12/internal/version"
	"github.com/ethereum/go-ethereum/common"
)

func (h *evmResponseHandler) prepareMintResponseV2(
	ctx context.Context,
	request *bookv2.MintRequest,
	response *bookv2.MintResponse,
) *bookv2.MintResponse {
	if response.Header.Status != typesv1.StatusType_STATUS_TYPE_SUCCESS {
		return response
	}

	if !common.IsHexAddress(request.BuyerAddress) {
		errMsg := fmt.Sprintf("Invalid BuyerAddress: %s", request.BuyerAddress)
		h.logger.Error(errMsg)
		return mintErrResponseV2(errMsg)
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
			return mintErrResponseV2(errMsg)
		}
		h.logger.Debugf("Token URI JSON: %s", jsonPlain)
		response.BookingTokenUri = tokenURI
	}
	h.logger.Debugf("Token URI: %s", response.BookingTokenUri)

	buyableUntil, err := h.verifyAndFixBuyableUntil(response.BuyableUntil, time.Now())
	if err != nil {
		h.logger.Error(err)
		return mintErrResponseV2(err.Error())
	}
	response.BuyableUntil = buyableUntil

	price, paymentToken, isoCurrency, err := h.priceHandler.GetPriceAndTokenV2(ctx, response.Price)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Errorf(errMessage)
		return mintErrResponseV2(errMessage)
	}

	receipt, tokenID, err := h.bookingService.MintBookingToken(
		ctx,
		buyerAddress,
		response.BookingTokenUri,
		big.NewInt(response.BuyableUntil.Seconds),
		price,
		paymentToken,
		isoCurrency,
		false,
	)
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		h.logger.Errorf(errMessage)
		return mintErrResponseV2(errMessage)
	}
	txID := receipt.TxHash.Hex()

	h.logger.Infof("NFT minted with txID: %s", txID)

	h.subscribeForTokenBoughtEvent(ctx, tokenID, response.MintId.Value, buyableUntil)

	// TODO @evlekht pp will not know if we failed to mint or setup notification

	response.Header.Status = typesv1.StatusType_STATUS_TYPE_SUCCESS
	response.BookingTokenId = tokenID.Uint64()
	response.MintTransactionId = txID

	return response
}

func (h *evmResponseHandler) processMintResponseV2(
	ctx context.Context,
	response *bookv2.MintResponse,
) *bookv2.MintResponse {
	if response.Header.Status == typesv1.StatusType_STATUS_TYPE_FAILURE {
		return response
	}

	if response.MintTransactionId == "" {
		h.logger.Error(errMissingMintTxID)
		return mintErrResponseV2(errMissingMintTxID.Error())
	}

	tokenID := new(big.Int).SetUint64(response.BookingTokenId)

	price, paymentToken, _, err := h.priceHandler.GetPriceAndTokenV2(ctx, response.Price)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Errorf(errMessage)
		return mintErrResponseV2(errMessage)
	}

	receipt, err := h.bookingService.BuyBookingToken(ctx, tokenID, price, paymentToken)
	if err != nil {
		errMessage := fmt.Sprintf("error buying NFT: %v", err)
		h.logger.Errorf(errMessage)
		return mintErrResponseV2(errMessage)
	}

	response.BuyTransactionId = receipt.TxHash.Hex()
	h.logger.Infof("Bought NFT: buy-tx %s, mint-tx %s", response.BuyTransactionId, response.MintTransactionId)

	return response
}

func mintErrResponseV2(errMessage string) *bookv2.MintResponse {
	return &bookv2.MintResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_FAILURE,
			Alerts: []*typesv1.Alert{{
				Message: errMessage,
				Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
			}},
		},
	}
}

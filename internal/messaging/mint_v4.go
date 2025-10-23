// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"fmt"
	"math/big"
	"time"

	bookv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/ethereum/go-ethereum/common"
)

func (h *evmResponseHandler) prepareMintResponseV4(
	ctx context.Context,
	response *bookv4.MintResponse,
	request *bookv4.MintRequest,
) {
	if response.Header.Status != typesv4.StatusType_STATUS_TYPE_SUCCESS {
		return
	}

	if !common.IsHexAddress(request.BuyerAddress.Address) {
		errMsg := fmt.Sprintf("Invalid BuyerAddress: %s", request.BuyerAddress.Address)
		h.logger.Error(errMsg)
		h.responseHeaderHandler.AddError(response, errMsg)
		return
	}
	buyerAddress := common.HexToAddress(request.BuyerAddress.Address)

	h.logger.Debugf("Token URI: %s", response.BookingTokenUri)

	buyableUntil, err := h.verifyAndFixBuyableUntil(response.BuyableUntil, time.Now())
	if err != nil {
		h.logger.Error(err)
		h.responseHeaderHandler.AddError(response, err.Error())
		return
	}
	response.BuyableUntil = buyableUntil

	price, paymentToken, isoCurrency, err := h.priceHandler.GetPriceAndTokenV4(ctx, response.Price)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Errorf(errMessage)
		h.responseHeaderHandler.AddError(response, errMessage)
		return
	}

	receipt, tokenID, err := h.bookingService.MintBookingToken(
		ctx,
		buyerAddress,
		response.BookingTokenUri,
		big.NewInt(response.BuyableUntil.Seconds),
		price,
		paymentToken,
		isoCurrency,
		response.Cancellable,
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

	response.Header.Status = typesv4.StatusType_STATUS_TYPE_SUCCESS
	response.BookingTokenId = tokenID.Uint64()
	response.MintTransactionId = &typesv4.EVMTransactionID{
		Hash: txID,
	}
}

func (h *evmResponseHandler) processMintResponseV4(ctx context.Context, response *bookv4.MintResponse) {
	if response.MintTransactionId == nil || response.MintTransactionId.Hash == "" {
		h.logger.Error(errMissingMintTxID)
		h.responseHeaderHandler.AddError(response, errMissingMintTxID.Error())
		return
	}

	tokenID := new(big.Int).SetUint64(response.BookingTokenId)
	price, paymentToken, _, err := h.priceHandler.GetPriceAndTokenV4(ctx, response.Price)
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

	response.BuyTransactionId = &typesv4.EVMTransactionID{
		Hash: receipt.TxHash.Hex(),
	}

	h.logger.Infof("Bought NFT: buy-tx %s, mint-tx %s", response.BuyTransactionId.Hash, response.MintTransactionId.Hash)

	if response.Cancellable {
		if err := h.eventListener.SubscribeCancellationEvents(ctx, tokenID); err != nil {
			err := fmt.Errorf("error subscribing for cancellation events as distributor (tokenID: %d, mintID: %s): %w", tokenID.Int64(), response.MintId.Value, err)
			h.logger.Error(err)
			response.Header.Alerts = append(response.Header.Alerts, &typesv4.Alert{
				Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
				Message: err.Error(),
			})
		}
		h.logger.Infof("Subscribed for cancellation events as distributor (tokenID: %s)", tokenID.String())
	}
}

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
	"google.golang.org/protobuf/proto"
)

func (h *evmResponseHandler) prepareMintResponseV4(
	ctx context.Context,
	request *bookv4.MintRequest,
	response *bookv4.MintResponse,
) {
	successResp := response.GetSuccessResponse()
	if successResp == nil {
		return
	}

	h.logger.Debugf("Token URI: %s", successResp.BookingTokenUri)

	buyableUntil, err := h.verifyAndFixBuyableUntil(successResp.BuyableUntil, time.Now())
	if err != nil {
		h.logger.Error(err)
		h.responseHeaderHandler.AddError(response, err.Error())
		return
	}
	successResp.BuyableUntil = buyableUntil

	price, paymentToken, isoCurrency, err := h.priceHandler.GetPriceAndTokenV4(ctx, successResp.Price)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Errorf(errMessage)
		h.responseHeaderHandler.AddError(response, errMessage)
		return
	}

	receipt, tokenID, err := h.bookingService.MintBookingToken(
		ctx,
		common.HexToAddress(request.BuyerAddress.Address),
		successResp.BookingTokenUri,
		big.NewInt(successResp.BuyableUntil.Seconds),
		price,
		paymentToken,
		isoCurrency,
		successResp.Cancellable,
	)
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.responseHeaderHandler.AddError(response, errMessage)
		return
	}
	txID := receipt.TxHash.Hex()

	h.logger.Infof("NFT minted with txID: %s", txID)

	h.subscribeForTokenBoughtEvent(ctx, tokenID, successResp.MintId.Value, buyableUntil)

	// TODO @evlekht pp will not know if we failed to mint or setup notification

	successResp.BookingTokenId = tokenID.Uint64()
	successResp.MintTransactionId = &typesv4.EVMTransactionID{Hash: txID}
}

func (h *evmResponseHandler) processMintResponseV4(
	ctx context.Context,
	request *bookv4.MintRequest,
	response *bookv4.MintResponse,
) {
	successResp := response.GetSuccessResponse()
	if successResp == nil {
		return
	}

	if successResp.MintTransactionId == nil {
		h.logger.Error(errMissingMintTxID)
		h.responseHeaderHandler.AddError(response, errMissingMintTxID.Error())
		return
	}

	if !proto.Equal(request.ExpectedPrice, successResp.Price) {
		errMessage := "expected price does not match the mint response price"
		h.logger.Error(errMessage)
		h.responseHeaderHandler.AddError(response, errMessage)
		return
	}

	tokenID := new(big.Int).SetUint64(successResp.BookingTokenId)
	price, paymentToken, _, err := h.priceHandler.GetPriceAndTokenV4(ctx, successResp.Price)
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

	successResp.BuyTransactionId = &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	h.logger.Infof("Bought NFT: buy-tx %s, mint-tx %s", successResp.BuyTransactionId.Hash, successResp.MintTransactionId.Hash)

	if successResp.Cancellable {
		if err := h.eventListener.SubscribeCancellationEvents(ctx, tokenID); err != nil {
			err := fmt.Errorf("error subscribing for cancellation events as distributor (tokenID: %d, mintID: %s): %w", tokenID.Int64(), successResp.MintId.Value, err)
			h.logger.Error(err)
			successResp.Header.Alerts = append(successResp.Header.Alerts, &typesv4.Alert{
				Code:    typesv4.AlertCode_ALERT_CODE_INFORMATIONAL, // TODO@ // TODO@ code in pp mock
				Message: err.Error(),
			})
		}
		h.logger.Infof("Subscribed for cancellation events as distributor (tokenID: %s)", tokenID.String())
	}
}

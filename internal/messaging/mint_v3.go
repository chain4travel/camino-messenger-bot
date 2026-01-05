// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"fmt"
	"math/big"
	"time"

	bookv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/version"
	"github.com/ethereum/go-ethereum/common"
)

func (h *evmResponseHandler) prepareMintResponseV3(
	ctx context.Context,
	request *bookv3.MintRequest,
	response *bookv3.MintResponse,
) *bookv3.MintResponse {
	if response.Header.Status != typesv1.StatusType_STATUS_TYPE_SUCCESS {
		return response
	}

	if !common.IsHexAddress(request.BuyerAddress.Address) {
		errMsg := fmt.Sprintf("Invalid BuyerAddress: %s", request.BuyerAddress.Address)
		h.logger.Error(errMsg)
		return mintErrResponseV3(errMsg)
	}
	buyerAddress := common.HexToAddress(request.BuyerAddress.Address)

	if response.BookingTokenUri == "" {
		jsonPlain, tokenURI, err := createTokenURIforMintResponse(
			response.MintId.Value,
			request.BookingReference,
		)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to mint token: failed to generate tokenURI:  %s", err)
			h.logger.Error(errMsg)
			return mintErrResponseV3(errMsg)
		}
		h.logger.Debugf("Token URI JSON: %s", jsonPlain)
		response.BookingTokenUri = tokenURI
	}
	h.logger.Debugf("Token URI: %s", response.BookingTokenUri)

	buyableUntil, err := h.verifyAndFixBuyableUntil(response.BuyableUntil, time.Now())
	if err != nil {
		h.logger.Error(err)
		return mintErrResponseV3(err.Error())
	}
	response.BuyableUntil = buyableUntil

	price, paymentToken, isoCurrency, err := h.priceHandler.GetPriceAndTokenV3(ctx, response.Price)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Errorf(errMessage)
		return mintErrResponseV3(errMessage)
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
		return mintErrResponseV3(errMessage)
	}
	txID := receipt.TxHash.Hex()

	h.logger.Infof("NFT minted with txID: %s", txID)

	h.subscribeForTokenBoughtEvent(ctx, tokenID, response.MintId.Value, buyableUntil)

	// TODO @evlekht pp will not know if we failed to mint or setup notification

	response.Header.Status = typesv1.StatusType_STATUS_TYPE_SUCCESS
	response.BookingTokenId = tokenID.Uint64()
	response.MintTransactionId = &typesv3.EVMTransactionID{
		Hash: txID,
	}

	return response
}

func (h *evmResponseHandler) processMintResponseV3(
	ctx context.Context,
	response *bookv3.MintResponse,
) *bookv3.MintResponse {
	if response.Header.Status == typesv1.StatusType_STATUS_TYPE_FAILURE {
		return response
	}

	if response.MintTransactionId == nil || response.MintTransactionId.Hash == "" {
		h.logger.Error(errMissingMintTxID)
		return mintErrResponseV3(errMissingMintTxID.Error())
	}

	tokenID := new(big.Int).SetUint64(response.BookingTokenId)
	price, paymentToken, _, err := h.priceHandler.GetPriceAndTokenV3(ctx, response.Price)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Errorf(errMessage)
		return mintErrResponseV3(errMessage)
	}

	receipt, err := h.bookingService.BuyBookingToken(ctx, tokenID, price, paymentToken)
	if err != nil {
		errMessage := fmt.Sprintf("error buying NFT: %v", err)
		h.logger.Errorf(errMessage)
		return mintErrResponseV3(errMessage)
	}

	response.BuyTransactionId = &typesv3.EVMTransactionID{
		Hash: receipt.TxHash.Hex(),
	}

	h.logger.Infof("Bought NFT: buy-tx %s, mint-tx %s", response.BuyTransactionId.Hash, response.MintTransactionId.Hash)

	if response.Cancellable {
		if err := h.eventListener.SubscribeCancellationEvents(ctx, tokenID); err != nil {
			err := fmt.Errorf("error subscribing for cancellation events as distributor (tokenID: %d, mintID: %s): %w", tokenID.Int64(), response.MintId.Value, err)
			h.logger.Error(err)
			response.Header.Alerts = append(response.Header.Alerts, &typesv1.Alert{
				Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				Message: err.Error(),
			})
		}
		h.logger.Infof("Subscribed for cancellation events as distributor (tokenID: %s)", tokenID.String())
	}

	return response
}

func mintErrResponseV3(errMessage string) *bookv3.MintResponse {
	return &bookv3.MintResponse{
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

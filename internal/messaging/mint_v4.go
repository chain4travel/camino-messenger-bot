// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"fmt"
	"math/big"
	"time"

	bookv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v4"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v13/internal/version"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/proto"
)

func (h *evmResponseHandler) prepareMintResponseV4(
	ctx context.Context,
	request *bookv4.MintRequest,
	response *bookv4.MintResponse,
) *bookv4.MintResponse {
	successResp := response.GetSuccessResponse()
	if successResp == nil {
		return response
	}

	h.logger.Debugf("Token URI: %s", successResp.BookingTokenUri)

	buyableUntil, err := h.verifyAndFixBuyableUntil(successResp.BuyableUntil, time.Now())
	if err != nil {
		h.logger.Debug(err)
		return mintErrResponseV4(typesv4.ErrorCode_ERROR_CODE_INTERNAL, err.Error())
	}
	successResp.BuyableUntil = buyableUntil

	price, paymentToken, isoCurrency, err := h.priceHandler.GetPriceAndTokenV4(ctx, successResp.Price)
	if err != nil {
		errMsg := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Error(errMsg)
		return mintErrResponseV4(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMsg)
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
		errMsg := fmt.Sprintf("error minting booking token: %v", err)
		h.logger.Error(errMsg)
		return mintErrResponseV4(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMsg)
	}

	txID := receipt.TxHash.Hex()

	h.logger.Infof("NFT minted with txID: %s", txID)

	h.subscribeForTokenBoughtEvent(ctx, tokenID, successResp.MintId.Value, buyableUntil)

	// TODO @evlekht pp will not know if we failed to mint or setup notification

	successResp.BookingTokenId = tokenID.Uint64()
	successResp.MintTransactionId = &typesv4.EVMTransactionID{Hash: txID}

	return response
}

func (h *evmResponseHandler) processMintResponseV4(
	ctx context.Context,
	request *bookv4.MintRequest,
	response *bookv4.MintResponse,
) *bookv4.MintResponse {
	successResp := response.GetSuccessResponse()
	if successResp == nil {
		return response
	}

	if successResp.MintTransactionId == nil {
		h.logger.Debug(errMissingMintTxID)
		return mintErrResponseV4(typesv4.ErrorCode_ERROR_CODE_BUSINESS_PROCESS_ERROR, errMissingMintTxID.Error())
	}

	if !proto.Equal(request.ExpectedPrice, successResp.Price) {
		h.logger.Debug(errUnexpectedMintResponsePrice)
		return mintErrResponseV4(typesv4.ErrorCode_ERROR_CODE_BUSINESS_PROCESS_ERROR, errUnexpectedMintResponsePrice.Error())
	}

	tokenID := new(big.Int).SetUint64(successResp.BookingTokenId)
	price, paymentToken, _, err := h.priceHandler.GetPriceAndTokenV4(ctx, successResp.Price)
	if err != nil {
		errMsg := fmt.Sprintf("error getting price and payment token: %v", err)
		h.logger.Error(errMsg)
		return mintErrResponseV4(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMsg)
	}

	receipt, err := h.bookingService.BuyBookingToken(ctx, tokenID, price, paymentToken)
	if err != nil {
		errMsg := fmt.Sprintf("error buying booking token: %v", err)
		h.logger.Error(errMsg)
		return mintErrResponseV4(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMsg)
	}

	successResp.BuyTransactionId = &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	h.logger.Infof("Bought NFT: buy-tx %s, mint-tx %s", successResp.BuyTransactionId.Hash, successResp.MintTransactionId.Hash)

	if successResp.Cancellable {
		if err := h.eventListener.SubscribeCancellationEvents(ctx, tokenID); err != nil {
			err := fmt.Errorf("error subscribing for cancellation events as distributor (tokenID: %d, mintID: %s): %w", tokenID.Int64(), successResp.MintId.Value, err)
			h.logger.Error(err)
			successResp.Header.Alerts = append(successResp.Header.Alerts, &typesv4.Alert{
				Code:    typesv4.AlertCode_ALERT_CODE_INFORMATIONAL,
				Message: err.Error(),
			})
		}
		h.logger.Infof("Subscribed for cancellation events as distributor (tokenID: %s)", tokenID.String())
	}

	return response
}

func mintErrResponseV4(code typesv4.ErrorCode, errMessage string) *bookv4.MintResponse {
	return &bookv4.MintResponse{
		Response: &bookv4.MintResponse_ErrorResponse{
			ErrorResponse: &bookv4.MintErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors: []*typesv4.Error{{
						Code:    code,
						Message: errMessage,
					}},
				},
			},
		},
	}
}

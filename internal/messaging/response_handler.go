/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	typesv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1alpha"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/chain4travel/camino-messenger-bot/internal/tvm"
	"github.com/chain4travel/caminotravelvm/actions"
	"github.com/chain4travel/caminotravelvm/consts"
	"go.uber.org/zap"
)

var _ ResponseHandler = (*TvmResponseHandler)(nil)

type ResponseHandler interface {
	HandleResponse(ctx context.Context, msgType MessageType, request *RequestContent, response *ResponseContent)
}
type TvmResponseHandler struct {
	tvmClient *tvm.Client
	logger    *zap.SugaredLogger
}

func (h *TvmResponseHandler) HandleResponse(ctx context.Context, msgType MessageType, request *RequestContent, response *ResponseContent) {
	switch msgType {
	case MintRequest: // distributor will post-process a mint request to buy the returned NFT
		if h.handleMintRequest(ctx, response) {
			return
		}
	case MintResponse: // provider will act upon receiving a mint response by minting an NFT
		if h.handleMintResponse(ctx, response, request) {
			return
		}
	}
}

func (h *TvmResponseHandler) handleMintResponse(ctx context.Context, response *ResponseContent, request *RequestContent) bool {
	owner := h.tvmClient.Address()
	if response.MintResponse.Header == nil {
		response.MintResponse.Header = &typesv1alpha.ResponseHeader{}
	}
	buyer, err := codec.ParseAddressBech32(consts.HRP, request.MintRequest.BuyerAddress)
	if err != nil {
		addErrorToResponseHeader(response, fmt.Sprintf("error parsing buyer address: %v", err))
		return true
	}
	price, err := strconv.Atoi(response.MintResponse.Price.Value)
	if err != nil {
		addErrorToResponseHeader(response, fmt.Sprintf("error parsing price value: %v", err))
		return true
	}
	success, txID, err := h.tvmClient.SendTxAndWait(ctx, createNFTAction(owner, buyer, uint64(response.MintResponse.BuyableUntil.Seconds), uint64(price), response.MintResponse.MintId))
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		if errors.Is(err, context.DeadlineExceeded) {
			errMessage = fmt.Sprintf("%v: %v", tvm.ErrAwaitTxConfirmationTimeout, h.tvmClient.Timeout)
		}
		addErrorToResponseHeader(response, errMessage)
		return true
	}
	if !success {
		addErrorToResponseHeader(response, "minting NFT tx failed")
		return true
	}
	h.logger.Infof("NFT minted with txID: %s\n", txID)
	response.MintResponse.Header.Status = typesv1alpha.StatusType_STATUS_TYPE_SUCCESS
	response.MintTransactionId = txID.String()
	return false
}

func (h *TvmResponseHandler) handleMintRequest(ctx context.Context, response *ResponseContent) bool {
	if response.MintResponse.Header == nil {
		response.MintResponse.Header = &typesv1alpha.ResponseHeader{}
	}
	if response.MintTransactionId == "" {
		addErrorToResponseHeader(response, "missing mint transaction id")
		return true
	}
	mintID, err := ids.FromString(response.MintTransactionId)
	if err != nil {
		addErrorToResponseHeader(response, fmt.Sprintf("error parsing mint transaction id: %v", err))
		return true
	}

	success, txID, err := h.tvmClient.SendTxAndWait(ctx, transferNFTAction(h.tvmClient.Address(), mintID))
	if err != nil {
		errMessage := fmt.Sprintf("error buying NFT: %v", err)
		if errors.Is(err, context.DeadlineExceeded) {
			errMessage = fmt.Sprintf("%v: %v", tvm.ErrAwaitTxConfirmationTimeout, h.tvmClient.Timeout)
		}
		addErrorToResponseHeader(response, errMessage)
		return true
	}
	if !success {
		addErrorToResponseHeader(response, "buying NFT failed")
		return true
	}

	h.logger.Infof("Bought NFT (txID=%s) with ID: %s\n", txID, mintID)
	response.BuyTransactionId = txID.String()
	return false
}

func addErrorToResponseHeader(response *ResponseContent, errMessage string) {
	response.MintResponse.Header.Status = typesv1alpha.StatusType_STATUS_TYPE_FAILURE
	response.MintResponse.Header.Alerts = append(response.MintResponse.Header.Alerts, &typesv1alpha.Alert{
		Message: errMessage,
		Type:    typesv1alpha.AlertType_ALERT_TYPE_ERROR,
	})
}

func NewResponseHandler(tvmClient *tvm.Client, logger *zap.SugaredLogger) *TvmResponseHandler {
	return &TvmResponseHandler{tvmClient: tvmClient, logger: logger}
}

func createNFTAction(owner, buyer codec.Address, purchaseExpiration, price uint64, metadata string) chain.Action {
	return &actions.CreateNFT{
		Owner:                owner,
		Issuer:               owner,
		Buyer:                buyer,
		PurchaseExpiration:   purchaseExpiration,
		Asset:                ids.ID{},
		Price:                price,
		CancellationPolicies: actions.CancellationPolicies{},
		Metadata:             []byte(metadata),
	}
}

func transferNFTAction(newOwner codec.Address, nftID ids.ID) chain.Action {
	return &actions.TransferNFT{
		To:             newOwner,
		ID:             nftID,
		OnChainPayment: false, // TODO change based on tchain configuration
		Memo:           nil,
	}
}

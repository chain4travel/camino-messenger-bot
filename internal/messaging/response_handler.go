/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"
	"errors"
	"fmt"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/chain4travel/camino-messenger-bot/internal/tvm"
	"github.com/chain4travel/caminotravelvm/actions"
	"github.com/chain4travel/caminotravelvm/consts"
	"go.uber.org/zap"
	"strconv"
)

var _ ResponseHandler = (*TvmResponseHandler)(nil)

type ResponseHandler interface {
	HandleResponse(ctx context.Context, msgType MessageType, request *RequestContent, response *ResponseContent) error
}
type TvmResponseHandler struct {
	tvmClient *tvm.Client
	logger    *zap.SugaredLogger
}

func (h *TvmResponseHandler) HandleResponse(ctx context.Context, msgType MessageType, request *RequestContent, response *ResponseContent) error {
	switch msgType {
	case MintRequest:
		if response.MintTransactionId == "" {
			return fmt.Errorf("missing mint transaction id")
		}
		mintID, err := ids.FromString(response.MintTransactionId)
		if err != nil {
			return fmt.Errorf("error parsing mint transaction id: %w", err)
		}
		success, txID, err := h.tvmClient.SendTxAndWait(ctx, transferNFTAction(h.tvmClient.Address(), mintID))
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %v", tvm.ErrAwaitTxConfirmationTimeout, h.tvmClient.Timeout)
		}
		if err != nil {
			return fmt.Errorf("error buying NFT: %v", err)
		}
		if !success {
			return fmt.Errorf("buying NFT failed")
		}
		h.logger.Infof("Bought NFT with ID: %s\n", txID)
		response.BuyTransactionId = txID.String()
		return nil
	case MintResponse:
		owner := h.tvmClient.Address()
		buyer, err := codec.ParseAddressBech32(consts.HRP, request.MintRequest.BuyerAddress)
		if err != nil {
			return fmt.Errorf("error parsing buyer address: %w", err)
		}
		price, err := strconv.Atoi(response.MintResponse.Price.Value)
		if err != nil {
			return fmt.Errorf("error parsing price value: %w", err)
		}
		success, txID, err := h.tvmClient.SendTxAndWait(ctx, createNFTAction(owner, buyer, uint64(response.MintResponse.BuyableUntil.Seconds), uint64(price), response.MintResponse.MintId))
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %v", tvm.ErrAwaitTxConfirmationTimeout, h.tvmClient.Timeout)
		}
		if err != nil {
			return fmt.Errorf("error minting NFT: %v", err)
		}
		if !success {
			return fmt.Errorf("minting NFT tx failed")
		}
		h.logger.Infof("NFT minted with txID: %s\n", txID)
		response.MintTransactionId = txID.String()
		return nil
	}
	return nil
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

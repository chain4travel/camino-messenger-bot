/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"
	"errors"
	"fmt"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/chain4travel/camino-messenger-bot/internal/tvm"
	"github.com/chain4travel/hypersdk/examples/touristicvm/actions"
	"github.com/chain4travel/hypersdk/examples/touristicvm/consts"
)

var _ ResponseHandler = (*TvmResponseHandler)(nil)

type ResponseHandler interface {
	HandleResponse(ctx context.Context, msgType MessageType, response *ResponseContent) error
}
type TvmResponseHandler struct {
	tvmClient *tvm.Client
}

func (h *TvmResponseHandler) HandleResponse(ctx context.Context, msgType MessageType, response *ResponseContent) error {

	if false { //TODO replace with msgType == Mintresponse
		owner, err := codec.ParseAddressBech32(consts.HRP, response.PingMessage) //TODO replace with bot's addr
		if err != nil {
			return fmt.Errorf("error parsing address: %v", err)
		}

		success, txID, err := h.tvmClient.SendTxAndWait(ctx, createNFTAction("https://example.com", "qweqwe", owner))
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %v", tvm.ErrAwaitTxConfirmationTimeout, h.tvmClient.Timeout)
		}
		if err != nil {
			return fmt.Errorf("error minting NFT: %v", err)
		}
		if !success {
			return fmt.Errorf("minting NFT tx failed")
		}
		fmt.Printf("NFT minted with txID: %s\n", txID)
		return nil
	}
	return nil
}

func NewResponseHandler(tvmClient *tvm.Client) *TvmResponseHandler {
	return &TvmResponseHandler{tvmClient: tvmClient}
}
func createNFTAction(URL, metadata string, recipient codec.Address) chain.Action {
	return &actions.CreateNFT{
		Metadata: []byte(metadata),
		Owner:    recipient,
		URL:      []byte(URL),
	}
}

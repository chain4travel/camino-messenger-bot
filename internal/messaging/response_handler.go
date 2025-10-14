// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"time"

	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	bookv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v3"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/common"
	eventlistener "github.com/chain4travel/camino-messenger-bot/v11/internal/eventlistener"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/booking"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

type tokenBuyableUntil struct {
	Default time.Duration
	Minimal time.Duration
	Maximal time.Duration
}

var _ ResponseHandler = (*evmResponseHandler)(nil)

type ResponseHandler interface {
	// Processes incoming response
	ProcessResponseMessage(ctx context.Context, responseMsg *types.Message)

	// Prepares response by performing any necessary modifications to it
	// It expects the request and response to be of the same service.
	PrepareResponseMessage(ctx context.Context, requestMsg *types.Message, responseMsg *types.Message)

	// Prepares request by performing any necessary modifications to it
	PrepareRequest(request protoreflect.ProtoMessage) error
}

func NewResponseHandler(
	logger *zap.SugaredLogger,
	cmAccountAddress ethCommon.Address,
	eventListener eventlistener.EventListener,
	bookingService booking.Service,
	responseHeaderHandler common.ResponseHeaderHandler,
	priceHandler common.PriceHandler,
	e2eTestMode bool,
) (ResponseHandler, error) {
	tokenBuyableUntil := tokenBuyableUntil{
		Default: 300 * time.Second,
		Minimal: 70 * time.Second,
		Maximal: 600 * time.Second,
	}
	if e2eTestMode {
		tokenBuyableUntil.Default = 10 * time.Second
		tokenBuyableUntil.Minimal = 5 * time.Second
		tokenBuyableUntil.Maximal = 20 * time.Second
	}

	return &evmResponseHandler{
		logger:                logger,
		responseHeaderHandler: responseHeaderHandler,
		priceHandler:          priceHandler,
		cmAccountAddressStr:   cmAccountAddress.Hex(),
		bookingService:        bookingService,
		eventListener:         eventListener,
		tokenBuyableUntil:     tokenBuyableUntil,
	}, nil
}

type evmResponseHandler struct {
	logger                *zap.SugaredLogger
	responseHeaderHandler common.ResponseHeaderHandler
	priceHandler          common.PriceHandler
	cmAccountAddressStr   string
	bookingService        booking.Service
	eventListener         eventlistener.EventListener
	tokenBuyableUntil     tokenBuyableUntil
}

// Processes incoming response
func (h *evmResponseHandler) ProcessResponseMessage(
	ctx context.Context,
	responseMsg *types.Message,
) {
	switch response := responseMsg.Content.(type) {
	case *bookv2.MintResponse: // distributor will post-process a mint request to buy the returned NFT
		h.processMintResponseV2(ctx, response)
	case *bookv3.MintResponse: // distributor will post-process a mint request to buy the returned NFT
		h.processMintResponseV3(ctx, response)
	}
}

// Prepares response by performing any necessary modifications to it.
// It expects the request and response to be of the same service.
func (h *evmResponseHandler) PrepareResponseMessage(
	ctx context.Context,
	requestMsg *types.Message,
	responseMsg *types.Message,
) {
	switch response := responseMsg.Content.(type) {
	case *bookv2.MintResponse: // supplier will act upon receiving a mint response by minting an NFT
		h.prepareMintResponseV2(ctx, response, requestMsg.Content.(*bookv2.MintRequest))
	case *bookv3.MintResponse: // supplier will act upon receiving a mint response by minting an NFT
		h.prepareMintResponseV3(ctx, response, requestMsg.Content.(*bookv3.MintRequest))
	}
}

// Prepares request by performing any necessary modifications to it
func (h *evmResponseHandler) PrepareRequest(request protoreflect.ProtoMessage) error {
	switch request := request.(type) {
	case *bookv2.MintRequest:
		request.BuyerAddress = h.cmAccountAddressStr
	case *bookv3.MintRequest:
		request.BuyerAddress = &typesv3.EVMAddress{
			Address: h.cmAccountAddressStr,
		}
	}
	return nil
}

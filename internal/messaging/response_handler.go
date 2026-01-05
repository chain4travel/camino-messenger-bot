// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"time"

	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	bookv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v3"
	bookv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v4"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"

	"google.golang.org/protobuf/reflect/protoreflect"

	eventlistener "github.com/chain4travel/camino-messenger-bot/v12/internal/eventlistener"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/messaging/message"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/price"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/booking"

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
	ProcessResponseMessage(ctx context.Context, requestMsg *message.Message, responseMsg *message.Message)

	// Prepares response by performing any necessary modifications to it
	// It expects the request and response to be of the same service.
	PrepareResponseMessage(ctx context.Context, requestMsg *message.Message, responseMsg *message.Message)

	// Prepares request by performing any necessary modifications to it
	PrepareRequest(request protoreflect.ProtoMessage)
}

func NewResponseHandler(
	logger *zap.SugaredLogger,
	cmAccountAddress ethCommon.Address,
	eventListener eventlistener.EventListener,
	bookingService booking.Service,
	priceHandler price.Handler,
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
		logger:              logger,
		priceHandler:        priceHandler,
		cmAccountAddressStr: cmAccountAddress.Hex(),
		bookingService:      bookingService,
		eventListener:       eventListener,
		tokenBuyableUntil:   tokenBuyableUntil,
	}, nil
}

type evmResponseHandler struct {
	logger              *zap.SugaredLogger
	priceHandler        price.Handler
	cmAccountAddressStr string
	bookingService      booking.Service
	eventListener       eventlistener.EventListener
	tokenBuyableUntil   tokenBuyableUntil
}

// Processes incoming response
func (h *evmResponseHandler) ProcessResponseMessage(
	ctx context.Context,
	requestMsg *message.Message,
	responseMsg *message.Message,
) {
	// distributor will post-process a mint request to buy the returned NFT
	switch response := responseMsg.Content.(type) {
	case *bookv2.MintResponse:
		responseMsg.Content = h.processMintResponseV2(ctx, response)
	case *bookv3.MintResponse:
		responseMsg.Content = h.processMintResponseV3(ctx, response)
	case *bookv4.MintResponse:
		responseMsg.Content = h.processMintResponseV4(ctx, requestMsg.Content.(*bookv4.MintRequest), response)
	}
}

// Prepares response by performing any necessary modifications to it.
// It expects the request and response to be of the same service.
func (h *evmResponseHandler) PrepareResponseMessage(
	ctx context.Context,
	requestMsg *message.Message,
	responseMsg *message.Message,
) {
	// supplier will act upon receiving a mint response by minting an NFT
	switch response := responseMsg.Content.(type) {
	case *bookv2.MintResponse:
		responseMsg.Content = h.prepareMintResponseV2(ctx, requestMsg.Content.(*bookv2.MintRequest), response)
	case *bookv3.MintResponse:
		responseMsg.Content = h.prepareMintResponseV3(ctx, requestMsg.Content.(*bookv3.MintRequest), response)
	case *bookv4.MintResponse:
		responseMsg.Content = h.prepareMintResponseV4(ctx, requestMsg.Content.(*bookv4.MintRequest), response)
	}
}

// Prepares request by performing any necessary modifications to it
func (h *evmResponseHandler) PrepareRequest(request protoreflect.ProtoMessage) {
	switch request := request.(type) {
	case *bookv2.MintRequest:
		request.BuyerAddress = h.cmAccountAddressStr
	case *bookv3.MintRequest:
		request.BuyerAddress = &typesv3.EVMAddress{Address: h.cmAccountAddressStr}
	case *bookv4.MintRequest:
		request.BuyerAddress = &typesv4.EVMAddress{Address: h.cmAccountAddressStr}
	}
}

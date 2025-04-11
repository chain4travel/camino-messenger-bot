// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"time"

	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"

	"google.golang.org/protobuf/reflect/protoreflect"

	eventlistener "github.com/chain4travel/camino-messenger-bot/internal/event_listener"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	"github.com/chain4travel/camino-messenger-bot/pkg/erc20"

	"github.com/ethereum/go-ethereum/common"
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

	// Adds an error message to the response header
	AddErrorToResponseHeader(response protoreflect.ProtoMessage, errMessage string)
}

func NewResponseHandler(
	logger *zap.SugaredLogger,
	cmAccountAddress common.Address,
	eventListener eventlistener.EventListener,
	bookingService booking.Service,
	erc20 erc20.Service,
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
		cmAccountAddressStr: cmAccountAddress.Hex(),
		bookingService:      bookingService,
		eventListener:       eventListener,
		erc20:               erc20,
		tokenBuyableUntil:   tokenBuyableUntil,
	}, nil
}

type evmResponseHandler struct {
	logger              *zap.SugaredLogger
	cmAccountAddressStr string
	bookingService      booking.Service
	eventListener       eventlistener.EventListener
	erc20               erc20.Service
	tokenBuyableUntil   tokenBuyableUntil
}

// Processes incoming response
func (h *evmResponseHandler) ProcessResponseMessage(
	ctx context.Context,
	responseMsg *types.Message,
) {
	switch response := responseMsg.Content.(type) {
	case *bookv1.MintResponse: // distributor will post-process a mint request to buy the returned NFT
		h.processMintResponseV1(ctx, response)
	case *bookv2.MintResponse: // distributor will post-process a mint request to buy the returned NFT
		h.processMintResponseV2(ctx, response)
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
	case *bookv1.MintResponse: // supplier will act upon receiving a mint response by minting an NFT
		h.prepareMintResponseV1(ctx, response, requestMsg.Content.(*bookv1.MintRequest))
	case *bookv2.MintResponse: // supplier will act upon receiving a mint response by minting an NFT
		h.prepareMintResponseV2(ctx, response, requestMsg.Content.(*bookv2.MintRequest))
	}
}

// Prepares request by performing any necessary modifications to it
func (h *evmResponseHandler) PrepareRequest(request protoreflect.ProtoMessage) error {
	switch request := request.(type) {
	case *bookv1.MintRequest:
		request.BuyerAddress = h.cmAccountAddressStr
	case *bookv2.MintRequest:
		request.BuyerAddress = h.cmAccountAddressStr
	}
	return nil
}

func (h *evmResponseHandler) AddErrorToResponseHeader(response protoreflect.ProtoMessage, errMessage string) {
	headerFieldDescriptor := response.ProtoReflect().Descriptor().Fields().ByName("header")
	headerReflectValue := response.ProtoReflect().Get(headerFieldDescriptor)

	switch header := headerReflectValue.Message().Interface().(type) {
	case *typesv1.ResponseHeader:
		addErrorToResponseHeaderV1(header, errMessage)
	default:
		h.logger.Errorf("failed add error to response header: %v", errMessage)
	}
}

func addErrorToResponseHeaderV1(header *typesv1.ResponseHeader, errMessage string) {
	header.Status = typesv1.StatusType_STATUS_TYPE_FAILURE
	header.Alerts = append(header.Alerts, &typesv1.Alert{
		Message: errMessage,
		Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
	})
}

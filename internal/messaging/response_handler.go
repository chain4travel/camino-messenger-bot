//  Copyright (C) 2022-2024, Chain4Travel AG. All rights reserved.
//  See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"time"

	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	"github.com/chain4travel/camino-messenger-bot/pkg/events"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

const (
	buyableUntilDurationDefault = 300 * time.Second
	buyableUntilDurationMinimal = 70 * time.Second
	buyableUntilDurationMaximal = 600 * time.Second
)

var (
	_ ResponseHandler = (*evmResponseHandler)(nil)

	zeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
)

// TODO@ move this to processor?
// TODO@ two reason its there:
// TODO@ 1) to make processor not depend on protobuf types (but package depends regardless)
// TODO@ 2) to abstract all message-specific logic into response handler. But I'm not sure if it worth it.
type ResponseHandler interface {
	// Processes incoming response
	ProcessResponseMessage(ctx context.Context, requestMsg *types.Message, responseMsg *types.Message)

	// Prepares response by performing any necessary modifications to it
	PrepareResponseMessage(ctx context.Context, requestMsg *types.Message, responseMsg *types.Message)

	// Prepares request by performing any necessary modifications to it
	PrepareRequest(msgType types.MessageType, request protoreflect.ProtoMessage) error

	// Adds an error message to the response header
	AddErrorToResponseHeader(response protoreflect.ProtoMessage, errMessage string)
}

func NewResponseHandler(
	botKey *ecdsa.PrivateKey,
	ethClient *ethclient.Client,
	logger *zap.SugaredLogger,
	cmAccountAddress common.Address,
	bookingTokenAddress common.Address,
	serviceRegistry ServiceRegistry,
) (ResponseHandler, error) {
	bookingService, err := booking.NewService(cmAccountAddress, botKey, ethClient, logger)
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}

	bookingToken, err := bookingtoken.NewBookingtoken(bookingTokenAddress, ethClient)
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}
	return &evmResponseHandler{
		ethClient:           ethClient,
		logger:              logger,
		cmAccountAddress:    cmAccountAddress,
		bookingTokenAddress: bookingTokenAddress,
		bookingService:      *bookingService,
		bookingToken:        *bookingToken,
		serviceRegistry:     serviceRegistry,
		evmEventListener:    events.NewEventListener(ethClient, logger),
	}, nil
}

type evmResponseHandler struct {
	ethClient           *ethclient.Client
	logger              *zap.SugaredLogger
	cmAccountAddress    common.Address
	bookingTokenAddress common.Address
	bookingService      booking.Service
	bookingToken        bookingtoken.Bookingtoken
	serviceRegistry     ServiceRegistry
	evmEventListener    *events.EventListener
}

func (h *evmResponseHandler) ProcessResponseMessage(
	ctx context.Context,
	requestMsg *types.Message,
	responseMsg *types.Message,
) {
	switch requestMsg.Type {
	case generated.MintServiceV1Request: // distributor will post-process a mint request to buy the returned NFT
		h.processMintResponseV1(ctx, responseMsg.Content)
	case generated.MintServiceV2Request: // distributor will post-process a mint request to buy the returned NFT
		h.processMintResponseV2(ctx, responseMsg.Content)
	}
}

func (h *evmResponseHandler) PrepareResponseMessage(
	ctx context.Context,
	requestMsg *types.Message,
	responseMsg *types.Message,
) {
	switch response := responseMsg.Content.(type) {
	case *bookv1.MintResponse: // supplier will act upon receiving a mint response by minting an NFT
		h.prepareMintResponseV1(ctx, response, requestMsg.Content)
	case *bookv2.MintResponse: // supplier will act upon receiving a mint response by minting an NFT
		h.prepareMintResponseV2(ctx, response, requestMsg.Content)
	}
}

func (h *evmResponseHandler) PrepareRequest(msgType types.MessageType, request protoreflect.ProtoMessage) error {
	switch msgType {
	case generated.MintServiceV2Request:
		mintReq, ok := request.(*bookv2.MintRequest)
		if !ok {
			return nil
		}
		mintReq.BuyerAddress = h.cmAccountAddress.Hex()
	case generated.MintServiceV1Request:
		mintReq, ok := request.(*bookv1.MintRequest)
		if !ok {
			return nil
		}
		mintReq.BuyerAddress = h.cmAccountAddress.Hex()
	}
	return nil
}

// Waits for a transaction to be mined
func (h *evmResponseHandler) waitTransaction(ctx context.Context, tx *ethTypes.Transaction) (receipt *ethTypes.Receipt, err error) {
	h.logger.Infof("Waiting for transaction to be mined...\n")

	receipt, err = bind.WaitMined(ctx, h.ethClient, tx)
	if err != nil {
		return receipt, err
	}

	if receipt.Status != ethTypes.ReceiptStatusSuccessful {
		return receipt, fmt.Errorf("transaction failed: %v", receipt)
	}

	h.logger.Infof("Successfully mined. Block Nr: %s Gas used: %d\n", receipt.BlockNumber, receipt.GasUsed)

	return receipt, nil
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

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
	"github.com/chain4travel/camino-messenger-bot/pkg/erc20"
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

type ResponseHandler interface {
	HandleResponse(ctx context.Context, msgType types.MessageType, request protoreflect.ProtoMessage, response protoreflect.ProtoMessage)
	HandleRequest(ctx context.Context, msgType types.MessageType, request protoreflect.ProtoMessage) error
	AddErrorToResponseHeader(response protoreflect.ProtoMessage, errMessage string)
}

func NewResponseHandler(
	botKey *ecdsa.PrivateKey,
	ethClient *ethclient.Client,
	logger *zap.SugaredLogger,
	cmAccountAddress common.Address,
	bookingTokenAddress common.Address,
	serviceRegistry ServiceRegistry,
	tokenCacheSize int,
) (ResponseHandler, error) {
	erc20, err := erc20.NewERC20Service(ethClient, tokenCacheSize)
	if err != nil {
		return nil, err
	}

	bookingService, err := booking.NewService(&cmAccountAddress, botKey, ethClient, logger, erc20)
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
		erc20:               erc20,
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
	erc20               erc20.Service
}

func (h *evmResponseHandler) HandleResponse(ctx context.Context, msgType types.MessageType, request protoreflect.ProtoMessage, response protoreflect.ProtoMessage) {
	switch msgType {
	case generated.MintServiceV1Request: // distributor will post-process a mint request to buy the returned NFT
		if h.handleMintRequestV1(ctx, response) {
			return // TODO @evlekht we don't need this if true/false then do nothing
		}
	case generated.MintServiceV1Response: // supplier will act upon receiving a mint response by minting an NFT
		if h.handleMintResponseV1(ctx, response, request) {
			return // TODO @evlekht we don't need this if true/false then do nothing
		}
	case generated.MintServiceV2Request: // distributor will post-process a mint request to buy the returned NFT
		if h.handleMintRequestV2(ctx, response) {
			return // TODO @evlekht we don't need this if true/false then do nothing
		}
	case generated.MintServiceV2Response: // supplier will act upon receiving a mint response by minting an NFT
		if h.handleMintResponseV2(ctx, response, request) {
			return // TODO @evlekht we don't need this if true/false then do nothing
		}
	}
}

func (h *evmResponseHandler) HandleRequest(_ context.Context, msgType types.MessageType, request protoreflect.ProtoMessage) error {
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

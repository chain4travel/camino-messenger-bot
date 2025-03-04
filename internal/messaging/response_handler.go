// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"time"

	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	notificationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"

	"google.golang.org/grpc"
	grpc_metadata "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/pkg/erc20"
	"github.com/chain4travel/camino-messenger-bot/pkg/events"
	events_storage "github.com/chain4travel/camino-messenger-bot/pkg/events/storage"
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

// BookingStatus represents the status of a booking token
type BookingStatus int

const (
	// BookingStatusBought indicates the token has been purchased
	BookingStatusBought BookingStatus = 3
	// BookingStatusExpired indicates the token has expired
	BookingStatusExpired BookingStatus = 4
)

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

	// ReloadTokensFromStorage loads all tokens from storage that are not marked as bought or expired
	// and re-registers their event listeners
	ReloadTokensFromStorage(ctx context.Context) error
}

// Make evmResponseHandler private again
type evmResponseHandler struct {
	ethClient           *ethclient.Client
	logger              *zap.SugaredLogger
	cmAccountAddress    common.Address
	bookingTokenAddress common.Address
	bookingService      booking.Service
	bookingToken        bookingtoken.Bookingtoken
	serviceRegistry     ServiceRegistry
	evmEventListener    *events.EventListener
	evmEventStorage     events_storage.Storage
	recordExpiration    bool
	erc20               erc20.Service
}

func NewResponseHandler(
	botKey *ecdsa.PrivateKey,
	ethClient *ethclient.Client,
	logger *zap.SugaredLogger,
	cmAccountAddress common.Address,
	bookingTokenAddress common.Address,
	serviceRegistry ServiceRegistry,
	cmAccounts cmaccounts.Service,
	tokenCacheSize int,
	storage events_storage.Storage,
	recordExpiration bool,
) (ResponseHandler, error) {
	erc20, err := erc20.NewERC20Service(ethClient, tokenCacheSize)
	if err != nil {
		return nil, err
	}

	bookingService, err := booking.NewService(cmAccountAddress, botKey, ethClient, logger, erc20, cmAccounts)
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
		evmEventStorage:     storage,
		erc20:               erc20,
		recordExpiration:    recordExpiration,
	}, nil
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
		request.BuyerAddress = h.cmAccountAddress.Hex()
	case *bookv2.MintRequest:
		request.BuyerAddress = h.cmAccountAddress.Hex()
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

// Implement the ReloadTokensFromStorage method from the ResponseHandler interface
func (h *evmResponseHandler) ReloadTokensFromStorage(ctx context.Context) error {
	session, err := h.evmEventStorage.NewSession(ctx)
	notificationClient := h.serviceRegistry.NotificationClient()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Abort()

	tokens, err := h.evmEventStorage.GetActiveTokenRecords(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to get active token records: %w", err)
	}

	h.logger.Infof("Reloading %d active tokens from storage", len(tokens))

	for _, token := range tokens {
		tokenID, ok := new(big.Int).SetString(token.TokenID, 10)
		if !ok {
			h.logger.Errorf("failed to parse token ID: %s", token.TokenID)
			continue
		}

		// ask for the on chain status of the token - is it bought or not

		// check the status of the token -> if it is not bought, then we can just expire the token
		tx, err := h.bookingToken.GetBookingStatus(nil, tokenID)
		if err != nil {
			h.logger.Errorf("failed to get booking status: %v", err)
			continue
		}

		// if it is bought or expired, no need to unregister the listeners and mark the token as bought in the database
		if tx == uint8(BookingStatusBought) || tx == uint8(BookingStatusExpired) {
			token.Bought = true

			update_session, err := h.evmEventStorage.NewSession(ctx)
			if err != nil {
				h.logger.Errorf("failed to create session: %v", err)
				continue
			}
			defer update_session.Abort()

			// If token is bought (status 3) but not marked as bought in our database, update it
			err = h.evmEventStorage.UpdateTokenRecord(ctx, update_session, token)
			if err != nil {
				h.logger.Errorf("failed to mark token as bought: %v", err)
			}

			// send a notification to the supplier plugin that the token is bought
			if _, err := notificationClient.TokenBoughtNotification(
				ctx,
				&notificationv1.TokenBought{
					TokenId: tokenID.Uint64(),
					TxId:    "", // TODO: check if important and if we can get the tx id from the booking token contract
					MintId:  token.MintID,
				},
				grpc.Header(&grpc_metadata.MD{}),
			); err != nil {
				h.logger.Errorf("error calling partner plugin TokenBoughtNotification service: %v", err)
			}

			continue
		}

		// if it is not bought, we need to register the listeners
		// send a notification to the supplier plugin that the token is expired

		// Parse expiration time from bytes
		expiresAt := big.NewInt(0).SetBytes(token.ExpiresAt)
		expirationTime := time.Unix(expiresAt.Int64(), 0)

		h.logger.Infof("Re-registering listeners for token %s, expires at %s", tokenID.String(), expirationTime)
		registerTokenListeners(h, tokenID, token.MintID, expirationTime)
	}

	return nil
}

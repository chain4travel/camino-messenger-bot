//  Copyright (C) 2022-2024, Chain4Travel AG. All rights reserved.
//  See the file LICENSE for licensing terms.

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
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	config "github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	"github.com/chain4travel/camino-messenger-bot/pkg/events"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/eth-go-bindings/erc20"
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
	ethClient *ethclient.Client,
	logger *zap.SugaredLogger,
	cfg *config.EvmConfig,
	serviceRegistry ServiceRegistry,
	evmEventListener *events.EventListener,
) (ResponseHandler, error) {
	ecdsaPk, err := crypto.HexToECDSA(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	bookingService, err := booking.NewService(common.HexToAddress(cfg.CMAccountAddress), ecdsaPk, ethClient, logger)
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}

	bookingToken, err := bookingtoken.NewBookingtoken(common.HexToAddress(cfg.BookingTokenAddress), ethClient)
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}
	return &evmResponseHandler{
		ethClient:           ethClient,
		logger:              logger,
		pk:                  ecdsaPk,
		cmAccountAddress:    common.HexToAddress(cfg.CMAccountAddress),
		bookingTokenAddress: common.HexToAddress(cfg.BookingTokenAddress),
		bookingService:      *bookingService,
		bookingToken:        *bookingToken,
		serviceRegistry:     serviceRegistry,
		evmEventListener:    evmEventListener,
	}, nil
}

type evmResponseHandler struct {
	ethClient           *ethclient.Client
	logger              *zap.SugaredLogger
	pk                  *ecdsa.PrivateKey
	cmAccountAddress    common.Address
	bookingTokenAddress common.Address
	bookingService      booking.Service
	bookingToken        bookingtoken.Bookingtoken
	serviceRegistry     ServiceRegistry
	evmEventListener    *events.EventListener
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
	switch msgType { //nolint:gocritic
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

func (h *evmResponseHandler) handleMintResponseV2(ctx context.Context, response protoreflect.ProtoMessage, request protoreflect.ProtoMessage) bool {
	mintResp, ok := response.(*bookv2.MintResponse)
	if !ok {
		return false
	}
	mintReq, ok := request.(*bookv2.MintRequest)
	if !ok {
		return false
	}
	if mintResp.Header == nil {
		mintResp.Header = &typesv1.ResponseHeader{}
	}

	// TODO: @VjeraTurk check if CMAccount exists
	// TODO @evlekht ensure that mintReq.BuyerAddress is c-chain address format, not x/p/t chain
	buyerAddress := common.HexToAddress(mintReq.BuyerAddress)

	tokenURI := mintResp.BookingTokenUri

	if tokenURI == "" {
		// Get a Token URI for the token.
		var jsonPlain string
		jsonPlain, tokenURI, _ = createTokenURIforMintResponse(mintResp)
		h.logger.Debugf("Token URI JSON: %s\n", jsonPlain)
	} else {
		h.logger.Debugf("Token URI: %s\n", tokenURI)
	}

	currentTime := time.Now()

	switch {
	case mintResp.BuyableUntil == nil || mintResp.BuyableUntil.Seconds == 0:
		// BuyableUntil not set
		mintResp.BuyableUntil = timestamppb.New(currentTime.Add(buyableUntilDurationDefault))

	case mintResp.BuyableUntil.Seconds < timestamppb.New(currentTime).Seconds:
		// BuyableUntil in the past
		errMsg := fmt.Sprintf("Refused to mint token - BuyableUntil in the past:  %v", mintResp.BuyableUntil)
		h.AddErrorToResponseHeader(response, errMsg)
		return true

	case mintResp.BuyableUntil.Seconds < timestamppb.New(currentTime.Add(buyableUntilDurationMinimal)).Seconds:
		// BuyableUntil too early
		mintResp.BuyableUntil = timestamppb.New(currentTime.Add(buyableUntilDurationMinimal))

	case mintResp.BuyableUntil.Seconds > timestamppb.New(currentTime.Add(buyableUntilDurationMaximal)).Seconds:
		// BuyableUntil too late
		mintResp.BuyableUntil = timestamppb.New(currentTime.Add(buyableUntilDurationMaximal))
	}

	// MINT TOKEN
	txID, tokenID, err := h.mintv2(
		ctx,
		buyerAddress,
		tokenURI,
		big.NewInt(mintResp.BuyableUntil.Seconds),
		mintResp.Price,
	)
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(response, errMessage)
		return true
	}

	h.logger.Infof("NFT minted with txID: %s\n", txID)

	h.onBookingTokenMint(tokenID, mintResp.MintId, mintResp.BuyableUntil.AsTime())

	// Header is of typev1
	mintResp.Header.Status = typesv1.StatusType_STATUS_TYPE_SUCCESS
	// Disable Linter: This code will be removed with the new mint logic and protocol
	mintResp.BookingTokenId = tokenID.Uint64()
	mintResp.MintTransactionId = txID
	return false
}

func (h *evmResponseHandler) handleMintRequestV2(ctx context.Context, response protoreflect.ProtoMessage) bool {
	mintResp, ok := response.(*bookv2.MintResponse)
	if !ok {
		return false
	}
	if mintResp.Header == nil {
		mintResp.Header = &typesv1.ResponseHeader{}
	}
	if mintResp.MintTransactionId == "" {
		h.AddErrorToResponseHeader(response, "missing mint transaction id")
		return true
	}

	tokenID := new(big.Int).SetUint64(mintResp.BookingTokenId)

	txID, err := h.buy(ctx, tokenID)
	if err != nil {
		errMessage := fmt.Sprintf("error buying NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(response, errMessage)
		return true
	}

	h.logger.Infof("Bought NFT (txID=%s) with ID: %s\n", txID, mintResp.MintTransactionId)
	mintResp.BuyTransactionId = txID
	return false
}

// Mints a BookingToken with the supplier private key and reserves it for the buyer address
// For testing you can use this uri: "data:application/json;base64,eyJuYW1lIjoiQ2FtaW5vIE1lc3NlbmdlciBCb29raW5nVG9rZW4gVGVzdCJ9Cg=="
func (h *evmResponseHandler) mintv2(
	ctx context.Context,
	reservedFor common.Address,
	uri string,
	expiration *big.Int,
	price *typesv2.Price,
) (string, *big.Int, error) {
	bigIntPrice := big.NewInt(0)
	paymentToken := zeroAddress
	var err error

	switch currency := price.Currency.Currency.(type) {
	case *typesv2.Currency_NativeToken:
		bigIntPrice, err = h.bookingService.ConvertPriceToBigInt(price, int32(18)) // CAM uses 18 decimals
		if err != nil {
			return "", nil, err
		}
		paymentToken = zeroAddress
	case *typesv2.Currency_TokenCurrency:
		if !common.IsHexAddress(currency.TokenCurrency.ContractAddress) {
			return "", nil, fmt.Errorf("invalid contract address: %s", currency.TokenCurrency.ContractAddress)
		}
		contractAddress := common.HexToAddress(currency.TokenCurrency.ContractAddress)

		token, err := erc20.NewErc20(contractAddress, h.ethClient)
		if err != nil {
			return "", nil, fmt.Errorf("failed to instantiate ERC20 contract: %w", err)
		}

		tokenDecimals, err := token.Decimals(&bind.CallOpts{Context: ctx})
		if err != nil {
			return "", nil, fmt.Errorf("failed to fetch token decimals: %w", err)
		}

		bigIntPrice, err = h.bookingService.ConvertPriceToBigInt(price, int32(tokenDecimals))
		if err != nil {
			return "", nil, err
		}

		paymentToken = contractAddress
	case *typesv2.Currency_IsoCurrency:
		// For IsoCurrency, keep price as 0 and paymentToken as zeroAddress
		bigIntPrice = big.NewInt(0)
		paymentToken = zeroAddress
	}

	tx, err := h.bookingService.MintBookingToken(
		reservedFor,
		uri,
		expiration,
		bigIntPrice,
		paymentToken)
	if err != nil {
		return "", nil, err
	}

	// Wait for transaction to be mined
	receipt, err := bind.WaitMined(ctx, h.ethClient, tx)
	if err != nil {
		return "", nil, err
	}

	tokenID := big.NewInt(0)

	for _, mLog := range receipt.Logs {
		event, err := h.bookingToken.ParseTokenReserved(*mLog)
		if err == nil {
			tokenID = event.TokenId
			h.logger.Infof("[TokenReserved] TokenID: %s ReservedFor: %s Price: %s, PaymentToken: %s", event.TokenId, event.ReservedFor, event.Price, event.PaymentToken)
		}
	}

	return tx.Hash().Hex(), tokenID, nil
}

// TODO @VjeraTurk code that creates and handles context should be improved, since its not doing job in separate goroutine,
// Buys a token with the buyer private key. Token must be reserved for the buyer address.
func (h *evmResponseHandler) buy(ctx context.Context, tokenID *big.Int) (string, error) {
	tx, err := h.bookingService.BuyBookingToken(tokenID)
	if err != nil {
		return "", err
	}

	receipt, err := h.waitTransaction(ctx, tx)
	if err != nil {
		return "", err
	}
	if receipt.Status != ethTypes.ReceiptStatusSuccessful {
		return "", fmt.Errorf("transaction failed: %v", receipt)
	}

	h.logger.Infof("Transaction sent!\nTransaction hash: %s\n", tx.Hash().Hex())

	return tx.Hash().Hex(), nil
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

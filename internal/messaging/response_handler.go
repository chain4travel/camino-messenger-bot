//  Copyright (C) 2022-2024, Chain4Travel AG. All rights reserved.
//  See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"time"

	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	notificationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	grpc "google.golang.org/grpc"
	grpc_metadata "google.golang.org/grpc/metadata"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	config "github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/messages"
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

type ResponseHandler interface {
	HandleResponse(ctx context.Context, msgType messages.MessageType, request protoreflect.ProtoMessage, response protoreflect.ProtoMessage)
	HandleRequest(ctx context.Context, msgType messages.MessageType, request protoreflect.ProtoMessage) error
	AddErrorToResponseHeader(msgType messages.MessageType, response protoreflect.ProtoMessage, errMessage string)
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

func (h *evmResponseHandler) HandleResponse(ctx context.Context, msgType messages.MessageType, request protoreflect.ProtoMessage, response protoreflect.ProtoMessage) {
	switch msgType {
	case messages.MintRequest: // distributor will post-process a mint request to buy the returned NFT
		if h.handleMintRequest(ctx, response) {
			return
		}
	case messages.MintResponse: // supplier will act upon receiving a mint response by minting an NFT
		if h.handleMintResponseV2(ctx, response, request) {
			return
		}
	}
}

func (h *evmResponseHandler) HandleRequest(_ context.Context, msgType messages.MessageType, request protoreflect.ProtoMessage) error {
	switch msgType { //nolint:gocritic
	case messages.MintRequest:
		mintReq, ok := request.(*bookv2.MintRequest)
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
		h.AddErrorToResponseHeader(messages.MintResponse, response, errMsg)
		return true

	case mintResp.BuyableUntil.Seconds < timestamppb.New(currentTime.Add(buyableUntilDurationMinimal)).Seconds:
		// BuyableUntil too early
		mintResp.BuyableUntil = timestamppb.New(currentTime.Add(buyableUntilDurationMinimal))

	case mintResp.BuyableUntil.Seconds > timestamppb.New(currentTime.Add(buyableUntilDurationMaximal)).Seconds:
		// BuyableUntil too late
		mintResp.BuyableUntil = timestamppb.New(currentTime.Add(buyableUntilDurationMaximal))
	}

	// MINT TOKEN
	txID, tokenID, err := h.mint(
		ctx,
		buyerAddress,
		tokenURI,
		big.NewInt(mintResp.BuyableUntil.Seconds),
		mintResp.Price,
	)
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(messages.MintResponse, response, errMessage)
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

func (h *evmResponseHandler) handleMintRequest(ctx context.Context, response protoreflect.ProtoMessage) bool {
	mintResp, ok := response.(*bookv2.MintResponse)
	if !ok {
		return false
	}
	if mintResp.Header == nil {
		mintResp.Header = &typesv1.ResponseHeader{}
	}
	if mintResp.MintTransactionId == "" {
		h.AddErrorToResponseHeader(messages.MintResponse, response, "missing mint transaction id")
		return true
	}

	tokenID := new(big.Int).SetUint64(mintResp.BookingTokenId)

	txID, err := h.buy(ctx, tokenID)
	if err != nil {
		errMessage := fmt.Sprintf("error buying NFT: %v", err)
		h.logger.Errorf(errMessage)
		h.AddErrorToResponseHeader(messages.MintResponse, response, errMessage)
		return true
	}

	h.logger.Infof("Bought NFT (txID=%s) with ID: %s\n", txID, mintResp.MintTransactionId)
	mintResp.BuyTransactionId = txID
	return false
}

// Mints a BookingToken with the supplier private key and reserves it for the buyer address
// For testing you can use this uri: "data:application/json;base64,eyJuYW1lIjoiQ2FtaW5vIE1lc3NlbmdlciBCb29raW5nVG9rZW4gVGVzdCJ9Cg=="
func (h *evmResponseHandler) mint(
	ctx context.Context,
	reservedFor common.Address,
	uri string,
	expiration *big.Int,
	price *typesv2.Price,
) (string, *big.Int, error) {
	bigIntPrice := big.NewInt(0)
	paymentToken := zeroAddress
	var err error

	//  TODO:
	// (in booking package)
	// define paymentToken from currency
	// if TokenCurrency get paymentToken contract and call decimals()
	// calculate the price in big int without loosing precision

	switch price.Currency.Currency.(type) {
	case *typesv2.Currency_NativeToken:
		bigIntPrice, err = h.bookingService.ConvertPriceToBigInt(price, int32(18)) // CAM uses 18 decimals
		if err != nil {
			return "", nil, err
		}
		paymentToken = zeroAddress
	case *typesv2.Currency_TokenCurrency:
		// Add logic to handle TokenCurrency
		// if contract address is zeroAddress, then it is native token
		return "", nil, fmt.Errorf("TokenCurrency not supported yet")
	case *typesv2.Currency_IsoCurrency:
		// Add logic to handle IsoCurrency
		return "", nil, fmt.Errorf("IsoCurrency not supported yet")
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
	if receipt.Status != types.ReceiptStatusSuccessful {
		return "", fmt.Errorf("transaction failed: %v", receipt)
	}

	h.logger.Infof("Transaction sent!\nTransaction hash: %s\n", tx.Hash().Hex())

	return tx.Hash().Hex(), nil
}

// Waits for a transaction to be mined
func (h *evmResponseHandler) waitTransaction(ctx context.Context, tx *types.Transaction) (receipt *types.Receipt, err error) {
	h.logger.Infof("Waiting for transaction to be mined...\n")

	receipt, err = bind.WaitMined(ctx, h.ethClient, tx)
	if err != nil {
		return receipt, err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return receipt, fmt.Errorf("transaction failed: %v", receipt)
	}

	h.logger.Infof("Successfully mined. Block Nr: %s Gas used: %d\n", receipt.BlockNumber, receipt.GasUsed)

	return receipt, nil
}

func (h *evmResponseHandler) onBookingTokenMint(tokenID *big.Int, mintID *typesv1.UUID, buyableUntil time.Time) {
	notificationClient := h.serviceRegistry.NotificationClient()
	expirationTimer := &time.Timer{}

	unsubscribeTokenBought, err := h.evmEventListener.RegisterTokenBoughtHandler(
		h.bookingTokenAddress,
		[]*big.Int{tokenID},
		nil,
		func(e any) {
			expirationTimer.Stop()
			h.logger.Infof("Token bought event received for token %s", tokenID.String())
			event := e.(*bookingtoken.BookingtokenTokenBought)

			if _, err := notificationClient.TokenBoughtNotification(
				context.Background(),
				&notificationv1.TokenBought{
					TokenId: tokenID.Uint64(),
					TxId:    event.Raw.TxHash.Hex(),
					MintId:  mintID,
				},
				grpc.Header(&grpc_metadata.MD{}),
			); err != nil {
				h.logger.Errorf("error calling partner plugin TokenBoughtNotification service: %v", err)
			}
		},
	)
	if err != nil {
		h.logger.Errorf("failed to register handler: %v", err)
		// TODO @evlekht send some notification to partner plugin
		return
	}

	expirationTimer = time.AfterFunc(time.Until(buyableUntil), func() {
		unsubscribeTokenBought()
		h.logger.Infof("Token %s expired", tokenID.String())

		if _, err := notificationClient.TokenExpiredNotification(
			context.Background(),
			&notificationv1.TokenExpired{
				TokenId: tokenID.Uint64(),
				MintId:  mintID,
			},
			grpc.Header(&grpc_metadata.MD{}),
		); err != nil {
			h.logger.Errorf("error calling partner plugin TokenExpiredNotification service: %v", err)
		}
	})
}

// TODO @evlekht check if those structs are needed as exported here, otherwise make them private or move to another pkg
type Attribute struct {
	TraitType string `json:"trait_type"`
	Value     string `json:"value"`
}

type HotelJSON struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Date        string      `json:"date,omitempty"`
	ExternalURL string      `json:"external_url,omitempty"`
	Image       string      `json:"image,omitempty"`
	Attributes  []Attribute `json:"attributes,omitempty"`
}

func generateAndEncodeJSON(name, description, date, externalURL, image string, attributes []Attribute) (string, string, error) {
	hotel := HotelJSON{
		Name:        name,
		Description: description,
		Date:        date,
		ExternalURL: externalURL,
		Image:       image,
		Attributes:  attributes,
	}

	jsonData, err := json.Marshal(hotel)
	if err != nil {
		return "", "", err
	}

	encoded := base64.StdEncoding.EncodeToString(jsonData)
	return string(jsonData), encoded, nil
}

// Generates a token data URI from a MintResponse object. Returns jsonPlain and a
// data URI with base64 encoded json data.
//
// TODO: @havan: We need decide what data needs to be in the tokenURI JSON and add
// those fields to the MintResponse. These will be shown in the UI of wallets,
// explorers etc.
func createTokenURIforMintResponse(mintResponse *bookv2.MintResponse) (string, string, error) {
	// TODO: What should we use for a token name? This will be shown in the UI of wallets, explorers etc.
	name := "CM Booking Token"

	// TODO: What should we use for a token description? This will be shown in the UI of wallets, explorers etc.
	description := "This NFT represents the booking with the specified attributes."

	// Dummy data
	date := "2024-09-27"

	externalURL := "https://camino.network"

	// Placeholder Image
	image := "https://camino.network/static/images/N9IkxmG-Sg-1800.webp"

	attributes := []Attribute{
		{
			TraitType: "Mint ID",
			Value:     mintResponse.GetMintId().Value,
		},
		{
			TraitType: "Reference",
			Value:     mintResponse.GetProviderBookingReference(),
		},
	}

	jsonPlain, jsonEncoded, err := generateAndEncodeJSON(
		name,
		description,
		date,
		externalURL,
		image,
		attributes,
	)
	if err != nil {
		return "", "", err
	}

	// Add data URI scheme
	tokenURI := "data:application/json;base64," + jsonEncoded

	return jsonPlain, tokenURI, nil
}

func (h *evmResponseHandler) AddErrorToResponseHeader(msgType messages.MessageType, response protoreflect.ProtoMessage, errMessage string) {
	headerFieldDescriptor := response.ProtoReflect().Descriptor().Fields().ByName("Header")
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

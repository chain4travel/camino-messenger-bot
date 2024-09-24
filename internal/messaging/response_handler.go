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

	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"

	config "github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var _ ResponseHandler = (*evmResponseHandler)(nil)

type ResponseHandler interface {
	HandleResponse(ctx context.Context, msgType MessageType, request *RequestContent, response *ResponseContent)
	HandleRequest(ctx context.Context, msgType MessageType, request *RequestContent) error
}

func NewResponseHandler(ethClient *ethclient.Client, logger *zap.SugaredLogger, cfg *config.EvmConfig) (ResponseHandler, error) {
	pk := new(secp256k1.PrivateKey)
	// UnmarshalText expects the private key in quotes
	if err := pk.UnmarshalText([]byte("\"" + cfg.PrivateKey + "\"")); err != nil {
		logger.Fatalf("Failed to parse private key: %v", err)
	}
	ecdsaPk := pk.ToECDSA()
	bookingService, err := booking.NewService(common.HexToAddress(cfg.CMAccountAddress), ecdsaPk, ethClient, logger)
	if err != nil {
		log.Fatalf("%v", err)
		return nil, err
	}

	bookingToken, err := bookingtoken.NewBookingtoken(common.HexToAddress(cfg.BookingTokenAddress), ethClient)
	if err != nil {
		log.Fatalf("%v", err)
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
		// Disable Linter: This code will be removed with the new BookingToken implementation
		buyableUntilDefault: time.Second * time.Duration(cfg.BuyableUntilDefault), // #nosec G115
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
	buyableUntilDefault time.Duration
}

func (h *evmResponseHandler) HandleResponse(ctx context.Context, msgType MessageType, request *RequestContent, response *ResponseContent) {
	switch msgType {
	case MintRequest: // distributor will post-process a mint request to buy the returned NFT
		if h.handleMintRequest(ctx, response) {
			return
		}
	case MintResponse: // supplier will act upon receiving a mint response by minting an NFT
		if h.handleMintResponse(ctx, response, request) {
			return
		}
	}
}

func (h *evmResponseHandler) HandleRequest(_ context.Context, msgType MessageType, request *RequestContent) error {
	switch msgType { //nolint:gocritic
	case MintRequest:
		request.BuyerAddress = h.cmAccountAddress.Hex()
	}
	return nil
}

func (h *evmResponseHandler) handleMintResponse(ctx context.Context, response *ResponseContent, request *RequestContent) bool {
	if response.MintResponse.Header == nil {
		response.MintResponse.Header = &typesv1.ResponseHeader{}
	}

	//TODO: @VjeraTurk check if CMAccount exists
	// TODO @evlekht ensure that request.MintRequest.BuyerAddress is c-chain address format, not x/p/t chain
	buyerAddress := common.HexToAddress(request.MintRequest.BuyerAddress)

	// Get a Token URI for the token.
	jsonPlain, tokenURI, err := createTokenURIforMintResponse(response.MintResponse)
	if err != nil {
		errMsg := fmt.Sprintf("error creating token URI: %v", err)
		h.logger.Debugf(errMsg) // TODO: @VjeraTurk change to Error after we stop using mocked uri data
		addErrorToResponseHeader(response, errMsg)
		return true
	}

	h.logger.Debugf("Token URI JSON: %s\n", jsonPlain)

	if response.MintResponse.BuyableUntil == nil || response.MintResponse.BuyableUntil.Seconds == 0 {
		response.MintResponse.BuyableUntil = timestamppb.New(time.Now().Add(h.buyableUntilDefault))
	}
	// MINT TOKEN
	txID, tokenID, err := h.mint(
		ctx,
		buyerAddress,
		tokenURI,
		big.NewInt(response.MintResponse.BuyableUntil.Seconds),
	)
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		h.logger.Errorf(errMessage)
		addErrorToResponseHeader(response, errMessage)
		return true
	}

	h.logger.Infof("NFT minted with txID: %s\n", txID)
	response.MintResponse.Header.Status = typesv1.StatusType_STATUS_TYPE_SUCCESS
	// Disable Linter: This code will be removed with the new mint logic and protocol
	response.MintResponse.BookingToken = &typesv1.BookingToken{TokenId: int32(tokenID.Int64())} // #nosec G115
	response.MintTransactionId = txID
	return false
}

func (h *evmResponseHandler) handleMintRequest(ctx context.Context, response *ResponseContent) bool {
	if response.MintResponse.Header == nil {
		response.MintResponse.Header = &typesv1.ResponseHeader{}
	}
	if response.MintTransactionId == "" {
		addErrorToResponseHeader(response, "missing mint transaction id")
		return true
	}

	value64 := uint64(response.BookingToken.TokenId)
	tokenID := new(big.Int).SetUint64(value64)

	txID, err := h.buy(ctx, tokenID)
	if err != nil {
		errMessage := fmt.Sprintf("error buying NFT: %v", err)
		h.logger.Errorf(errMessage)
		addErrorToResponseHeader(response, errMessage)
		return true
	}

	h.logger.Infof("Bought NFT (txID=%s) with ID: %s\n", txID, response.MintTransactionId)
	response.BuyTransactionId = txID
	return false
}

// Mints a BookingToken with the supplier private key and reserves it for the buyer address
// For testing you can use this uri: "data:application/json;base64,eyJuYW1lIjoiQ2FtaW5vIE1lc3NlbmdlciBCb29raW5nVG9rZW4gVGVzdCJ9Cg=="
func (h *evmResponseHandler) mint(
	ctx context.Context,
	reservedFor common.Address,
	uri string,
	expiration *big.Int,
) (string, *big.Int, error) {

	zeroAddress := common.HexToAddress("0x0000000000000000000000000000000000000000")

	tx, err := h.bookingService.MintBookingToken(
		reservedFor,
		uri,
		expiration,
		big.NewInt(0),
		zeroAddress)
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

	//
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
	h.logger.Infof("Gass %v", receipt)
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
func createTokenURIforMintResponse(mintResponse *bookv1.MintResponse) (string, string, error) {
	// TODO: What should we use for a token name? This will be shown in the UI of wallets, explorers etc.
	name := "CM Booking Token"

	// TODO: What should we use for a token description? This will be shown in the UI of wallets, explorers etc.
	description := "This NFT represents the booking with the specified attributes."

	// Dummy data
	date := "2024-06-24"

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

func addErrorToResponseHeader(response *ResponseContent, errMessage string) {
	response.MintResponse.Header.Status = typesv1.StatusType_STATUS_TYPE_FAILURE
	response.MintResponse.Header.Alerts = append(response.MintResponse.Header.Alerts, &typesv1.Alert{
		Message: errMessage,
		Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
	})
}

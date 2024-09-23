//  Copyright (C) 2022-2024, Chain4Travel AG. All rights reserved.
//  See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"

	config "github.com/chain4travel/camino-messenger-bot/config"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var _ ResponseHandler = (*evmResponseHandler)(nil)

type ResponseHandler interface {
	HandleResponse(ctx context.Context, msgType MessageType, request *RequestContent, response *ResponseContent)
	HandleRequest(ctx context.Context, msgType MessageType, request *RequestContent) error
}

func NewResponseHandler(ethClient *ethclient.Client, logger *zap.SugaredLogger, cfg *config.EvmConfig) (ResponseHandler, error) {
	abi, err := loadABI(cfg.BookingTokenABIFile)
	if err != nil {
		return nil, err
	}

	ecdsaPk, err := crypto.HexToECDSA(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Get Ethereum Address from private key
	cChainAddress := crypto.PubkeyToAddress(ecdsaPk.PublicKey)
	logger.Infof("C-Chain address: %s", cChainAddress)

	// Check supplier name and set default if empty
	supplierName := cfg.SupplierName
	if supplierName == "" {
		supplierName = "Default Supplier"
		logger.Infof("Supplier name cannot be empty. Setting name to: %s \n", supplierName)
	}

	return &evmResponseHandler{
		ethClient:           ethClient,
		logger:              logger,
		pk:                  ecdsaPk,
		tokenABI:            abi,
		cChainAddress:       cChainAddress,
		bookingTokenAddress: common.HexToAddress(cfg.BookingTokenAddress),
		supplierName:        supplierName,
		// Disable Linter: This code will be removed with the new BookingToken implementation
		buyableUntilDefault: time.Second * time.Duration(cfg.BuyableUntilDefault), // #nosec G115
	}, nil
}

type evmResponseHandler struct {
	ethClient           *ethclient.Client
	logger              *zap.SugaredLogger
	pk                  *ecdsa.PrivateKey
	tokenABI            *abi.ABI
	cChainAddress       common.Address
	bookingTokenAddress common.Address
	supplierName        string
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
		request.BuyerAddress = h.cChainAddress.Hex()
	}
	return nil
}

func (h *evmResponseHandler) handleMintResponse(ctx context.Context, response *ResponseContent, request *RequestContent) bool {
	if response.MintResponse.Header == nil {
		response.MintResponse.Header = &typesv1.ResponseHeader{}
	}

	packedData, err := h.tokenABI.Pack("getSupplierName", h.cChainAddress)
	if err != nil {
		errMsg := fmt.Sprintf("Error packing data: %v", err)
		h.logger.Errorf(errMsg)
		addErrorToResponseHeader(response, errMsg)
		return true
	}

	msg := ethereum.CallMsg{
		To:   &h.bookingTokenAddress,
		Data: packedData,
	}
	result, err := h.ethClient.CallContract(ctx, msg, nil)
	if err != nil {
		errMsg := fmt.Sprintf("Error calling contract: %v", err)
		h.logger.Errorf(errMsg)
		addErrorToResponseHeader(response, errMsg)
		return true
	}

	// Unpack the result
	var supplierName string
	err = h.tokenABI.UnpackIntoInterface(&supplierName, "getSupplierName", result)
	if err != nil {
		errMsg := fmt.Sprintf("Error unpacking result: %v", err)
		h.logger.Infof(errMsg)
		addErrorToResponseHeader(response, errMsg)
		return true
	}

	if supplierName != h.supplierName {
		h.logger.Debugf("Not registered with correct supplier name: %v != %v", supplierName, h.supplierName)
		h.logger.Debugf("Registering with supplierName: %v", h.supplierName)
		err := h.register(ctx)
		if err != nil {
			errMsg := fmt.Sprintf("error registering supplier: %v", err)
			h.logger.Debugf(errMsg)
			addErrorToResponseHeader(response, errMsg)
			return true
		}
	} else {
		h.logger.Debugf("Supplier is already registered with supplierName: %v", supplierName)
	}

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

// Registers a new supplier with the BookingToken contract
func (h *evmResponseHandler) register(ctx context.Context) error {
	nonce, err := h.ethClient.PendingNonceAt(ctx, h.cChainAddress)
	if err != nil {
		return err
	}

	gasPrice, err := h.ethClient.SuggestGasPrice(ctx)
	if err != nil {
		return err
	}

	packed, err := h.tokenABI.Pack("registerSupplier", h.supplierName)
	if err != nil {
		return err
	}

	gasLimit := uint64(170000)

	tx := types.NewTransaction(nonce, h.bookingTokenAddress, big.NewInt(0), gasLimit, gasPrice, packed)

	chainID, err := h.ethClient.NetworkID(ctx)
	if err != nil {
		return err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), h.pk)
	if err != nil {
		return err
	}

	err = h.ethClient.SendTransaction(ctx, signedTx)
	if err != nil {
		return err
	}

	h.logger.Infof("Transaction sent!\nTransaction hash: %s\n", signedTx.Hash().Hex())

	// Wait for transaction to be mined
	receipt, err := h.waitTransaction(ctx, signedTx)
	if err != nil {
		if gasLimit == receipt.GasUsed {
			h.logger.Errorf("Transaction Gas Limit reached. Please use shorter supplier name.\n")
		}
		return err
	}

	return nil
}

// Mints a BookingToken with the supplier private key and reserves it for the buyer address
// For testing you can use this uri: "data:application/json;base64,eyJuYW1lIjoiQ2FtaW5vIE1lc3NlbmdlciBCb29raW5nVG9rZW4gVGVzdCJ9Cg=="
func (h *evmResponseHandler) mint(
	ctx context.Context,
	reservedFor common.Address,
	uri string,
	expiration *big.Int,
) (string, *big.Int, error) {
	nonce, err := h.ethClient.PendingNonceAt(ctx, h.cChainAddress)
	if err != nil {
		return "", nil, err
	}

	gasPrice, err := h.ethClient.SuggestGasPrice(ctx)
	if err != nil {
		return "", nil, err
	}

	packed, err := h.tokenABI.Pack("safeMint", reservedFor, uri, expiration)
	if err != nil {
		return "", nil, err
	}

	// Set safe gas limit for now
	gasLimit := uint64(1200000)
	tx := types.NewTransaction(nonce, h.bookingTokenAddress, big.NewInt(0), gasLimit, gasPrice, packed)

	chainID, err := h.ethClient.NetworkID(ctx)
	if err != nil {
		return "", nil, err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), h.pk)
	if err != nil {
		return "", nil, err
	}

	err = h.ethClient.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", nil, err
	}

	h.logger.Infof("Transaction sent!\nTransaction hash: %s\n", signedTx.Hash().Hex())

	// Wait for transaction to be mined
	receipt, err := h.waitTransaction(ctx, signedTx)
	if err != nil {
		if gasLimit == receipt.GasUsed {
			h.logger.Infof("Transaction Gas Limit reached. Please check your inputs.\n")
		}
		return "", nil, err
	}

	// Define the TokenReservation structure
	type TokenReservation struct {
		ReservedFor         common.Address
		TokenID             *big.Int
		ExpirationTimestamp *big.Int
	}

	// Get the event signature hash
	event := h.tokenABI.Events["TokenReservation"]
	eventSignature := event.ID.Hex()

	var tokenID *big.Int

	// Iterate over the logs to find the event
	for _, vLog := range receipt.Logs {
		if vLog.Topics[0].Hex() == eventSignature {
			// Decode indexed parameters
			reservedFor := common.HexToAddress(vLog.Topics[1].Hex())
			tokenID = new(big.Int).SetBytes(vLog.Topics[2].Bytes())

			// Decode non-indexed parameters
			var reservation TokenReservation
			err := h.tokenABI.UnpackIntoInterface(&reservation, "TokenReservation", vLog.Data)
			if err != nil {
				return "", nil, err
			}
			reservation.ReservedFor = reservedFor
			reservation.TokenID = tokenID

			// Print the reservation details
			h.logger.Infof("Reservation Details:\n")
			h.logger.Infof("Token ID    : %s\n", reservation.TokenID.String())
			h.logger.Infof("Reserved For: %s\n", reservation.ReservedFor.Hex())
			h.logger.Infof("Expiration  : %s\n", reservation.ExpirationTimestamp.String())
		}
	}

	return signedTx.Hash().Hex(), tokenID, nil
}

// TODO @VjeraTurk code that creates and handles context should be improved, since its not doing job in separate goroutine,
// Buys a token with the buyer private key. Token must be reserved for the buyer address.
func (h *evmResponseHandler) buy(ctx context.Context, tokenID *big.Int) (string, error) {
	nonce, err := h.ethClient.PendingNonceAt(ctx, h.cChainAddress)
	if err != nil {
		return "", err
	}

	gasPrice, err := h.ethClient.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}

	packed, err := h.tokenABI.Pack("buy", tokenID)
	if err != nil {
		return "", err
	}

	// Set safe gas limit for now
	gasLimit := uint64(200000)

	tx := types.NewTransaction(nonce, h.bookingTokenAddress, big.NewInt(0), gasLimit, gasPrice, packed)

	chainID, err := h.ethClient.NetworkID(ctx)
	if err != nil {
		return "", err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), h.pk)
	if err != nil {
		return "", err
	}

	err = h.ethClient.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", err
	}

	h.logger.Infof("Transaction sent!\nTransaction hash: %s\n", signedTx.Hash().Hex())

	// Wait for transaction to be mined
	receipt, err := h.waitTransaction(ctx, signedTx)
	if err != nil {
		if gasLimit == receipt.GasUsed {
			h.logger.Infof("Transaction Gas Limit reached. Please check your inputs.\n")
		}
		return "", err
	}

	return signedTx.Hash().Hex(), nil
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

// Loads an ABI file
func loadABI(filePath string) (*abi.ABI, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	abi, err := abi.JSON(strings.NewReader(string(file)))
	if err != nil {
		return nil, err
	}
	return &abi, nil
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

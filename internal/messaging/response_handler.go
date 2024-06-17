//  Copyright (C) 2022-2024, Chain4Travel AG. All rights reserved.
//  See the file LICENSE for licensing terms.

package messaging

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	bookv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1alpha"
	typesv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1alpha"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"

	config "github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/evm"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var _ ResponseHandler = (*EvmResponseHandler)(nil)

type ResponseHandler interface {
	HandleResponse(ctx context.Context, msgType MessageType, request *RequestContent, response *ResponseContent)
	HandleRequest(ctx context.Context, msgType MessageType, request *RequestContent) error
}
type EvmResponseHandler struct {
	ethClient *ethclient.Client
	logger    *zap.SugaredLogger
	pk        *secp256k1.PrivateKey
	cfg       *config.EvmConfig
}

func (h *EvmResponseHandler) HandleResponse(ctx context.Context, msgType MessageType, request *RequestContent, response *ResponseContent) {
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

func (h *EvmResponseHandler) HandleRequest(_ context.Context, msgType MessageType, request *RequestContent) error {
	switch msgType { //nolint:gocritic
	case MintRequest:
		request.BuyerAddress = crypto.PubkeyToAddress(h.pk.ToECDSA().PublicKey).Hex()
	}
	return nil
}

func (h *EvmResponseHandler) handleMintResponse(_ context.Context, response *ResponseContent, request *RequestContent) bool {
	if response.MintResponse.Header == nil {
		response.MintResponse.Header = &typesv1alpha.ResponseHeader{}
	}
	abi, err := loadABI(h.cfg.BookingTokenABIFile)
	if err != nil {
		addErrorToResponseHeader(response, fmt.Sprintf("error loading ABI: %v", err))
		return true
	}
	address := crypto.PubkeyToAddress(h.pk.ToECDSA().PublicKey)

	bookingTokenAddress := common.HexToAddress(h.cfg.BookingTokenAddress)

	packedData, err := abi.Pack("getSupplierName", address)
	if err != nil {
		errMsg := fmt.Sprintf("Error packing data: %v", err)
		h.logger.Errorf(errMsg)
		addErrorToResponseHeader(response, errMsg)
		return true
	}

	msg := ethereum.CallMsg{
		To:   &bookingTokenAddress,
		Data: packedData,
	}
	result, err := h.ethClient.CallContract(context.Background(), msg, nil)
	if err != nil {
		errMsg := fmt.Sprintf("Error calling contract: %v", err)
		h.logger.Errorf(errMsg)
		addErrorToResponseHeader(response, errMsg)
		return true
	}

	// Unpack the result
	var supplierName string
	err = abi.UnpackIntoInterface(&supplierName, "getSupplierName", result)
	if err != nil {
		h.logger.Infof("Error unpacking result: %v", err)
		addErrorToResponseHeader(response, fmt.Sprintf("error unpacking result: %v", err))
		return true
	}

	if supplierName != h.cfg.SupplierName {
		h.logger.Debugf("Not registered with correct supplier name: %v != %v", supplierName, h.cfg.SupplierName)
		h.logger.Debugf("Registerring with supplierName: %v", h.cfg.SupplierName)
		err := register(h.ethClient, h.logger, abi, h.pk.ToECDSA(), h.cfg.SupplierName, bookingTokenAddress)
		if err != nil {
			addErrorToResponseHeader(response, fmt.Sprintf("error registering supplier: %v", err))
			return true
		}
	} else {
		h.logger.Debugf("Supplier is already registered with supplierName: %v", supplierName)
	}
	// TODO @evlekht ensure that request.MintRequest.BuyerAddress is c-chain address format, not x/p/t chain
	buyer := common.HexToAddress(request.MintRequest.BuyerAddress)

	h.logger.Debugf("abi: %v", abi)

	// TODO @evlekht unhardocoded, update the protocol to have those information in the mint response.
	// uri := "data:application/json;base64,eyJuYW1lIjoiQ2FtaW5vIE1lc3NlbmdlciBCb29raW5nVG9rZW4gVGVzdCJ9Cg=="

	// Get a Token URI for the token.
	jsonPlain, uri, err := createTokenURIforMintResponse(response.MintResponse)
	if err != nil {
		addErrorToResponseHeader(response, fmt.Sprintf("error creating token URI: %v", err))
		return true
	}

	h.logger.Debugf("Token URI JSON: %s\n", jsonPlain)

	if response.MintResponse.BuyableUntil == nil || response.MintResponse.BuyableUntil.Seconds == 0 {
		response.MintResponse.BuyableUntil = timestamppb.New(time.Now().Add(time.Second * time.Duration(h.cfg.BuyableUntilDefault)))
	}
	// MINT TOKEN
	txID, tokenID, err := mint(
		h.ethClient,
		h.logger,
		bookingTokenAddress,
		abi,
		h.pk.ToECDSA(),
		buyer,
		uri,
		big.NewInt(response.MintResponse.BuyableUntil.Seconds),
	)
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		if errors.Is(err, context.DeadlineExceeded) {
			errMessage = fmt.Sprintf("transaction deadline exceeded: %v", evm.ErrAwaitTxConfirmationTimeout)
		}
		addErrorToResponseHeader(response, errMessage)
		return true
	}

	h.logger.Infof("NFT minted with txID: %s\n", txID)
	response.MintResponse.Header.Status = typesv1alpha.StatusType_STATUS_TYPE_SUCCESS
	response.MintResponse.BookingToken = &typesv1alpha.BookingToken{TokenId: int32(tokenID.Int64())}
	response.MintTransactionId = txID
	return false
}

func (h *EvmResponseHandler) handleMintRequest(_ context.Context, response *ResponseContent) bool {
	if response.MintResponse.Header == nil {
		response.MintResponse.Header = &typesv1alpha.ResponseHeader{}
	}
	if response.MintTransactionId == "" {
		addErrorToResponseHeader(response, "missing mint transaction id")
		return true
	}

	abi, err := loadABI(h.cfg.BookingTokenABIFile)
	if err != nil {
		addErrorToResponseHeader(response, fmt.Sprintf("error loading ABI: %v", err))
		return true
	}

	value64 := uint64(response.BookingToken.TokenId)
	tokenID := new(big.Int).SetUint64(value64)

	bookingTokenAddress := common.HexToAddress(h.cfg.BookingTokenAddress)

	txID, err := buy(
		h.ethClient,
		h.logger,
		bookingTokenAddress,
		abi,
		h.pk.ToECDSA(),
		tokenID)
	if err != nil {
		errMessage := fmt.Sprintf("error buying NFT: %v", err)
		if errors.Is(err, context.DeadlineExceeded) {
			errMessage = fmt.Sprintf("%v", evm.ErrAwaitTxConfirmationTimeout)
		}
		addErrorToResponseHeader(response, errMessage)
		return true
	}

	h.logger.Infof("Bought NFT (txID=%s) with ID: %s\n", txID, response.MintTransactionId)
	response.BuyTransactionId = txID
	return false
}

// Loads an ABI file
func loadABI(filePath string) (abi.ABI, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return abi.ABI{}, err
	}
	return abi.JSON(strings.NewReader(string(file)))
}

// Registers a new supplier with the BookingToken contract
func register(client *ethclient.Client, logger *zap.SugaredLogger, contractABI abi.ABI, privateKey *ecdsa.PrivateKey, supplierName string, bookingTokenAddress common.Address) error {
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := client.PendingNonceAt(context.Background(), address)
	if err != nil {
		return err
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}

	// Check supplier name and set default if empty
	if supplierName == "" {
		defaultSupplierName := "Default Supplier"
		logger.Infof("Supplier name cannot be empty. Setting name to: %s \n", defaultSupplierName)
		supplierName = defaultSupplierName
	}

	packed, err := contractABI.Pack("registerSupplier", supplierName)
	if err != nil {
		return err
	}

	gasLimit := uint64(170000)

	tx := types.NewTransaction(nonce, bookingTokenAddress, big.NewInt(0), gasLimit, gasPrice, packed)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}

	logger.Infof("Transaction sent!\nTransaction hash: %s\n", signedTx.Hash().Hex())

	// Wait for transaction to be mined
	receipt, err := waitTransaction(context.Background(), client, logger, signedTx)
	if err != nil {
		if gasLimit == receipt.GasUsed {
			logger.Errorf("Transaction Gas Limit reached. Please use shorter supplier name.\n")
		}
		return err
	}

	return nil
}

// Mints a BookingToken with the supplier private key and reserves it for the buyer address
// For testing you can use this uri: "data:application/json;base64,eyJuYW1lIjoiQ2FtaW5vIE1lc3NlbmdlciBCb29raW5nVG9rZW4gVGVzdCJ9Cg=="
func mint(
	client *ethclient.Client,
	logger *zap.SugaredLogger,
	bookingTokenAddress common.Address,
	contractABI abi.ABI,
	privateKey *ecdsa.PrivateKey,
	reservedFor common.Address,
	uri string,
	expiration *big.Int,
) (string, *big.Int, error) {
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := client.PendingNonceAt(context.Background(), address)
	if err != nil {
		return "", nil, err
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", nil, err
	}

	packed, err := contractABI.Pack("safeMint", reservedFor, uri, expiration)
	if err != nil {
		return "", nil, err
	}

	// Set safe gas limit for now
	gasLimit := uint64(1200000)
	tx := types.NewTransaction(nonce, bookingTokenAddress, big.NewInt(0), gasLimit, gasPrice, packed)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return "", nil, err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", nil, err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", nil, err
	}

	logger.Infof("Transaction sent!\nTransaction hash: %s\n", signedTx.Hash().Hex())

	// Wait for transaction to be mined
	receipt, err := waitTransaction(context.Background(), client, logger, signedTx)
	if err != nil {
		if gasLimit == receipt.GasUsed {
			logger.Infof("Transaction Gas Limit reached. Please check your inputs.\n")
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
	event := contractABI.Events["TokenReservation"]
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
			err := contractABI.UnpackIntoInterface(&reservation, "TokenReservation", vLog.Data)
			if err != nil {
				return "", nil, err
			}
			reservation.ReservedFor = reservedFor
			reservation.TokenID = tokenID

			// Print the reservation details
			logger.Infof("Reservation Details:\n")
			logger.Infof("Token ID    : %s\n", reservation.TokenID.String())
			logger.Infof("Reserved For: %s\n", reservation.ReservedFor.Hex())
			logger.Infof("Expiration  : %s\n", reservation.ExpirationTimestamp.String())
		}
	}

	return signedTx.Hash().Hex(), tokenID, nil
}

// Buys a token with the buyer private key. Token must be reserved for the buyer address.
func buy(client *ethclient.Client, logger *zap.SugaredLogger, bookingTokenAddress common.Address, contractABI abi.ABI, privateKey *ecdsa.PrivateKey, tokenID *big.Int) (string, error) {
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := client.PendingNonceAt(context.Background(), address)
	if err != nil {
		return "", err
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", err
	}

	packed, err := contractABI.Pack("buy", tokenID)
	if err != nil {
		return "", err
	}

	// Set safe gas limit for now
	gasLimit := uint64(200000)

	tx := types.NewTransaction(nonce, bookingTokenAddress, big.NewInt(0), gasLimit, gasPrice, packed)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return "", err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", err
	}

	logger.Infof("Transaction sent!\nTransaction hash: %s\n", signedTx.Hash().Hex())

	// Wait for transaction to be mined
	receipt, err := waitTransaction(context.Background(), client, logger, signedTx)
	if err != nil {
		if gasLimit == receipt.GasUsed {
			logger.Infof("Transaction Gas Limit reached. Please check your inputs.\n")
		}
		return "", err
	}

	return signedTx.Hash().Hex(), nil
}

// Waits for a transaction to be mined
func waitTransaction(ctx context.Context, b bind.DeployBackend, logger *zap.SugaredLogger, tx *types.Transaction) (receipt *types.Receipt, err error) {
	logger.Infof("Waiting for transaction to be mined...\n")

	receipt, err = bind.WaitMined(ctx, b, tx)
	if err != nil {
		return receipt, err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return receipt, fmt.Errorf("transaction failed: %v", receipt)
	}

	logger.Infof("Successfully mined. Block Nr: %s Gas used: %d\n", receipt.BlockNumber, receipt.GasUsed)

	return receipt, nil
}

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
func createTokenURIforMintResponse(mintResponse *bookv1alpha.MintResponse) (string, string, error) {
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
			Value:     mintResponse.GetMintId(),
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
	response.MintResponse.Header.Status = typesv1alpha.StatusType_STATUS_TYPE_FAILURE
	response.MintResponse.Header.Alerts = append(response.MintResponse.Header.Alerts, &typesv1alpha.Alert{
		Message: errMessage,
		Type:    typesv1alpha.AlertType_ALERT_TYPE_ERROR,
	})
}

func NewResponseHandler(ethClient *ethclient.Client, logger *zap.SugaredLogger, pk *secp256k1.PrivateKey, cfg *config.EvmConfig) *EvmResponseHandler {
	return &EvmResponseHandler{ethClient: ethClient, logger: logger, pk: pk, cfg: cfg}
}

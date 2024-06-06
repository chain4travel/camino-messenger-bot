/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */

package messaging

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	//"crypto/ecdsa"

	//"errors"
	//"fmt"
	//"math/big"

	//"github.com/ethereum/go-ethereum/accounts/abi"
	//"github.com/ethereum/go-ethereum/common"
	//"github.com/ethereum/go-ethereum/core/types"
	//"github.com/ethereum/go-ethereum/crypto"

	typesv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1alpha"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/spf13/viper"

	//"github.com/chain4travel/camino-messenger-bot/internal/evm"

	"github.com/chain4travel/camino-messenger-bot/internal/evm"
	"github.com/chain4travel/caminotravelvm/actions"

	//"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var _ ResponseHandler = (*EvmResponseHandler)(nil)

type ResponseHandler interface {
	HandleResponse(ctx context.Context, msgType MessageType, request *RequestContent, response *ResponseContent)
}
type EvmResponseHandler struct {
	ethClient *ethclient.Client
	logger    *zap.SugaredLogger
	pk        *secp256k1.PrivateKey
}

func (h *EvmResponseHandler) HandleResponse(ctx context.Context, msgType MessageType, request *RequestContent, response *ResponseContent) {
	switch msgType {
	case MintRequest: // distributor will post-process a mint request to buy the returned NFT
		if h.handleMintRequest(ctx, response) {
			return
		}
	case MintResponse: // provider will act upon receiving a mint response by minting an NFT
		if h.handleMintResponse(ctx, response, request) {
			return
		}
	}
}

func (h *EvmResponseHandler) handleMintResponse(ctx context.Context, response *ResponseContent, request *RequestContent) bool {
	//TODO: alter to use booking token
	// owner := h.ethClient.Address()
	// if response.MintResponse.Header == nil {
	// 	response.MintResponse.Header = &typesv1alpha.ResponseHeader{}
	// }
	// TODO @evlekht ensure that request.MintRequest.BuyerAddress is c-chain address format, not x/p/t chain
	buyer := common.HexToAddress(request.MintRequest.BuyerAddress)
	// price, err := strconv.Atoi(response.MintResponse.Price.Value)
	// if err != nil {
	// 	addErrorToResponseHeader(response, fmt.Sprintf("error parsing price value: %v", err))
	// 	return true
	// }
	abi, err := loadABI("/home/dev/Documents/Chain4Travel/camino-messenger-bot/abi")
	if err != nil {
		addErrorToResponseHeader(response, fmt.Sprintf("error loading ABI: %v", err))
		return true
	}

	// TODO @evlekht unhardocoded, figure out what it is at all
	uri := "data:application/json;base64,eyJuYW1lIjoiQ2FtaW5vIE1lc3NlbmdlciBCb29raW5nVG9rZW4gVGVzdCJ9Cg=="
	txID, err := mint(
		h.ethClient,
		abi,
		h.pk.ToECDSA(),
		buyer,
		uri,
		big.NewInt(response.MintResponse.BuyableUntil.Seconds),
	)
	if err != nil {
		errMessage := fmt.Sprintf("error minting NFT: %v", err)
		if errors.Is(err, context.DeadlineExceeded) {
			errMessage = fmt.Sprintf("%v: %v", evm.ErrAwaitTxConfirmationTimeout)
		}
		addErrorToResponseHeader(response, errMessage)
		return true
	}

	h.logger.Infof("NFT minted with txID: %s\n", txID)
	response.MintResponse.Header.Status = typesv1alpha.StatusType_STATUS_TYPE_SUCCESS
	response.MintTransactionId = txID
	return false
}

func (h *EvmResponseHandler) handleMintRequest(ctx context.Context, response *ResponseContent) bool {
	//TODO: alter to use booking token
	if response.MintResponse.Header == nil {
		response.MintResponse.Header = &typesv1alpha.ResponseHeader{}
	}
	if response.MintTransactionId == "" {
		addErrorToResponseHeader(response, "missing mint transaction id")
		return true
	}
	mintID, err := ids.FromString(response.MintTransactionId)
	if err != nil {
		addErrorToResponseHeader(response, fmt.Sprintf("error parsing mint transaction id: %v", err))
		return true
	}
	success, txID, err := h.ethClient.SendTxAndWait(ctx, transferNFTAction(h.ethClient.Address(), mintID))
	if err != nil {
		errMessage := fmt.Sprintf("error buying NFT: %v", err)
		if errors.Is(err, context.DeadlineExceeded) {
			errMessage = fmt.Sprintf("%v", evm.ErrAwaitTxConfirmationTimeout)
		}
		addErrorToResponseHeader(response, errMessage)
		return true
	}
	if !success {
		addErrorToResponseHeader(response, "buying NFT failed")
		return true
	}

	h.logger.Infof("Bought NFT (txID=%s) with ID: %s\n", txID, mintID)
	response.BuyTransactionId = txID.String()
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

// Mints a BookingToken with the supplier private key and reserves it for the buyer address
// For testing you can use this uri: "data:application/json;base64,eyJuYW1lIjoiQ2FtaW5vIE1lc3NlbmdlciBCb29raW5nVG9rZW4gVGVzdCJ9Cg=="
func mint(
	client *ethclient.Client,
	contractABI abi.ABI,
	privateKey *ecdsa.PrivateKey,
	reservedFor common.Address,
	uri string,
	expiration *big.Int,
) (string, error) {
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := client.PendingNonceAt(context.Background(), address)
	if err != nil {
		return "", err
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", err
	}

	packed, err := contractABI.Pack("safeMint", reservedFor, uri, expiration)
	if err != nil {
		return "", err
	}

	gasLimit := uint64(600000)

	tx := types.NewTransaction(nonce, common.HexToAddress(viper.GetString("booking_token_addr")), big.NewInt(0), gasLimit, gasPrice, packed)

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

	fmt.Printf("Transaction sent!\nTransaction hash: %s\n", signedTx.Hash().Hex())

	// Wait for transaction to be mined
	receipt, err := waitTransaction(context.Background(), client, signedTx)
	if err != nil {
		if gasLimit == receipt.GasUsed {
			fmt.Printf("Transaction Gas Limit reached. Please check your inputs.\n")
		}
		return "", err
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

	// Iterate over the logs to find the event
	for _, vLog := range receipt.Logs {
		if vLog.Topics[0].Hex() == eventSignature {
			// Decode indexed parameters
			reservedFor := common.HexToAddress(vLog.Topics[1].Hex())
			tokenId := new(big.Int).SetBytes(vLog.Topics[2].Bytes())

			// Decode non-indexed parameters
			var reservation TokenReservation
			err := contractABI.UnpackIntoInterface(&reservation, "TokenReservation", vLog.Data)
			if err != nil {
				return "", err
			}
			reservation.ReservedFor = reservedFor
			reservation.TokenID = tokenId

			// Print the reservation details
			fmt.Printf("Reservation Details:\n")
			fmt.Printf("Token ID    : %s\n", reservation.TokenID.String())
			fmt.Printf("Reserved For: %s\n", reservation.ReservedFor.Hex())
			fmt.Printf("Expiration  : %s\n", reservation.ExpirationTimestamp.String())
		}
	}

	return signedTx.Hash().Hex(), nil
}

// Waits for a transaction to be mined
func waitTransaction(ctx context.Context, b bind.DeployBackend, tx *types.Transaction) (receipt *types.Receipt, err error) {

	fmt.Printf("Waiting for transaction to be mined...\n")

	receipt, err = bind.WaitMined(ctx, b, tx)
	if err != nil {
		return receipt, err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return receipt, fmt.Errorf("transaction failed: %v", receipt)
	}

	fmt.Printf("Successfully mined. Block Nr: %s Gas used: %d\n", receipt.BlockNumber, receipt.GasUsed)

	return receipt, nil
}

func addErrorToResponseHeader(response *ResponseContent, errMessage string) {
	response.MintResponse.Header.Status = typesv1alpha.StatusType_STATUS_TYPE_FAILURE
	response.MintResponse.Header.Alerts = append(response.MintResponse.Header.Alerts, &typesv1alpha.Alert{
		Message: errMessage,
		Type:    typesv1alpha.AlertType_ALERT_TYPE_ERROR,
	})
}

func NewResponseHandler(ethClient *ethclient.Client, logger *zap.SugaredLogger, pk *secp256k1.PrivateKey) *EvmResponseHandler {
	return &EvmResponseHandler{ethClient: ethClient, logger: logger, pk: pk}
}

func createNFTAction(owner, buyer codec.Address, purchaseExpiration, price uint64, metadata string) chain.Action {
	return &actions.CreateNFT{
		Owner:                owner,
		Issuer:               owner,
		Buyer:                buyer,
		PurchaseExpiration:   purchaseExpiration,
		Asset:                ids.ID{},
		Price:                price,
		CancellationPolicies: actions.CancellationPolicies{},
		Metadata:             []byte(metadata),
	}
}

func transferNFTAction(newOwner codec.Address, nftID ids.ID) chain.Action {
	return &actions.TransferNFT{
		To:             newOwner,
		ID:             nftID,
		OnChainPayment: false, // TODO change based on tchain configuration
		Memo:           nil,
	}
}

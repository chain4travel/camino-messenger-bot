package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os/signal"
	"syscall"

	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {

	// Set your CM Account address here
	cmAccountAddr := common.HexToAddress("0x1B9dac1f224DAEA0bB1Ee5e01Eb51E4551CEb7AB")

	reservedFor := common.HexToAddress("0xB682106bEbf1017D8a147959Ed508768670a3162")
	uri := "TOKEN_URI"
	expirationTimestamp := big.NewInt(1)
	price := &big.Int{}

	// Set BookingToken address here
	//bookingTokenAddr := common.HexToAddress("0xe55E387F5474a012D1b048155E25ea78C7DBfBBC")

	// Initialize Ethereum, default value here is for Columbus testnet
	client, err := ethclient.Dial("wss://columbus.camino.network/ext/bc/C/ws")
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum client: %v", err)
	}
	//cmAccountInstance.mintBookingToken(address reservedFor, string memory uri, uint256 expirationTimestamp, uint256 price, IERC20 paymentToken)`

	cmAccount, err := cmaccount.NewCmaccount(cmAccountAddr, client)
	//cmAccount, err := cmaccount.NewCmaccountCaller(cmAccountAddr, client)

	if err != nil {
		log.Fatalf("Failed to instance: %v", err)
	}

	// TODO: @VjeraTurk how to call the mintBookingToken

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	nonce, err := client.PendingNonceAt(ctx, cmAccountAddr)
	if err != nil {
		return
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return
	}

	gasLimit := uint64(1200000)

	zeroAddress := common.Address{}

	chainID, err := client.NetworkID(ctx)

	pk := new(secp256k1.PrivateKey)
	// UnmarshalText expects the private key in quotes
	if err := pk.UnmarshalText([]byte("\"" + "PrivateKey-5wZusnKXrKjXYHA2XW35LoX8P7oVusc2kqjpecbwEsXx6Aygc" + "\"")); err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}
	ecdsaPk := pk.ToECDSA()

	signerFn := createSignerFn(ecdsaPk, chainID)
	tx, err := cmAccount.MintBookingToken(&bind.TransactOpts{
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Nonce:    new(big.Int).SetUint64(nonce),
		Signer:   signerFn,
	}, reservedFor, uri, expirationTimestamp, price, zeroAddress)
	if err != nil {
		log.Fatalf("Failed Minting: %v", err)
	}
	fmt.Printf("Transaction sent: %s", tx.Hash().Hex())

	//not applicable
	//cmAccount.mintBookingToken(reservedFor, uri, expirationTimestamp, price, zeroAddress)
	//cmAccount.MintBookingToken(reservedFor, uri, expirationTimestamp, price, zeroAddress)
	defer stop()
}

func createSignerFn(privateKey *ecdsa.PrivateKey, chainID *big.Int) bind.SignerFn {
	// Initialize EIP155Signer with the chain ID
	signer := types.NewEIP155Signer(chainID)

	return func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
		// Sign the transaction using the private key and EIP155Signer
		signedTx, err := types.SignTx(tx, signer, privateKey)
		if err != nil {
			return nil, err
		}
		return signedTx, nil
	}
}

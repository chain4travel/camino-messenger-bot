package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os/signal"
	"syscall"
	"time"

	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {

	// Set your CM Account address here
	cmAccountAddr := common.HexToAddress("<CM_ACCOUNT_ADDRESS>")

	reservedFor := common.HexToAddress("<CM_ACCOUNT_ADDRESS>")
	uri := "TOKEN_URI"
	expirationTimestamp := big.NewInt(time.Now().Add(time.Hour).Unix())
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	zeroAddress := common.Address{}

	chainID, err := client.NetworkID(ctx)

	ecdsaPk, err := crypto.HexToECDSA("<PRIVATE_KEY_HEX>")
	if err != nil {
		log.Fatalf("Failed to convert private key: %v", err)
	}

	transactOpts, err := bind.NewKeyedTransactorWithChainID(ecdsaPk, chainID)
	//fmt.Print("%s", pk)
	if err != nil {
		log.Fatalf("failed to create transactor")
	}
	tx, err := cmAccount.MintBookingToken(
		transactOpts,
		reservedFor,
		uri,
		expirationTimestamp,
		price,
		zeroAddress)
	if err != nil {
		log.Fatalf("Failed Minting: %v", err)
	}
	fmt.Printf("Transaction sent: %s", tx.Hash().Hex())

	bt, err := bookingtoken.NewBookingtoken(common.HexToAddress("0xe55E387F5474a012D1b048155E25ea78C7DBfBBC"), client)

	mintReceipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		log.Fatalf("Failed to wait for mint transaction to be mined: %v", err)
	}

	tokenID := big.NewInt(0)

	for _, mLog := range mintReceipt.Logs {
		event, err := bt.ParseTokenReserved(*mLog)
		if err == nil {
			tokenID = event.TokenId
			fmt.Printf("[TokenReserved] TokenID: %s ReservedFor: %s Price: %s, PaymentToken: %s", event.TokenId, event.ReservedFor, event.Price, event.PaymentToken)
		}
	}
	log.Print(tokenID)
	defer stop()
}

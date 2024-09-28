package main

import (
	"context"
	"flag"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"

	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
)

// Simple usage example for the BookingService
func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	sugar.Info("Starting Mint & Buy Example...")

	cmAccountAddrString := flag.String("cmaccount", "", "CMAccount Address. Ex: 0x....")
	// Take private key from command line
	pkString := flag.String("pk", "", "Private Key without the 0x notation")
	flag.Parse()

	// Set your CM Account address here
	cmAccountAddr := common.HexToAddress(*cmAccountAddrString)

	if cmAccountAddr == (common.Address{}) {
		sugar.Fatalf("CMAccount address cannot be empty")
	}

	if *pkString == "" {
		sugar.Fatalf("Private key cannot be empty")
	}

	sugar.Info("CMAccount address: ", cmAccountAddr.String())

	// Initialize client, default value here is for Columbus testnet
	client, err := ethclient.Dial("wss://columbus.camino.network/ext/bc/C/ws")
	if err != nil {
		sugar.Fatalf("Failed to connect to Ethereum client: %v", err)
	}

	pk, err := crypto.HexToECDSA(*pkString)
	if err != nil {
		sugar.Fatalf("Failed to parse private key: %v", err)
	}

	sugar.Info("Creating Booking Service...")
	bs, err := booking.NewService(cmAccountAddr, pk, client, sugar)
	if err != nil {
		sugar.Fatalf("Failed to create Booking Service: %v", err)
	}

	bt, err := bookingtoken.NewBookingtoken(common.HexToAddress("0xe55E387F5474a012D1b048155E25ea78C7DBfBBC"), client)

	// token uri
	tokenURI := "data:application/json;base64,eyJuYW1lIjoiYm90IGNtYWNjb3VudCBwa2cgYm9va2luZyB0b2tlbiB0ZXN0In0K"

	// expiration timestamp
	expiration := big.NewInt(time.Now().Add(time.Hour).Unix())

	// price
	price := big.NewInt(0)

	// Mint a new booking token
	//
	// Note that here we used the same CM Account address that is minting the
	// BookingToken as the `reservedFor` address. This is only done as an example,
	// because we will buy the token with the same CM Account.
	//
	// Under normal circumstances the reservedFor address should be another CM
	// Account address, generally the distributor's CM account address. And the
	// distributor should buy the token.
	mintTx, err := bs.MintBookingToken(
		cmAccountAddr, // reservedFor address
		tokenURI,
		expiration,
		price,
		common.HexToAddress("0x0000000000000000000000000000000000000000"), // payment token
	)
	if err != nil {
		sugar.Fatalf("Failed to mint booking token: %v", err)
	}

	// Wait for the transaction to be mined and get the receipt
	sugar.Info("Waiting for mint transaction to be mined...")
	mintReceipt, err := bind.WaitMined(context.Background(), client, mintTx)
	if err != nil {
		sugar.Fatalf("Failed to wait for mint transaction to be mined: %v", err)
	}

	tokenID := big.NewInt(0)

	for _, mLog := range mintReceipt.Logs {
		event, err := bt.ParseTokenReserved(*mLog)
		if err == nil {
			tokenID = event.TokenId
			sugar.Infof("[TokenReserved] TokenID: %s ReservedFor: %s Price: %s, PaymentToken: %s", event.TokenId, event.ReservedFor, event.Price, event.PaymentToken)
		}
	}

	// Sleep 5 seconds
	sugar.Info("Sleeping for 5 seconds...")
	time.Sleep(5 * time.Second)

	// Buy a new booking token
	//
	// This function should be called by an address that has the
	// BOOKING_OPERATOR_ROLE role in the `reservedFor` CMAccount.
	//
	// When bots are added to a CMAccount by the `addMessengerBot(address)`
	// function, this role is granted to the bot.
	buyTx, err := bs.BuyBookingToken(
		tokenID,
	)
	if err != nil {
		sugar.Fatalf("Failed to buy booking token: %v", err)
	}

	// Wait for the transaction to be mined and get the receipt
	sugar.Info("Waiting for buy transaction to be mined...")
	buyReceipt, err := bind.WaitMined(context.Background(), client, buyTx)
	if err != nil {
		sugar.Fatalf("Failed to wait for buy transaction to be mined: %v", err)
	}

	// Parse the logs
	for _, bLog := range buyReceipt.Logs {
		event, err := bt.ParseTokenBought(*bLog)
		if err == nil {
			sugar.Infof("[TokenBought] TokenID: %s, Buyer: %s", event.TokenId, event.Buyer)
		}
	}
}

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/chain4travel/camino-messenger-bot/pkg/events"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

func main() {
	// Variables

	// Set your CM Account address here
	cmAccountAddr := common.HexToAddress("0xd41786599F2B225A5A1eA35cDc4A2a6Fa9E92BeA")

	// Set BookingToken address here
	bookingTokenAddr := common.HexToAddress("0xe55E387F5474a012D1b048155E25ea78C7DBfBBC")

	// Initialize Ethereum, default value here is for Columbus testnet
	client, err := ethclient.Dial("wss://columbus.camino.network/ext/bc/C/ws")
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum client: %v", err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	// Create EventListener
	sugar.Info("Creating EventListener...")
	el := events.NewEventListener(client, sugar)

	// Register ServiceAdded handler, this is a cancel example. Will be cancelled when a service remove event received. below.
	cancelServiceAdded, err := el.RegisterServiceAddedHandler(cmAccountAddr, nil, func(event interface{}) {
		e := event.(*cmaccount.CmaccountServiceAdded)
		sugar.Infof("Received ServiceAdded event: \n CMAccount: %s \n ServiceName: %s", cmAccountAddr, e.ServiceName)
	})
	if err != nil {
		log.Fatalf("Failed to register handler: %v", err)
	}

	_, err = el.RegisterServiceRemovedHandler(cmAccountAddr, nil, func(event interface{}) {
		e := event.(*cmaccount.CmaccountServiceRemoved)
		sugar.Infof("Received ServiceRemoved event: \n CMAccount: %s \n ServiceName: %s", cmAccountAddr, e.ServiceName)
		sugar.Info("Cancelling ServiceAdded handler...")
		// When you want to stop listening
		cancelServiceAdded()
	})
	if err != nil {
		log.Fatalf("Failed to register handler: %v", err)
	}

	_, err = el.RegisterServiceFeeUpdatedHandler(cmAccountAddr, nil, func(event interface{}) {
		e := event.(*cmaccount.CmaccountServiceFeeUpdated)
		sugar.Infof("Received ServiceFeeUpdated event: \n CMAccount: %s \n ServiceName: %s", cmAccountAddr, e.ServiceName)
	})
	if err != nil {
		log.Fatalf("Failed to register handler: %v", err)
	}

	_, err = el.RegisterTokenBoughtHandler(bookingTokenAddr, nil, nil, func(event interface{}) {
		e := event.(*bookingtoken.BookingtokenTokenReserved)
		sugar.Infof("Received TokenBought event: \n BookingToken: %s \n TokenID: %s \n Buyer: %s \n Price: %s, \n PaymentToken: %s \n Expiration: %s", bookingTokenAddr, e.TokenId, e.Supplier, e.Price, e.PaymentToken, e.ExpirationTimestamp)
	})
	if err != nil {
		log.Fatalf("Failed to register handler: %v", err)
	}

	_, err = el.RegisterTokenReservedHandler(bookingTokenAddr, nil, nil, nil, func(event interface{}) {
		e := event.(*bookingtoken.BookingtokenTokenReserved)
		sugar.Infof("Received TokenReserved event: \n BookingToken: %s \n TokenID: %s \n ReservedFor: %s \n Supplier: %s \n Price: %s \n PaymentToken: %s \n Expiration: %s", bookingTokenAddr, e.TokenId, e.ReservedFor, e.Supplier, e.Price, e.PaymentToken, e.ExpirationTimestamp)
	})
	if err != nil {
		log.Fatalf("Failed to register handler: %v", err)
	}

	// Block forever until Ctrl+C is pressed
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	sugar.Info("Exiting application...")
}

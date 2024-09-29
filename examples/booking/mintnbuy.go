package main

import (
	"context"
	"flag"
	"log"
	"math/big"
	"time"

	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"

	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
)

var zeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")

// https://columbus.caminoscan.com/token/0x5b1c852dad36854B0dFFF61d2C13F108D8E01975
// https://caminoscan.com/token/0x026816DF82F78882DaC9370a35c497C254Ebd88E
var eurshToken = common.HexToAddress("0x5b1c852dad36854B0dFFF61d2C13F108D8E01975")

var polygonToken = common.HexToAddress("0x0000000000000000000000000000000000001010")

// https://polygonscan.com/token/0x1bfd67037b42cf73acf2047067bd4f2c47d9bfd6
var wBtcToken = common.HexToAddress("0x1bfd67037b42cf73acf2047067bd4f2c47d9bfd6") // not on Camino

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

	var paymentToken common.Address = zeroAddress
	var bigIntPrice *big.Int
	var price *typesv2.Price
	// https://polygonscan.com/unitconverter
	// ### Simple Price type message Price
	//
	// Value of the price, this should be an integer converted to string.
	//
	// This field is a string intentionally. Because the currency can be a crypto
	// currency, we need a reliable way to represent big integers as most of the crypto
	// currencies have 18 decimals precision.
	//
	// Definition of the price message: The combination of "value" and "decimals" fields
	// express always the value of the currency, not of the fraction of the currency [
	// ETH not wei, CAM and not aCAM, BTC and not Satoshi, EUR not EUR-Cents ] Be aware
	// that partners should not do rounding with crypto currencies.
	//
	// price
	// Example implementations: off-chain payment of 100 € or 100 $:
	// value=10000
	// decimals=2
	// iso_currency=EUR or USD

	priceEUR := &typesv2.Price{
		Value:    "10000",
		Decimals: 2,
		Currency: &typesv2.Currency{
			Currency: &typesv2.Currency_IsoCurrency{
				IsoCurrency: typesv2.IsoCurrency_ISO_CURRENCY_EUR,
			},
		},
	}

	// On-chain payment of 100.65 EURSH
	// value=10065
	// decimals=2
	// contract_address=0x...
	//	this currency has 5 decimals on Columbus and conclusively to create the
	//	transaction value, 10065 must be divided by 10^2 = 100.65 EURSH and created in
	//	its smallest fraction by multiplying  100.65 EURSH * 10^5 => 10065000 (example
	//	conversion to bigint without losing accuracy: bigint(10065) * 10^(5-2))

	priceEURSH := &typesv2.Price{
		Value:    "10065",
		Decimals: 2,
		Currency: &typesv2.Currency{
			Currency: &typesv2.Currency_TokenCurrency{
				TokenCurrency: &typesv2.TokenCurrency{
					ContractAddress: eurshToken.Hex(),
				},
			},
		},
	}

	// TODO: call decimals on eurshToken (should get 5)

	// On-chain payment of 0.0065 BTC
	//  value=65
	//  decimals=4
	//  contract_address=0x... Using
	//
	//	the contract address, we get the decimals decimals and the currency name or
	//	abbreviation: 8 decimals & WBTC Because we see 4 decimals specified in the
	//	message we divide 65 by 10^4 == 0.0065 WBTC (for showing in the front-end UIs)
	//
	//	This currency has 8 decimals on-chain and conclusively to use the value of
	//	0.0065 for on-chain operations must be converted to big integer as bigint(65) *
	//	10^(8-4) == 650000

	priceBTC := &typesv2.Price{
		Value:    "65",
		Decimals: 4,
		Currency: &typesv2.Currency{
			Currency: &typesv2.Currency_TokenCurrency{
				TokenCurrency: &typesv2.TokenCurrency{},
			},
		},
	}
	// On-chain payment of 1 nCAM value=1 decimals=9 this currency has denominator 18 on
	//
	//	Columbus and conclusively to mint the value of 1 nCam must be divided by 10^9 =
	//	0.000000001 CAM and minted in its smallest fraction by multiplying 0.000000001 *
	//	10^18 => 1000000000 aCAM

	priceCAM := &typesv2.Price{
		Value:    "1",
		Decimals: 9,
		Currency: &typesv2.Currency{
			Currency: &typesv2.Currency_NativeToken{
				NativeToken: &emptypb.Empty{},
			},
		},
	}

	sugar.Infof("%v %v %v %v", priceEUR, priceEURSH, priceBTC, priceCAM)
	sugar.Infof("%v", price)

	// bigIntPrice, _ = bs.ConvertPriceToBigInt(*priceEURSH, int32(5))
	// bigIntPrice, _ = bs.ConvertPriceToBigInt(*priceCAM, int32(18))

	paymentToken = zeroAddress
	bigIntPrice = big.NewInt(0)
	// price = priceEURSH
	// price = priceBTC // case of unsupported token?
	price = priceCAM

	switch price.Currency.Currency.(type) {
	case *typesv2.Currency_NativeToken:
		bigIntPrice, err = bs.ConvertPriceToBigInt(*price, int32(18)) //CAM uses 18 decimals
		if err != nil {
			sugar.Errorf("Failed to convert price to big.Int: %v", err)
			return
		}
		sugar.Infof("Converted the price big.Int: %v", bigIntPrice)
		paymentToken = zeroAddress
	case *typesv2.Currency_TokenCurrency:
		// Add logic to handle TokenCurrency
		// if contract address is zeroAddress, then it is native token
		sugar.Infof("TokenCurrency not supported yet")
		return
	case *typesv2.Currency_IsoCurrency:
		// Add logic to handle IsoCurrency
		sugar.Infof("IsoCurrency not supported yet")
		return
	}

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
		bigIntPrice,
		paymentToken,
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
			sugar.Infof("[TokenReserved] TokenID: %s ReservedFor: %s Price: %s, PaymentToken: %s,  TokenId:  %s", event.TokenId, event.ReservedFor, event.Price, event.PaymentToken, tokenID)
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

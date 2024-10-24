package main

import (
	"context"
	"flag"
	"log"
	"math/big"
	"time"

	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"

	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts"
	erc20 "github.com/chain4travel/camino-messenger-bot/pkg/erc20"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtokenv2"
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

	erc20, err := erc20.NewERC20Service(client, 100)
	if err != nil {
		sugar.Fatalf("Failed to create ERC20 service: %v", err)
	}

	cmAccounts, err := cmaccounts.NewService(sugar, 100, client)
	if err != nil {
		sugar.Fatalf("Failed to create CMAccounts service: %v", err)
	}

	pk, err := crypto.HexToECDSA(*pkString)
	if err != nil {
		sugar.Fatalf("Failed to parse private key: %v", err)
	}

	sugar.Info("Creating Booking Service...")
	bs, err := booking.NewService(cmAccountAddr, pk, client, sugar, erc20, cmAccounts)
	if err != nil {
		sugar.Fatalf("Failed to create Booking Service: %v", err)
	}

	bt, err := bookingtokenv2.NewBookingtokenv2(common.HexToAddress("0xe55E387F5474a012D1b048155E25ea78C7DBfBBC"), client)
	if err != nil {
		sugar.Fatalf("Failed to create BookingToken contract binding: %v", err)
	}

	// token uri
	tokenURI := "data:application/json;base64,eyJuYW1lIjoiYm90IGNtYWNjb3VudCBwa2cgYm9va2luZyB0b2tlbiB0ZXN0In0K"

	// expiration timestamp
	expiration := big.NewInt(time.Now().Add(time.Hour).Unix())

	nativeTokenAddress := common.HexToAddress("0x0000000000000000000000000000000000000000")
	// https://columbus.caminoscan.com/token/0x5b1c852dad36854B0dFFF61d2C13F108D8E01975
	eurshToken := common.HexToAddress("0x5b1c852dad36854B0dFFF61d2C13F108D8E01975")
	// testToken := common.HexToAddress("0x53A0b6A344C8068B211d47f177F0245F5A99eb2d")

	var paymentToken common.Address = nativeTokenAddress
	var priceBigInt *big.Int
	var price *typesv2.Price

	// Example prices for ISO Currency
	priceEUR := &typesv2.Price{
		Value:    "10000",
		Decimals: 2,
		Currency: &typesv2.Currency{
			Currency: &typesv2.Currency_IsoCurrency{
				IsoCurrency: typesv2.IsoCurrency_ISO_CURRENCY_EUR,
			},
		},
	}

	// Example prices for Token Currency
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

	// Example prices for Native Token
	priceCAM := &typesv2.Price{
		Value:    "1",
		Decimals: 9,
		Currency: &typesv2.Currency{
			Currency: &typesv2.Currency_NativeToken{
				NativeToken: &emptypb.Empty{},
			},
		},
	}

	sugar.Infof("%v %v %v %v", priceEUR, priceEURSH, priceCAM)
	sugar.Infof("%v", price)

	paymentToken = nativeTokenAddress
	priceBigInt = big.NewInt(0)
	price = priceEURSH
	// price = priceCAM

	switch currency := price.Currency.Currency.(type) {
	case *typesv2.Currency_NativeToken:
		priceBigInt, err = bs.ConvertPriceToBigInt(price.Value, price.Decimals, int32(18)) // CAM uses 18 decimals
		if err != nil {
			sugar.Errorf("Failed to convert price to big.Int: %v", err)
			return
		}
		sugar.Infof("Converted the price big.Int: %v", priceBigInt)
		paymentToken = nativeTokenAddress
	case *typesv2.Currency_TokenCurrency:
		if !common.IsHexAddress(currency.TokenCurrency.ContractAddress) {
			sugar.Errorf("invalid contract address: %v", currency.TokenCurrency.ContractAddress)
		}
		contractAddress := common.HexToAddress(currency.TokenCurrency.ContractAddress)

		// Fetch decimals from the ERC20 contract

		tokenDecimals, err := erc20.Decimals(context.Background(), contractAddress)
		if err != nil {
			sugar.Errorf("failed to fetch token decimals: %w", err)
		}

		priceBigInt, err = bs.ConvertPriceToBigInt(price.Value, price.Decimals, int32(tokenDecimals))
		if err != nil {
			sugar.Errorf("Failed to convert price to big.Int: %v", err)
		}
		paymentToken = contractAddress
	case *typesv2.Currency_IsoCurrency:
		priceBigInt = big.NewInt(0)
		paymentToken = nativeTokenAddress
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

	receipt, err := bs.MintBookingToken(
		context.Background(),
		cmAccountAddr, // reservedFor address
		tokenURI,
		expiration,
		priceBigInt,
		paymentToken,
		true,
	)
	if err != nil {
		sugar.Fatalf("Failed to mint booking token: %v", err)
	}

	tokenID := big.NewInt(0)

	for _, mLog := range receipt.Logs {
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
	buyReceipt, err := bs.BuyBookingToken(
		context.Background(),
		tokenID,
	)
	if err != nil {
		sugar.Fatalf("Failed to buy booking token: %v", err)
	}

	// Parse the logs
	for _, bLog := range buyReceipt.Logs {
		event, err := bt.ParseTokenBought(*bLog)
		if err == nil {
			sugar.Infof("[TokenBought] TokenID: %s, Buyer: %s", event.TokenId, event.Buyer)
		}
	}
}

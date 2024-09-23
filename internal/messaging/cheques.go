package messaging

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	"github.com/chain4travel/camino-messenger-bot/pkg/db"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var _ ChequeHandler = (*evmChequeHandler)(nil)

type evmChequeHandler struct {
	logger            *zap.SugaredLogger
	ethClient         *ethclient.Client
	evmConfig         *config.EvmConfig
	cmAccountAddress  common.Address
	cmAccountInstance *cmaccount.Cmaccount
	chainId           *big.Int
	privateKey        *ecdsa.PrivateKey

	cfg *config.EvmConfig
}

type ChequeHandler interface {
	issueCheque(ctx context.Context, fromCmAccount common.Address, toCmAccount common.Address, toBot common.Address, amount *big.Int) ([]byte, error)
	getLastCashIn(ctx context.Context, fromCmAccount common.Address, toBot common.Address) (*LastCashIn, error)
	getServiceFeeByName(serviceName string, CmAccountAddress common.Address) (*big.Int, error)
	isBotAllowed() (bool, error)
}

type LastCashIn struct {
	Counter   *big.Int
	Amount    *big.Int
	CreatedAt *big.Int
	ExpiresAt *big.Int
}

func NewChequeHandler(logger *zap.SugaredLogger, ethClient *ethclient.Client, evmConfig *config.EvmConfig, chainId *big.Int) (ChequeHandler, error) {
	pk := new(secp256k1.PrivateKey)

	if err := pk.UnmarshalText([]byte("\"" + evmConfig.PrivateKey + "\"")); err != nil {
		logger.Fatalf("Failed to parse private key: %v", err)
	}
	ecdsaPk := pk.ToECDSA()

	// Get Ethereum Address from private key
	botCChainAddress := crypto.PubkeyToAddress(ecdsaPk.PublicKey)
	logger.Infof("C-Chain address: %s", botCChainAddress)

	// Get contract address as common.Address
	cmAccountAddress := common.HexToAddress(evmConfig.CMAccountAddress)

	// Instantiate the contract binding
	cmAccountInstance, err := cmaccount.NewCmaccount(cmAccountAddress, ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate contract binding: %v", err)
	}

	return &evmChequeHandler{
		ethClient:         ethClient,
		cmAccountAddress:  cmAccountAddress,
		cmAccountInstance: cmAccountInstance,
		chainId:           chainId,
		privateKey:        ecdsaPk,
		logger:            logger,
		evmConfig:         evmConfig,
	}, nil
}

func (cm *evmChequeHandler) issueCheque(ctx context.Context, fromCMAccount common.Address, toCMAccount common.Address, toBot common.Address, amount *big.Int) ([]byte, error) {

	// get counter from db
	fromBot := getAddressFromECDSAPrivateKey(cm.privateKey)

	db.GetLatestCounter(ctx, fromBot, toBot.Hex())

	signer := cheques.NewSigner(cm.privateKey, cm.chainId)
	cheque := &cheques.Cheque{
		FromCMAccount: fromCMAccount,
		ToCMAccount:   toCMAccount,
		ToBot:         toBot,
		Counter:       big.NewInt(0),
		Amount:        amount,
		CreatedAt:     big.NewInt(0),
		ExpiresAt:     big.NewInt(0),
	}

	cm.logger.Infof("Cheque issued with tx hash: %s", tx.Hash().Hex())
	return tx.Hash().Bytes(), nil
}

func (cm *evmChequeHandler) getLastCashIn(ctx context.Context, fromBot common.Address, toBot common.Address) (*LastCashIn, error) {
	// Use contract binding to interact with the contract
	callOpts := &bind.CallOpts{
		Context: ctx,
	}

	cashIn, err := cm.cmAccount.GetLastCashIn(callOpts, fromBot, toBot)
	if err != nil {
		return nil, fmt.Errorf("failed to get last cash in: %v", err)
	}

	// Return as LastCashIn struct
	return &LastCashIn{
		Counter:   cashIn.Counter,
		Amount:    cashIn.Amount,
		CreatedAt: cashIn.CreatedAt,
		ExpiresAt: cashIn.ExpiresAt,
	}, nil
}

func (cm *evmChequeHandler) getServiceFeeByName(serviceName string, CMAccountAddress common.Address) (*big.Int, error) {
	// Hash the service name
	serviceHash := serviceNameToHash(serviceName)

	callOpts := &bind.CallOpts{
		Context: context.Background(),
	}

	// Use contract binding to get the service fee
	serviceFee, err := cm.CMAccount.GetServiceFee(callOpts, serviceHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get service fee: %v", err)
	}

	return serviceFee, nil
}

func (cm *evmChequeHandler) isBotAllowed() (bool, error) {
	callOpts := &bind.CallOpts{
		Context: context.Background(),
	}

	botAddress, err := getAddressFromECDSAPrivateKey(cm.privateKey)
	if err != nil {
		return false, err
	}

	// Use contract binding to check if the bot is allowed
	isAllowed, err := cm.CMAccount.IsBotAllowed(callOpts, botAddress)
	if err != nil {
		return false, fmt.Errorf("failed to check if bot is allowed: %v", err)
	}

	return isAllowed, nil
}

func (cm *evmChequeHandler) buildTransactOpts(ctx context.Context) (*bind.TransactOpts, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(cm.privateKey, cm.chainId)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyed transactor: %v", err)
	}
	auth.Context = ctx
	auth.GasLimit = uint64(300000) // You can adjust the gas limit accordingly

	return auth, nil
}

func (cm *evmChequeHandler) isBotAllowed() (bool, error) {
	callOpts := &bind.CallOpts{
		Context: context.Background(),
	}

	botAddress, err := getAddressFromECDSAPrivateKey(cm.privateKey)
	if err != nil {
		return false, err
	}

	// Use contract binding to check if the bot is allowed
	isAllowed, err := cm.cmAccountInstance.IsBotAllowed(callOpts, botAddress)
	if err != nil {
		return false, fmt.Errorf("Bot does not have necessary permission: %v", err)
	}

	return isAllowed, nil
}

func getAddressFromECDSAPrivateKey(privateKey *ecdsa.PrivateKey) (common.Address, error) {
	// Derive the public key
	publicKey := privateKey.Public()

	// Convert the public key to the correct type (elliptic.Curve type)
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, fmt.Errorf("error casting public key to ECDSA")
	}

	// Generate the Ethereum address from the public key
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return address, nil
}

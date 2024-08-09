package messaging

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/chain4travel/camino-messenger-bot/config"
	"go.uber.org/zap"
)

type Cheque struct {
	FromCMAccount common.Address
	ToCMAccount   common.Address
	ToBot         common.Address
	Counter       *big.Int
	Amount        *big.Int
	Timestamp     time.Time
}

type evmChequeHandler struct {
	client                  *ethclient.Client
	messengerCashierABI     *abi.ABI
	messengerCashierAddress common.Address
	cmAccountAddress        common.Address
	privateKey              *ecdsa.PrivateKey
	ctx                     context.Context
}

type ChequeHandler interface {
	issueCheque(fromCMAccount common.Address, toCMAccount, toBot common.Address, amount *big.Int) (*types.Transaction, error)
	verifyCheque()
	calculateDomainSeparator(cfg *config.EvmConfig) (common.Hash, error)
	calculateChequeHash(domainSeparator common.Hash, cheque Cheque) common.Hash
}

func NewChequeHandler(client *ethclient.Client, logger *zap.SugaredLogger, cfg *config.EvmConfig, privateKey *ecdsa.PrivateKey) (*evmChequeHandler, error) {
	abi, err := loadABI(cfg.MessengerCashierABIFile)
	if err != nil {
		return nil, err
	}

	pk := new(secp256k1.PrivateKey)
	// UnmarshalText expects the private key in quotes
	if err := pk.UnmarshalText([]byte("\"" + cfg.PrivateKey + "\"")); err != nil {
		logger.Fatalf("Failed to parse private key: %v", err)
	}
	ecdsaPk := pk.ToECDSA()

	// Get Ethereum Address from private key
	botCChainAddress := crypto.PubkeyToAddress(ecdsaPk.PublicKey)
	logger.Infof("C-Chain address: %s", botCChainAddress)

	return &evmChequeHandler{
		client:                  client,
		messengerCashierAddress: botCChainAddress,
		cmAccountAddress:        botCChainAddress,
		privateKey:              ecdsaPk,
		ctx:                     context.Context,
	}, nil
}

func (cm *evmChequeHandler) issueCheque(fromCMAccount, toCMAccount, toBot common.Address, amount *big.Int) ([]byte, error) {
	// Prepare the cheque data
	cheque := Cheque{
		FromCMAccount: fromCMAccount,
		ToCMAccount:   toCMAccount,
		ToBot:         toBot,
		Counter:       big.NewInt(1), // Assuming this is the first cheque
		Amount:        amount,
		Timestamp:     time.Now(),
	}

	// Calculate the domain separator
	domainSeparator, err := cm.calculateDomainSeparator(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate domain separator: %v", err)
	}

	// Calculate the cheque hash
	finalHash := cm.calculateChequeHash(domainSeparator, cheque)

	// Sign the cheque hash
	signature, err := crypto.Sign(finalHash.Bytes(), cm.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign hash: %v", err)
	}

	return signature, nil
}

func (cm *evmChequeHandler) calculateDomainSeparator(cfg *config.EvmConfig) (common.Hash, error) {
	chainID, err := cm.client.NetworkID(cm.ctx)
	if err != nil {
		return common.Hash{}, err
	}

	domainType := "EIP712Domain(string name,string version,uint256 chainId)"
	domainTypeHash := crypto.Keccak256Hash([]byte(domainType))
	nameHash := crypto.Keccak256Hash([]byte(cfg.DomainName))
	versionHash := crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", cfg.DomainVersion)))
	chainIDHash := crypto.Keccak256Hash(chainID.Bytes())
	verifyingContract := common.HexToAddress(config.MessengerCashierAddressKey).Bytes()

	domainSeparator := crypto.Keccak256Hash(
		domainTypeHash.Bytes(),
		nameHash.Bytes(),
		versionHash.Bytes(),
		chainIDHash.Bytes(),
		verifyingContract,
	)

	return domainSeparator, nil
}

func (cm *evmChequeHandler) calculateChequeHash(domainSeparator common.Hash, cheque Cheque) common.Hash {
	// Define the cheque type
	chequeType := "Cheque(address fromCMAccount,address toCMAccount,address toBot,uint256 counter,uint256 amount,uint256 timestamp)"
	chequeTypeHash := crypto.Keccak256Hash([]byte(chequeType))

	timestampBytes := big.NewInt(cheque.Timestamp.Unix()).Bytes()

	// Hash the cheque data
	chequeHash := crypto.Keccak256Hash(
		chequeTypeHash.Bytes(),
		cheque.FromCMAccount.Bytes(),
		cheque.ToCMAccount.Bytes(),
		cheque.ToBot.Bytes(),
		crypto.Keccak256Hash(cheque.Counter.Bytes()).Bytes(),
		crypto.Keccak256Hash(cheque.Amount.Bytes()).Bytes(),
		crypto.Keccak256Hash(timestampBytes).Bytes(),
	)

	// Combine domain and cheque hashes
	finalHash := crypto.Keccak256(
		[]byte("\x19\x01"),
		domainSeparator.Bytes(),
		chequeHash.Bytes(),
	)

	return common.BytesToHash(finalHash)
}

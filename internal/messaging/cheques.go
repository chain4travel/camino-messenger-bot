package messaging

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	"fmt"

	"github.com/ethereum/go-ethereum"
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
	fromCMAccount common.Address
	toCMAccount   common.Address
	toBot         common.Address
	counter       *big.Int
	amount        *big.Int
	createdAt     time.Time
	expiresAt     time.Time
}

// caminogo feature/tchain
// asb -> verification

type evmChequeHandler struct {
	ethClient        *ethclient.Client
	CMAccountABI     *abi.ABI
	CMAccountAddress common.Address
	privateKey       *ecdsa.PrivateKey
	logger           *zap.SugaredLogger
	domainVersion    uint64
	domainName       string
}

type ChequeHandler interface {
	issueCheque(fromCMAccount string, toCMAccount string, toBot string, amount *big.Int) (*types.Transaction, error)
	verifyCheque()
	calculateDomainSeparator(cfg *config.EvmConfig) (common.Hash, error)
	calculateChequeHash(domainSeparator common.Hash, cheque Cheque) common.Hash
	getLastCashIn(fromBot common.Address, toBot common.Address)
}

type LastCashIn struct {
	counter   *big.Int
	amount    *big.Int
	createdAt *big.Int
	expiresAt *big.Int
}

// vms/touristicvm/txs/cashout_cheque_tx.go
// each cheque received need to verify with local data, amount >= previous_am
// verify signature / cache of cmAccount
// on each received cheque call smart contract
// verify locally the count/amount because its more uptodate than smart contract
//  skip check if data is newer than smart contract >= sc

// 1. verify  signature - with SC
// 2. verify amount and count locally

func NewChequeHandler(ethClient *ethclient.Client, logger *zap.SugaredLogger, cfg *config.EvmConfig) (*evmChequeHandler, error) {
	abi, err := loadABI(cfg.CMAccountABIFile)
	if err != nil {
		return nil, err
	}

	pk := new(secp256k1.PrivateKey)

	if err := pk.UnmarshalText([]byte("\"" + cfg.PrivateKey + "\"")); err != nil {
		logger.Fatalf("Failed to parse private key: %v", err)
	}
	ecdsaPk := pk.ToECDSA()

	// Get Ethereum Address from private key
	botCChainAddress := crypto.PubkeyToAddress(ecdsaPk.PublicKey)
	logger.Infof("C-Chain address: %s", botCChainAddress)

	// Get contract address as common.Address
	CMAccountAddress := common.HexToAddress(cfg.CMAccountAddress)

	return &evmChequeHandler{
		ethClient:        ethClient,
		CMAccountABI:     abi,
		CMAccountAddress: CMAccountAddress,
		privateKey:       ecdsaPk,
		logger:           logger,
		domainVersion:    cfg.DomainVersion,
		domainName:       cfg.DomainName,
	}, nil
}

func (cm *evmChequeHandler) issueCheque(ctx context.Context, fromCMAccount, toCMAccount, toBot common.Address, amount *big.Int) ([]byte, error) {
	// Prepare the cheque data
	// amount
	// counter should be persistent from db
	// on check creation increment value +1
	// amount -> total amount that was paid to this recipient from this sender overall (get previous amount and increment by new value)
	// counter total amount of messages (sender/recipient)

	// get last cash in from smart contract
	lastCashIn, err := cm.getLastCashIn(ctx, fromCMAccount, toBot)
	if err != nil {
		return nil, fmt.Errorf("failed to get last cash in: %v", err)
	}
	// if data is newer than smart contract, skip check (check if local data is >= smart contract data)

	_amount := lastCashIn.amount.Add(lastCashIn.amount, amount)
	_counter := lastCashIn.counter.Add(lastCashIn.counter, big.NewInt(1))

	// check if there is newer info in database

	cheque := Cheque{
		fromCMAccount: fromCMAccount,
		toCMAccount:   toCMAccount,
		toBot:         toBot,
		counter:       _counter, // Assuming this is the first cheque
		amount:        _amount,
		createdAt:     time.Now(),
		expiresAt:     time.Now().Add(24 * time.Hour),
	}

	// Sign the cheque hash
	signature, err := signCheque(ctx, cm, cheque)
	if err != nil {
		return nil, fmt.Errorf("failed to sign hash: %v", err)
	}
	// increment counter and save new counter to database

	_counter = _counter.Add(_counter, big.NewInt(1))
	_amount = _amount.Add(_amount, amount)

	return signature, nil
}

func signCheque(ctx context.Context, handler *evmChequeHandler, cheque Cheque) ([]byte, error) {
	// Get the chain ID
	chainID, err := handler.ethClient.NetworkID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %v", err)
	}

	// Calculate domain separator
	domainSeparator, err := calculateDomainSeparator(handler.domainName, handler.domainVersion, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate domain separator: %v", err)
	}

	// Calculate cheque hash
	chequeHash, err := calculateChequeHash(cheque)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cheque hash: %v", err)
	}

	// Calculate typed data hash
	finalHash := calculateTypedDataHash(domainSeparator, chequeHash)

	// Sign the hash
	signature, err := crypto.Sign(finalHash.Bytes(), handler.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign the hash: %v", err)
	}

	return signature, nil
}

func (cm *evmChequeHandler) getLastCashIn(ctx context.Context, fromBot, toBot common.Address) (*LastCashIn, error) {
	// Pack the method call with parameters
	input, err := cm.messengerCashierABI.Pack("getLastCashIn", fromBot, toBot)
	if err != nil {
		return nil, fmt.Errorf("failed to pack arguments: %v", err)
	}

	// Call the contract using ethclient.Client
	output, err := cm.ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &cm.messengerCashierAddress,
		Data: input,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %v", err)
	}

	// Unpack the result into a LastCashIn struct
	var cashIn LastCashIn
	err = cm.messengerCashierABI.UnpackIntoInterface(&cashIn, "getLastCashIn", output)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %v", err)
	}

	return &cashIn, nil
}

func (cm *evmChequeHandler) verifyCheque(ctx context.Context, cheque Cheque, signature []byte) (*ChequeVerifiedEvent, error) {
	// Get the nonce for the transaction
	nonce, err := cm.ethClient.PendingNonceAt(ctx, cm.CMAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %v", err)
	}

	// Get the suggested gas price
	gasPrice, err := cm.ethClient.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %v", err)
	}

	// Set a safe gas limit
	gasLimit := uint64(200000)

	// Pack the method call with parameters
	packed, err := cm.CMAccountABI.Pack("verifyCheque", cheque, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to pack arguments: %v", err)
	}

	// Create a new transaction
	tx := types.NewTransaction(
		nonce,
		cm.CMAccountAddress,
		big.NewInt(0), // No value sent with the transaction
		gasLimit,
		gasPrice,
		packed,
	)

	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, cm.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send the transaction
	err = cm.ethClient.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %v", err)
	}

	// Wait for the transaction to be mined and fetch the event
	txHash := signedTx.Hash()
	event, err := cm.getChequeVerifiedEvent(txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get ChequeVerified event: %v", err)
	}

	// If no event was found, abort the operation and return an error
	if event == nil {
		return nil, fmt.Errorf("cheque verification failed: no ChequeVerified event emitted")
	}

	return event, nil
}

func calculateDomainSeparator(domainName string, domainVersion uint64, chainID *big.Int) (common.Hash, error) {
	domainType := "EIP712Domain(string name,string version,uint256 chainId)"
	domainTypeHash := crypto.Keccak256Hash([]byte(domainType))
	nameHash := crypto.Keccak256Hash([]byte(domainName))
	versionHash := crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", domainVersion)))

	// Pack the parameters and hash them together
	encodedData := crypto.Keccak256(
		domainTypeHash.Bytes(),
		nameHash.Bytes(),
		versionHash.Bytes(),
		crypto.Keccak256Hash(chainID.Bytes()).Bytes(),
	)

	return common.BytesToHash(encodedData), nil
}

func calculateChequeHash(cheque Cheque) (common.Hash, error) {
	chequeType := "MessengerCheque(address fromCMAccount,address toCMAccount,address toBot,uint256 counter,uint256 amount,uint256 createdAt,uint256 expiresAt)"
	chequeTypeHash := crypto.Keccak256Hash([]byte(chequeType))

	createdAtBytes := big.NewInt(cheque.createdAt.Unix()).Bytes()
	expiresAtBytes := big.NewInt(cheque.expiresAt.Unix()).Bytes()

	// Pack the cheque data
	encodedCheque := crypto.Keccak256(
		chequeTypeHash.Bytes(),
		cheque.fromCMAccount.Bytes(),
		cheque.toCMAccount.Bytes(),
		cheque.toBot.Bytes(),
		crypto.Keccak256Hash(cheque.counter.Bytes()).Bytes(),
		crypto.Keccak256Hash(cheque.amount.Bytes()).Bytes(),
		crypto.Keccak256Hash(createdAtBytes).Bytes(),
		crypto.Keccak256Hash(expiresAtBytes).Bytes(),
	)

	return common.BytesToHash(encodedCheque), nil
}

func calculateTypedDataHash(domainSeparator common.Hash, chequeHash common.Hash) common.Hash {
	// "\x19\x01" prefix is added according to EIP-712
	finalHash := crypto.Keccak256(
		[]byte("\x19\x01"),
		domainSeparator.Bytes(),
		chequeHash.Bytes(),
	)
	return common.BytesToHash(finalHash)
}

type ChequeVerifiedEvent struct {
	FromCMAccount common.Address
	ToCMAccount   common.Address
	FromBot       common.Address
	ToBot         common.Address
	Counter       *big.Int
	Amount        *big.Int
	Payment       *big.Int
}

func (cm *evmChequeHandler) getChequeVerifiedEvent(txHash common.Hash) (*ChequeVerifiedEvent, error) {
	receipt, err := cm.ethClient.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %v", err)
	}

	for _, log := range receipt.Logs {
		// Check if the log is of the ChequeVerified event
		event := new(ChequeVerifiedEvent)
		err := cm.CMAccountABI.UnpackIntoInterface(event, "ChequeVerified", log.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack log: %v", err)
		}

		// Parse indexed fields
		event.FromCMAccount = common.HexToAddress(log.Topics[1].Hex())
		event.ToCMAccount = common.HexToAddress(log.Topics[2].Hex())
		event.FromBot = common.HexToAddress(log.Topics[3].Hex())
		event.ToBot = common.HexToAddress(log.Topics[4].Hex())

		return event, nil
	}

	return nil, fmt.Errorf("no ChequeVerified event found in transaction logs")
}

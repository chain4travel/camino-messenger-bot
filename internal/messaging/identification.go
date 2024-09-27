package messaging

import (
	"log"
	"math/big"
	"strings"

	config "github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
	"maunium.net/go/mautrix/id"
)

const cmAccountsCacheSize = 100

var _ IdentificationHandler = (*evmIdentificationHandler)(nil)

var roleHash = crypto.Keccak256Hash([]byte("CHEQUE_OPERATOR_ROLE"))

type evmIdentificationHandler struct {
	ethClient          *ethclient.Client
	cfg                *config.EvmConfig
	matrixHost         string
	myCMAccountAddress common.Address
	cmAccounts         *lru.Cache[common.Address, *cmaccount.Cmaccount]
}

type IdentificationHandler interface {
	getMyCMAccountAddress() common.Address
	getFirstBotUserIDFromCMAccountAddress(common.Address) (id.UserID, error)
}

func NewIdentificationHandler(ethClient *ethclient.Client, _ *zap.SugaredLogger, cfg *config.EvmConfig, mCfg *config.MatrixConfig) (IdentificationHandler, error) {
	cmAccountsCache, err := lru.New[common.Address, *cmaccount.Cmaccount](cmAccountsCacheSize)
	if err != nil {
		return nil, err
	}

	return &evmIdentificationHandler{
		ethClient:          ethClient,
		cfg:                cfg,
		matrixHost:         mCfg.Host,
		myCMAccountAddress: common.HexToAddress(cfg.CMAccountAddress),
		cmAccounts:         cmAccountsCache,
	}, nil
}

func (ih *evmIdentificationHandler) getMyCMAccountAddress() common.Address {
	return ih.myCMAccountAddress
}

func (ih *evmIdentificationHandler) getFirstBotUserIDFromCMAccountAddress(cmAccountAddress common.Address) (id.UserID, error) {
	botAddress, err := ih.getFirstBotFromCMAccountAddress(cmAccountAddress)
	if err != nil {
		return "", err
	}

	return userIDFromAddress(botAddress, ih.matrixHost), nil
}

func (ih *evmIdentificationHandler) getFirstBotFromCMAccountAddress(cmAccountAddress common.Address) (common.Address, error) {
	bots, err := ih.getAllBotAddressesFromCMAccountAddress(cmAccountAddress)
	if err != nil {
		return common.Address{}, err
	}
	return bots[0], nil
}

func (ih *evmIdentificationHandler) getAllBotAddressesFromCMAccountAddress(cmAccountAddress common.Address) ([]common.Address, error) {
	cmAccount, err := cmaccount.NewCmaccount(cmAccountAddress, ih.ethClient)
	if err != nil {
		log.Printf("Failed to get cm Account: %v", err)
		return nil, err
	}

	countBig, err := cmAccount.GetRoleMemberCount(&bind.CallOpts{}, roleHash)
	if err != nil {
		log.Printf("Failed to call contract function: %v", err)
		return nil, err
	}

	botsAddresses := []common.Address{}
	count := countBig.Int64()
	for i := int64(0); i < count; i++ {
		address, err := cmAccount.GetRoleMember(&bind.CallOpts{}, roleHash, big.NewInt(i))
		if err != nil {
			log.Printf("Failed to call contract function: %v", err)
		}
		botsAddresses = append(botsAddresses, address)
	}

	return botsAddresses, nil
}

func userIDFromAddress(address common.Address, host string) id.UserID {
	return id.NewUserID(strings.ToLower(address.Hex()), host)
}

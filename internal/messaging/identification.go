package messaging

import (
	"log"
	"math/big"

	config "github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	"maunium.net/go/mautrix/id"
)

var _ IdentificationHandler = (*evmIdentificationHandler)(nil)

var roleHash = crypto.Keccak256Hash([]byte("CHEQUE_OPERATOR_ROLE"))

type evmIdentificationHandler struct {
	ethClient       *ethclient.Client
	cfg             *config.EvmConfig
	matrixHost      string
	cmAccountBotMap map[common.Address]id.UserID
}

type IdentificationHandler interface {
	getAllBotAddressesFromCMAccountAddress(cmAccountAddress common.Address) ([]string, error)
	getFirstBotFromCMAccountAddress(cmAccountAddress common.Address) (string, error)
	isMyCMAccount(cmAccountAddress common.Address) bool
	getMyCMAccountAddress() common.Address
	getMatrixHost() string
	isBotInCMAccount(common.Address, common.Address) (bool, error)
	getCmAccount(id.UserID) (common.Address, bool)
	addToMap(common.Address, id.UserID)
	getBotFromMap(common.Address) (bool, id.UserID)
}

func NewIdentificationHandler(ethClient *ethclient.Client, _ *zap.SugaredLogger, cfg *config.EvmConfig, mCfg *config.MatrixConfig) (IdentificationHandler, error) {
	return &evmIdentificationHandler{
		ethClient:       ethClient,
		cfg:             cfg,
		matrixHost:      mCfg.Host,
		cmAccountBotMap: make(map[common.Address]id.UserID),
	}, nil
}

func (cm *evmIdentificationHandler) getMatrixHost() string {
	return cm.matrixHost
}

func (cm *evmIdentificationHandler) getMyCMAccountAddress() common.Address {
	return common.HexToAddress(cm.cfg.CMAccountAddress)
}

// Add configuration to the bot to configure to which CM-Account it belongs (to prevent that they're part of multiple CM-Accounts)
func (cm *evmIdentificationHandler) isMyCMAccount(cmAccountAddress common.Address) bool {
	return cmAccountAddress == common.HexToAddress(cm.cfg.CMAccountAddress)
}

func (cm *evmIdentificationHandler) getAllBotAddressesFromCMAccountAddress(cmAccountAddress common.Address) ([]string, error) {
	cmAccount, err := cmaccount.NewCmaccount(cmAccountAddress, cm.ethClient)
	if err != nil {
		log.Printf("Failed to get cm Account: %v", err)
		return nil, err
	}

	countBig, err := cmAccount.GetRoleMemberCount(&bind.CallOpts{}, roleHash)
	if err != nil {
		log.Printf("Failed to call contract function: %v", err)
		return nil, err
	}

	bots := []string{}
	count := countBig.Int64()
	for i := int64(0); i < count; i++ {
		address, err := cmAccount.GetRoleMember(&bind.CallOpts{}, roleHash, big.NewInt(i))
		if err != nil {
			log.Printf("Failed to call contract function: %v", err)
		}
		bots = append(bots, address.Hex())
	}

	return bots, nil
}

func (cm *evmIdentificationHandler) getFirstBotFromCMAccountAddress(cmAccountAddress common.Address) (string, error) {
	bots, err := cm.getAllBotAddressesFromCMAccountAddress(cmAccountAddress)
	if err != nil {
		return "", err
	}
	return bots[0], nil
}

func (cm *evmIdentificationHandler) isBotInCMAccount(botAddress common.Address, cmAccountAddress common.Address) (bool, error) {
	bots, err := cm.getAllBotAddressesFromCMAccountAddress(cmAccountAddress)
	if err != nil {
		return false, err
	}
	for _, b := range bots {
		if common.HexToAddress(b) == botAddress {
			return true, nil
		}
	}
	if cm.cmAccountBotMap[cmAccountAddress] == id.NewUserID(botAddress.Hex(), cm.getMatrixHost()) {
		delete(cm.cmAccountBotMap, cmAccountAddress)
	}
	return false, nil
}

func (cm *evmIdentificationHandler) getCmAccount(bot id.UserID) (common.Address, bool) {
	for key, b := range cm.cmAccountBotMap {
		if b == bot {
			return key, true
		}
	}
	return common.Address{}, false
}

func (cm *evmIdentificationHandler) addToMap(cmaccount common.Address, botID id.UserID) {
	cm.cmAccountBotMap[cmaccount] = botID
}

func (cm *evmIdentificationHandler) getBotFromMap(cmaccount common.Address) (bool, id.UserID) {
	bot := cm.cmAccountBotMap[cmaccount]
	if cm.cmAccountBotMap[cmaccount] == "" {
		return false, ""
	}
	return true, bot
}

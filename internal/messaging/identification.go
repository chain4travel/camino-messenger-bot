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
	"go.uber.org/zap"
)

// var CMAccountBotMap = map[common.Address]Bot{}

var CMAccountBotMap = map[common.Address]string{}

var _ IdentificationHandler = (*evmIdentificationHandler)(nil)

type evmIdentificationHandler struct {
	ethClient  *ethclient.Client
	cfg        *config.EvmConfig
	matrixHost string
}

type IdentificationHandler interface {
	getAllBotAddressesFromCMAccountAddress(cmAccountAddress common.Address) ([]string, error)
	getSingleBotFromCMAccountAddress(cmAccountAddress common.Address) (string, error)
	isMyCMAccount(cmAccountAddress common.Address) bool
	getMyCMAccountAddress() string
	getMatrixHost() string
	isBotInCMAccount(string, common.Address) (bool, error)
	findCmAccount(string) (common.Address, bool)
	addToMap(common.Address, string)
}

func NewIdentificationHandler(ethClient *ethclient.Client, _ *zap.SugaredLogger, cfg *config.EvmConfig, mCfg *config.MatrixConfig) (IdentificationHandler, error) {

	return &evmIdentificationHandler{
		ethClient:  ethClient,
		cfg:        cfg,
		matrixHost: mCfg.Host,
	}, nil
}

func (cm *evmIdentificationHandler) getMatrixHost() string {
	return cm.matrixHost
}

func (cm *evmIdentificationHandler) getMyCMAccountAddress() string {
	return cm.cfg.CMAccountAddress
}

// Add configuration to the bot to configure to which CM-Account it belongs (to prevent that they're part of multiple CM-Accounts)
func (cm *evmIdentificationHandler) isMyCMAccount(cmAccountAddress common.Address) bool {
	return cmAccountAddress == common.HexToAddress(cm.cfg.CMAccountAddress)
}

func (cm *evmIdentificationHandler) getAllBotAddressesFromCMAccountAddress(cmAccountAddress common.Address) ([]string, error) {
	roleHash := crypto.Keccak256Hash([]byte("CHEQUE_OPERATOR_ROLE"))

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
	// Check if count is greater than 0
	count := int(countBig.Int64())
	if count > 0 {
		for i := 0; i < count; i++ {
			address, err := cmAccount.GetRoleMember(&bind.CallOpts{}, roleHash, big.NewInt(int64(i)))
			if err != nil {
				log.Printf("Failed to call contract function: %v", err)
			}
			bots = append(bots, address.Hex())
		}
	} else {
		return bots, nil
	}

	return bots, nil
}

func (cm *evmIdentificationHandler) getSingleBotFromCMAccountAddress(cmAccountAddress common.Address) (string, error) {
	bots, err := cm.getAllBotAddressesFromCMAccountAddress(cmAccountAddress)
	if err != nil {
		return "", err
	}
	return bots[0], nil
}

func (cm *evmIdentificationHandler) isBotInCMAccount(botAddress string, cmAccountAddress common.Address) (bool, error) {
	bots, err := cm.getAllBotAddressesFromCMAccountAddress(cmAccountAddress)
	if err != nil {
		return false, err
	}
	for _, b := range bots {
		if strings.EqualFold(strings.ToUpper(b), strings.ToUpper(botAddress)) {
			return true, nil
		}
	}
	if CMAccountBotMap[cmAccountAddress] == "@"+botAddress+cm.getMatrixHost() {
		delete(CMAccountBotMap, cmAccountAddress)
	}
	return false, nil
}

func (cm *evmIdentificationHandler) findCmAccount(bot string) (common.Address, bool) {
	for key, b := range CMAccountBotMap {
		if b == bot {
			return key, true
		}
	}
	return common.Address{}, false
}

func (cm *evmIdentificationHandler) addToMap(cmaccount common.Address, botID string) {
	CMAccountBotMap[cmaccount] = botID
}

package messaging

import (
	"context"
	"log"
	"math/big"

	config "github.com/chain4travel/camino-messenger-bot/config"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

// var CMAccountBotMap = map[common.Address]Bot{}

var CMAccountBotMap = map[common.Address]string{}

var _ IdentificationHandler = (*evmIdentificationHandler)(nil)

type evmIdentificationHandler struct {
	ethClient    *ethclient.Client
	CMAccountABI *abi.ABI
	cfg          *config.EvmConfig
}

type IdentificationHandler interface {
	getAllBotAddressesFromCMAccountAddress(cmAccountAddress common.Address) ([]string, error)
	getSingleBotFromCMAccountAddress(cmAccountAddress common.Address) (string, error)
	isMyCMAccount(cmAccountAddress common.Address) bool
}

// Add configuration to the bot to configure to which CM-Account it belongs (to prevent that they're part of multiple CM-Accounts)
func (cm *evmIdentificationHandler) isMyCMAccount(cmAccountAddress common.Address) bool {
	return cmAccountAddress == common.HexToAddress(cm.cfg.CMAccountAddress)
}

func NewIdentificationHandler(ethClient *ethclient.Client, _ *zap.SugaredLogger, cfg *config.EvmConfig) (IdentificationHandler, error) {
	abi, err := loadABI(cfg.CMAccountABIFile)
	if err != nil {
		return nil, err
	}

	return &evmIdentificationHandler{
		ethClient:    ethClient,
		CMAccountABI: abi,
		cfg:          cfg,
	}, nil
}

func (cm *evmIdentificationHandler) getAllBotAddressesFromCMAccountAddress(cmAccountAddress common.Address) ([]string, error) {
	roleHash := crypto.Keccak256Hash([]byte("CHEQUE_OPERATOR_ROLE"))
	data, err := cm.CMAccountABI.Pack("getRoleMemberCount", roleHash)
	if err != nil {
		log.Fatalf("Failed to pack contract function call: %v", err)
	}

	callMsg := ethereum.CallMsg{
		To:   &cmAccountAddress,
		Data: data,
	}

	if err != nil {
		return nil, err
	}

	result, err := cm.ethClient.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		log.Fatalf("Failed to call contract function: %v", err)
		return nil, err
	}

	var countBig *big.Int
	err = cm.CMAccountABI.UnpackIntoInterface(&countBig, "getRoleMemberCount", result)
	if err != nil {
		log.Fatalf("Failed to unpack result: %v", err)
		return nil, err
	}

	bots := []string{}
	// Check if count is greater than 0
	count := int(countBig.Int64())
	if count > 0 {
		for i := 0; i < count; i++ {
			data, err := cm.CMAccountABI.Pack("getRoleMember", roleHash, big.NewInt(int64(i)))
			if err != nil {
				log.Fatalf("Failed to pack contract function call: %v", err)
			}

			callMsg := ethereum.CallMsg{
				To:   &cmAccountAddress,
				Data: data,
			}

			result, err := cm.ethClient.CallContract(context.Background(), callMsg, nil)
			if err != nil {
				log.Fatalf("Failed to call contract function: %v", err)
				return nil, err
			}

			var address common.Address
			err = cm.CMAccountABI.UnpackIntoInterface(&address, "getRoleMember", result)
			if err != nil {
				log.Fatalf("Failed to unpack result: %v", err)
				return nil, err
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

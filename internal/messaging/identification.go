package messaging

import (
	"context"
	"math/big"
	"net/url"
	"strings"

	cmaccountscache "github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts_cache"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	"maunium.net/go/mautrix/id"
)

var _ IdentificationHandler = (*evmIdentificationHandler)(nil)

var roleHash = crypto.Keccak256Hash([]byte("CHEQUE_OPERATOR_ROLE"))

type IdentificationHandler interface {
	getFirstBotUserIDFromCMAccountAddress(common.Address) (id.UserID, error)
}

func NewIdentificationHandler(
	ethClient *ethclient.Client,
	logger *zap.SugaredLogger,
	cmAccountAddress common.Address,
	matrixHost url.URL,
	cmAccounts cmaccountscache.CMAccountsCache,
) (IdentificationHandler, error) {
	return &evmIdentificationHandler{
		ethClient:          ethClient,
		matrixHost:         matrixHost.String(),
		myCMAccountAddress: cmAccountAddress,
		cmAccounts:         cmAccounts,
		logger:             logger,
	}, nil
}

type evmIdentificationHandler struct {
	ethClient          *ethclient.Client
	matrixHost         string
	myCMAccountAddress common.Address
	cmAccounts         cmaccountscache.CMAccountsCache
	logger             *zap.SugaredLogger
}

func (ih *evmIdentificationHandler) getFirstBotUserIDFromCMAccountAddress(cmAccountAddress common.Address) (id.UserID, error) {
	botAddress, err := ih.getFirstBotFromCMAccountAddress(cmAccountAddress)
	if err != nil {
		return "", err
	}

	return UserIDFromAddress(botAddress, ih.matrixHost), nil
}

func (ih *evmIdentificationHandler) getFirstBotFromCMAccountAddress(cmAccountAddress common.Address) (common.Address, error) {
	bots, err := ih.getAllBotAddressesFromCMAccountAddress(cmAccountAddress)
	if err != nil {
		return common.Address{}, err
	}
	return bots[0], nil
}

func (ih *evmIdentificationHandler) getAllBotAddressesFromCMAccountAddress(cmAccountAddress common.Address) ([]common.Address, error) {
	cmAccount, err := ih.cmAccounts.Get(cmAccountAddress)
	if err != nil {
		ih.logger.Errorf("Failed to get cm account: %v", err)
		return nil, err
	}

	countBig, err := cmAccount.GetRoleMemberCount(&bind.CallOpts{Context: context.TODO()}, roleHash)
	if err != nil {
		ih.logger.Errorf("Failed to call contract function: %v", err)
		return nil, err
	}

	count := countBig.Int64()
	botsAddresses := make([]common.Address, 0, count)
	for i := int64(0); i < count; i++ {
		address, err := cmAccount.GetRoleMember(&bind.CallOpts{Context: context.TODO()}, roleHash, big.NewInt(i))
		if err != nil {
			ih.logger.Errorf("Failed to call contract function: %v", err)
			continue
		}
		botsAddresses = append(botsAddresses, address)
	}

	return botsAddresses, nil
}

func UserIDFromAddress(address common.Address, host string) id.UserID {
	return id.NewUserID(strings.ToLower(address.Hex()), host)
}

func addressFromUserID(userID id.UserID) common.Address {
	return common.HexToAddress(userID.Localpart())
}

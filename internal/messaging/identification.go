package messaging

import (
	"context"
	"math/big"
	"net/url"
	"strings"

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
	matrixHost         string
	myCMAccountAddress common.Address
	cmAccounts         *lru.Cache[common.Address, *cmaccount.Cmaccount]
	logger             *zap.SugaredLogger
}

type IdentificationHandler interface {
	getFirstBotUserIDFromCMAccountAddress(common.Address) (id.UserID, error)
}

func NewIdentificationHandler(
	ethClient *ethclient.Client,
	logger *zap.SugaredLogger,
	cmAccountAddress common.Address,
	matrixHost url.URL,
) (IdentificationHandler, error) {
	cmAccountsCache, err := lru.New[common.Address, *cmaccount.Cmaccount](cmAccountsCacheSize)
	if err != nil {
		return nil, err
	}

	return &evmIdentificationHandler{
		ethClient:          ethClient,
		matrixHost:         matrixHost.String(),
		myCMAccountAddress: cmAccountAddress,
		cmAccounts:         cmAccountsCache,
		logger:             logger,
	}, nil
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
	cmAccount, ok := ih.cmAccounts.Get(cmAccountAddress)
	if !ok {
		var err error
		cmAccount, err = cmaccount.NewCmaccount(cmAccountAddress, ih.ethClient)
		if err != nil {
			ih.logger.Errorf("Failed to get cm Account: %v", err)
			return nil, err
		}
		ih.cmAccounts.Add(cmAccountAddress, cmAccount)
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

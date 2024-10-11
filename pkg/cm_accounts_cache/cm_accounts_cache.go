package cmaccountscache

import (
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru/v2"
)

var _ CMAccountsCache = (*cmAccountsCache)(nil)

type CMAccountsCache interface {
	Get(cmAccountAddr common.Address) (*cmaccount.Cmaccount, error)
}

type cmAccountsCache struct {
	ethClient  *ethclient.Client
	cmAccounts *lru.Cache[common.Address, *cmaccount.Cmaccount]
}

func NewCMAccountsCache(size int, ethClient *ethclient.Client) (CMAccountsCache, error) {
	cmAccounts, err := lru.New[common.Address, *cmaccount.Cmaccount](size)
	if err != nil {
		return nil, err
	}
	return &cmAccountsCache{cmAccounts: cmAccounts, ethClient: ethClient}, nil
}

func (c *cmAccountsCache) Get(cmAccountAddr common.Address) (*cmaccount.Cmaccount, error) {
	cmAccount, ok := c.cmAccounts.Get(cmAccountAddr)
	if ok {
		return cmAccount, nil
	}

	cmaccount, err := cmaccount.NewCmaccount(cmAccountAddr, c.ethClient)
	if err != nil {
		return nil, err
	}
	c.cmAccounts.Add(cmAccountAddr, cmaccount)

	return cmaccount, nil
}

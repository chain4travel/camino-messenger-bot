// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package blockchain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"sync"

	e2eCommon "github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/common"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtokenoperator"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccountmanager"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/erc1967proxy"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/nullusd"
	"github.com/chain4travel/caminogoeth-compat/caminoethvm/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const bookingTokenOperatorLibName = "12bd2f62b73a470fe0f6e02c33045f3191" //nolint:gosec // this is not credentials.

var (
	kycAdminRole                = big.NewInt(0b100)
	cmAccountNativeTokenPrefund = big.NewInt(0).Mul(e2eCommon.CAM, big.NewInt(100))

	ErrorAddServiceTxFailed = errors.New("failed to issue AddService tx")
)

func newClient(
	ctx context.Context,
	nodeURI string,
	prefundedKeys []*ecdsa.PrivateKey,
	adminKey *ecdsa.PrivateKey,
) (*Client, error) {
	chainRPCURL := "ws://" + nodeURI + "/ext/bc/C/ws"
	ethClient, err := ethclient.Dial(chainRPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the Ethereum client: %w", err)
	}

	ethChainID, err := ethClient.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	adminContract, err := contracts.NewCaminoAdmin(evmAdminContract, ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin contract binding: %w", err)
	}

	return &Client{
		nodeURI:       nodeURI,
		chainRPCURL:   chainRPCURL,
		ethClient:     ethClient,
		ethChainID:    ethChainID,
		prefundedKeys: prefundedKeys,
		adminKey:      adminKey,
		adminContract: adminContract,
		nonces:        make(map[common.Address]uint64),
	}, nil
}

type Client struct {
	NullUSD      *nullusd.Nullusd
	BookingToken *bookingtoken.Bookingtoken

	prefundedKeys               []*ecdsa.PrivateKey
	adminKey                    *ecdsa.PrivateKey
	nodeURI                     string
	chainRPCURL                 string
	ethClient                   *ethclient.Client
	ethChainID                  *big.Int
	bookingTokenContractAddress common.Address
	cmAccountManager            *cmaccountmanager.Cmaccountmanager
	cmAccountManagerAddress     common.Address
	adminContract               *contracts.CaminoAdmin

	nonces      map[common.Address]uint64
	noncesMutex sync.Mutex
}

func (c *Client) ETHClient() *ethclient.Client {
	return c.ethClient
}

func (c *Client) PrefundedKeys() []*ecdsa.PrivateKey {
	return c.prefundedKeys
}

func (c *Client) ChainRPCURL() string {
	return c.chainRPCURL
}

func (c *Client) BookingTokenContractAddress() common.Address {
	return c.bookingTokenContractAddress
}

func (c *Client) CreateCMAccount(ctx context.Context, owner *ecdsa.PrivateKey) (common.Address, *cmaccount.Cmaccount, error) {
	prefundAmount, err := c.cmAccountManager.GetPrefundAmount(&bind.CallOpts{Context: ctx})
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to get cm account creation prefund amount: %w", err)
	}

	transactor, err := c.transactor(ctx, c.adminKey, common.Big0)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	approveTx, err := c.NullUSD.Approve(transactor, c.cmAccountManagerAddress, prefundAmount)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to issue nullUSD.Approve tx: %w", err)
	}
	if _, err = c.waitTxSucceed(ctx, approveTx); err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to wait for nullUSD.Approve tx to succeed: %w", err)
	}

	transactor, err = c.transactor(ctx, c.adminKey, cmAccountNativeTokenPrefund)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	ownerAddr := crypto.PubkeyToAddress(owner.PublicKey)

	tx, err := c.cmAccountManager.CreateCMAccount(
		transactor,
		ownerAddr,
		ownerAddr,
	)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to issue cmAccountManager.CreateCMAccount tx: %w", err)
	}

	receipt, err := c.waitTxSucceed(ctx, tx)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to wait for cmAccountManager.CreateCMAccount tx to succeed: %w", err)
	}

	for _, log := range receipt.Logs {
		event, err := c.cmAccountManager.ParseCMAccountCreated(*log)
		if err == nil {
			cmAccount, err := cmaccount.NewCmaccount(event.Account, c.ethClient)
			if err != nil {
				return common.Address{}, nil, fmt.Errorf("failed to create cm account binding: %w", err)
			}

			return event.Account, cmAccount, nil
		}
	}

	return common.Address{}, nil, fmt.Errorf("failed to parse CMAccountCreated event from receipt logs")
}

func (c *Client) AddBotToCMAccount(
	ctx context.Context,
	cmAccountAddress common.Address,
	cmAccountOwnerKey *ecdsa.PrivateKey,
	botAddr common.Address,
) error {
	transactor, err := c.transactor(ctx, cmAccountOwnerKey, nil)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}

	cmAccount, err := c.CMAccount(cmAccountAddress)
	if err != nil {
		return fmt.Errorf("failed to get cm account binding: %w", err)
	}

	tx, err := cmAccount.AddMessengerBot(transactor, botAddr, common.Big0)
	if err != nil {
		return fmt.Errorf("failed to issue AddMessengerBot tx: %w", err)
	}

	if _, err := c.waitTxSucceed(ctx, tx); err != nil {
		return fmt.Errorf("failed to wait for AddMessengerBot tx to succeed: %w", err)
	}

	return nil
}

func (c *Client) AddCMService(
	ctx context.Context,
	cmAccountAddress common.Address,
	cmAccountOwnerKey *ecdsa.PrivateKey,
	serviceName string,
	serviceFee int64,
) error {
	transactor, err := c.transactor(ctx, cmAccountOwnerKey, nil)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}

	cmAccount, err := c.CMAccount(cmAccountAddress)
	if err != nil {
		return fmt.Errorf("failed to get cm account binding: %w", err)
	}

	tx, err := cmAccount.AddService(
		transactor,
		serviceName,
		big.NewInt(serviceFee),
		false,
		[]string{},
	)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrorAddServiceTxFailed, err)
	}

	if _, err := c.waitTxSucceed(ctx, tx); err != nil {
		return fmt.Errorf("failed to wait for AddCMService tx to succeed: %w", err)
	}

	return nil
}

func (c *Client) RegisterCMService(
	ctx context.Context,
	serviceName string,
) error {
	transactor, err := c.transactor(ctx, c.adminKey, nil)
	if err != nil {
		return fmt.Errorf("failed to create admin transactor: %w", err)
	}

	tx, err := c.cmAccountManager.RegisterService(transactor, serviceName)
	if err != nil {
		return fmt.Errorf("failed to issue RegisterService tx: %w", err)
	}

	if _, err := c.waitTxSucceed(ctx, tx); err != nil {
		return fmt.Errorf("failed to wait for RegisterService tx to succeed: %w", err)
	}

	return nil
}

func (c *Client) RegisterCMServices(
	ctx context.Context,
	serviceNames ...string,
) error {
	for _, serviceName := range serviceNames {
		if err := c.RegisterCMService(ctx, serviceName); err != nil {
			return fmt.Errorf("failed to register service %s: %w", serviceName, err)
		}
	}
	return nil
}

func (c *Client) Transfer(
	ctx context.Context,
	from *ecdsa.PrivateKey,
	to common.Address,
	amount *big.Int,
) error {
	gasPrice, err := c.ethClient.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to suggest gas price: %w", err)
	}

	nonce, err := c.nextNonce(ctx, crypto.PubkeyToAddress(from.PublicKey))
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	unsignedTx := types.NewTransaction(nonce, to, amount, 30000, gasPrice, nil)

	signedTx, err := types.SignTx(unsignedTx, types.NewEIP155Signer(c.ethChainID), from)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = c.ethClient.SendTransaction(ctx, signedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	if _, err := c.waitTxSucceed(ctx, signedTx); err != nil {
		return fmt.Errorf("failed to wait for transaction to succeed: %w", err)
	}

	return nil
}

func (c *Client) BalanceOf(ctx context.Context, addr common.Address) (*big.Int, error) {
	balance, err := c.ethClient.BalanceAt(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}
	return balance, nil
}

func (c *Client) CMAccount(addr common.Address) (*cmaccount.Cmaccount, error) {
	return cmaccount.NewCmaccount(addr, c.ethClient)
}

func (c *Client) SetKYC(ctx context.Context, account common.Address, isKYCVerified bool) error {
	transactor, err := c.transactor(ctx, c.adminKey, common.Big0)
	if err != nil {
		return fmt.Errorf("failed to create admin transactor: %w", err)
	}

	tx, err := c.adminContract.ApplyKycState(transactor, account, !isKYCVerified, evmAddrStateKYCVerified)
	if err != nil {
		return fmt.Errorf("failed to issue ApplyKycState tx: %w", err)
	}

	if _, err := c.waitTxSucceed(ctx, tx); err != nil {
		return fmt.Errorf("failed to wait for ApplyKycState tx to succeed: %w", err)
	}

	return nil
}

func (c *Client) transactor(ctx context.Context, key *ecdsa.PrivateKey, value *big.Int) (*bind.TransactOpts, error) {
	nonce, err := c.nextNonce(ctx, crypto.PubkeyToAddress(key.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(key, c.ethChainID)
	transactor.Context = ctx
	transactor.Value = value
	transactor.Nonce = new(big.Int).SetUint64(nonce)
	return transactor, err
}

func (c *Client) waitTxSucceed(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	receipt, err := bind.WaitMined(ctx, c.ethClient, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction to be mined: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction failed with status %d", receipt.Status)
	}
	return receipt, nil
}

func (c *Client) prepareAdmin(ctx context.Context) error {
	adminAddress := crypto.PubkeyToAddress(c.adminKey.PublicKey)

	transactor, err := c.transactor(ctx, c.adminKey, common.Big0)
	if err != nil {
		return fmt.Errorf("failed to create default transactor: %w", err)
	}

	adminGrantRoleTx, err := c.adminContract.GrantRole(transactor, adminAddress, kycAdminRole)
	if err != nil {
		return fmt.Errorf("failed to issue adminContract.GrantRole tx: %w", err)
	}

	if _, err := c.waitTxSucceed(ctx, adminGrantRoleTx); err != nil {
		return fmt.Errorf("failed to wait for adminContract.GrantRole tx to succeed: %w", err)
	}

	// set KYC state for admin account, so we can deploy contracts with it
	if err := c.SetKYC(ctx, adminAddress, true); err != nil {
		return fmt.Errorf("failed to set KYC state for admin account: %w", err)
	}

	return nil
}

// TODO @evlekht we can use some kind of snapshotting to skip the process of deploying SCs if we'll use persistent network
func (c *Client) prepareCMBContracts(ctx context.Context) error {
	c.noncesMutex.Lock()
	defer c.noncesMutex.Unlock()

	adminAddress := crypto.PubkeyToAddress(c.adminKey.PublicKey)

	// prepare contracts, will be done in 4 blocks

	transactor, err := bind.NewKeyedTransactorWithChainID(c.adminKey, c.ethChainID)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}
	transactor.Context = ctx

	// prepare CM Account Manager proxy initialization data

	cmAccountManagerABI, err := abi.JSON(strings.NewReader(cmaccountmanager.CmaccountmanagerABI))
	if err != nil {
		return fmt.Errorf("failed to parse cmAccountManager ABI: %w", err)
	}
	cmAccountManagerInitializeData, err := cmAccountManagerABI.Pack(
		"initialize",
		adminAddress,     // admin
		adminAddress,     // pauser
		adminAddress,     // upgrader
		adminAddress,     // versioner
		adminAddress,     // developerWallet
		big.NewInt(1000), // developerFeeBp (10%)
	)
	if err != nil {
		return fmt.Errorf("failed to pack cmAccountManager.initialize data: %w", err)
	}

	// block 1 (deploy BookingToken impl, CM Account Manager impl, BookingTokenOperator, nullUSD)

	bookingTokenImplAddress, bookingTokenImplTx, _, err := bookingtoken.DeployBookingtoken(transactor, c.ethClient)
	if err != nil {
		return fmt.Errorf("failed to deploy bookingToken implementation contract: %w", err)
	}
	cmAccountManagerImplAddress, cmAccountManagerImplTx, _, err := cmaccountmanager.DeployCmaccountmanager(transactor, c.ethClient)
	if err != nil {
		return fmt.Errorf("failed to deploy cmAccountManager implementation contract: %w", err)
	}
	bookingTokenOperatorAddress, bookingTokenOperatorTx, _, err := bookingtokenoperator.DeployBookingtokenoperator(transactor, c.ethClient)
	if err != nil {
		return fmt.Errorf("failed to deploy bookingTokenOperator contract: %w", err)
	}
	nullUSDAddress, nullUSDTx, _, err := nullusd.DeployNullusd(transactor, c.ethClient)
	if err != nil {
		return fmt.Errorf("failed to deploy nullUSD contract: %w", err)
	}

	if _, err := c.waitTxSucceed(ctx, bookingTokenImplTx); err != nil {
		return fmt.Errorf("failed to wait for bookingToken implementation deployment tx to succeed: %w", err)
	}
	if _, err := c.waitTxSucceed(ctx, cmAccountManagerImplTx); err != nil {
		return fmt.Errorf("failed to wait for cmAccountManager implementation deployment tx to succeed: %w", err)
	}
	if _, err := c.waitTxSucceed(ctx, bookingTokenOperatorTx); err != nil {
		return fmt.Errorf("failed to wait for bookingTokenOperator deployment tx to succeed: %w", err)
	}
	if _, err := c.waitTxSucceed(ctx, nullUSDTx); err != nil {
		return fmt.Errorf("failed to wait for nullUSD deployment tx to succeed: %w", err)
	}

	// prepare cmAccount implementation bytecode with linked Booking Token Operator contract

	cmAccountABI, err := abi.JSON(strings.NewReader(cmaccount.CmaccountABI))
	if err != nil {
		return fmt.Errorf("failed to parse cmAccount ABI: %w", err)
	}

	bookingTokenOperatorLinkingRegExp := regexp.MustCompile("__\\$" + bookingTokenOperatorLibName + "\\$__")
	cmAccountImplBytecode := bookingTokenOperatorLinkingRegExp.ReplaceAllString(
		cmaccount.CmaccountBin,
		strings.ToLower(bookingTokenOperatorAddress.String()[2:]),
	)

	// create nullUSD bindings

	c.NullUSD, err = nullusd.NewNullusd(nullUSDAddress, c.ethClient)
	if err != nil {
		return fmt.Errorf("failed to create nullUSD binding: %w", err)
	}

	// block 2 (deploy CM Account Manager proxy, CM Account implementation)

	cmAccountManagerProxyAddress, cmAccountManagerProxyTx, _, err := erc1967proxy.DeployErc1967proxy(
		transactor,
		c.ethClient,
		cmAccountManagerImplAddress,
		cmAccountManagerInitializeData,
	)
	if err != nil {
		return fmt.Errorf("failed to deploy cmAccountManager proxy contract: %w", err)
	}

	cmAccountImplAddress, cmAccountImplTx, _, err := bind.DeployContract(transactor, cmAccountABI, common.FromHex(cmAccountImplBytecode), c.ethClient)
	if err != nil {
		return fmt.Errorf("failed to deploy cmAccount implementation contract: %w", err)
	}

	if _, err := c.waitTxSucceed(ctx, cmAccountManagerProxyTx); err != nil {
		return fmt.Errorf("failed to wait for cmAccountManager proxy deployment tx to succeed: %w", err)
	}
	if _, err := c.waitTxSucceed(ctx, cmAccountImplTx); err != nil {
		return fmt.Errorf("failed to wait for cmAccount implementation deployment tx to succeed: %w", err)
	}

	// create cmAccountManager binding

	c.cmAccountManager, err = cmaccountmanager.NewCmaccountmanager(cmAccountManagerProxyAddress, c.ethClient)
	if err != nil {
		return fmt.Errorf("failed to create cmAccountManager binding: %w", err)
	}
	c.cmAccountManagerAddress = cmAccountManagerProxyAddress

	// prepare Booking Token proxy initialization data

	bookingTokenABI, err := abi.JSON(strings.NewReader(bookingtoken.BookingtokenABI))
	if err != nil {
		return fmt.Errorf("failed to parse cmAccountManager ABI: %w", err)
	}
	bookingTokenInitializeData, err := bookingTokenABI.Pack(
		"initialize",
		cmAccountManagerProxyAddress, // CM Account Manager address
		adminAddress,                 // defaultAdmin
		adminAddress,                 // upgrader
	)
	if err != nil {
		return fmt.Errorf("failed to pack bookingToken.initialize data: %w", err)
	}

	// block 3 (deploy Booking Token proxy, grant cmAccountManager roles for registering services and setting serviceFeeToken)

	serviceRegistryAdminRole, err := c.cmAccountManager.SERVICEREGISTRYADMINROLE(&bind.CallOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	grantServiceRegistryAdminRoleTx, err := c.cmAccountManager.GrantRole(transactor, serviceRegistryAdminRole, adminAddress)
	if err != nil {
		return fmt.Errorf("failed to issue cmAccountManager.GrantRole (serviceRegistryAdminRole): %w", err)
	}

	serviceFeeTokenAdminRole, err := c.cmAccountManager.SERVICEFEETOKENADMINROLE(&bind.CallOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	grantServiceFeeTokenAdminRoleTx, err := c.cmAccountManager.GrantRole(transactor, serviceFeeTokenAdminRole, adminAddress)
	if err != nil {
		return fmt.Errorf("failed to issue cmAccountManager.GrantRole (serviceFeeTokenAdminRole): %w", err)
	}

	bookingTokenProxyAddress, bookingTokenProxyTx, _, err := erc1967proxy.DeployErc1967proxy(
		transactor,
		c.ethClient,
		bookingTokenImplAddress,
		bookingTokenInitializeData,
	)
	if err != nil {
		return fmt.Errorf("failed to deploy bookingToken proxy contract: %w", err)
	}

	if _, err := c.waitTxSucceed(ctx, grantServiceRegistryAdminRoleTx); err != nil {
		return fmt.Errorf("failed to wait for cmAccountManager.GrantRole (serviceRegistryAdminRole) tx to succeed: %w", err)
	}
	if _, err := c.waitTxSucceed(ctx, grantServiceFeeTokenAdminRoleTx); err != nil {
		return fmt.Errorf("failed to wait for cmAccountManager.GrantRole (serviceFeeTokenAdminRole) tx to succeed: %w", err)
	}
	if _, err := c.waitTxSucceed(ctx, bookingTokenProxyTx); err != nil {
		return fmt.Errorf("failed to wait for bookingToken proxy deployment tx to succeed: %w", err)
	}

	// create bookingToken binding

	c.BookingToken, err = bookingtoken.NewBookingtoken(bookingTokenProxyAddress, c.ethClient)
	if err != nil {
		return fmt.Errorf("failed to create bookingToken binding: %w", err)
	}

	// block 4 (ReinitializeV2 Booking Token, link contracts, set service fee token in cmAccountManager)

	setBookingTokenAddressTx, err := c.cmAccountManager.SetBookingTokenAddress(
		transactor,
		bookingTokenProxyAddress,
	)
	if err != nil {
		return fmt.Errorf("failed to issue cmAccountManager.SetBookingTokenAddress tx: %w", err)
	}

	setAccountImplementationTx, err := c.cmAccountManager.SetAccountImplementation(
		transactor,
		cmAccountImplAddress,
	)
	if err != nil {
		return fmt.Errorf("failed to issue cmAccountManager.SetAccountImplementation tx: %w", err)
	}

	reinitializeV2Tx, err := c.BookingToken.ReinitializeV2(transactor, "BookingToken", "BToken")
	if err != nil {
		return fmt.Errorf("failed to issue bookingToken.ReinitializeV2 tx: %w", err)
	}

	minExpirationTimestampDiffRole, err := c.BookingToken.MINEXPIRATIONADMINROLE(&bind.CallOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	grantRoleTx, err := c.BookingToken.GrantRole(transactor, minExpirationTimestampDiffRole, adminAddress)
	if err != nil {
		return fmt.Errorf("failed to issue BookingToken.GrantRole (minExpirationTimestampDiffRole) tx: %w", err)
	}

	if _, err := c.waitTxSucceed(ctx, grantRoleTx); err != nil {
		return fmt.Errorf("failed to wait for BookingToken.GrantRole (minExpirationTimestampDiffRole) tx to succeed: %w", err)
	}

	updateExpirationTx, err := c.BookingToken.SetMinExpirationTimestampDiff(transactor, big.NewInt(e2eCommon.MinBuyableUntilInContract))
	if err != nil {
		return fmt.Errorf("failed to issue bookingToken.SetMinExpirationTimestampDiff tx: %w", err)
	}

	setServiceFeeTokenTx, err := c.cmAccountManager.SetServiceFeeToken(transactor, nullUSDAddress)
	if err != nil {
		return fmt.Errorf("failed to issue cmAccountManager.SetServiceFeeToken tx: %w", err)
	}

	if _, err := c.waitTxSucceed(ctx, setBookingTokenAddressTx); err != nil {
		return fmt.Errorf("failed to wait for cmAccountManager.SetBookingTokenAddress tx to succeed: %w", err)
	}
	if _, err := c.waitTxSucceed(ctx, setAccountImplementationTx); err != nil {
		return fmt.Errorf("failed to wait for cmAccountManager.SetAccountImplementation tx to succeed: %w", err)
	}
	if _, err := c.waitTxSucceed(ctx, reinitializeV2Tx); err != nil {
		return fmt.Errorf("failed to wait for bookingToken.ReinitializeV2 tx to succeed: %w", err)
	}
	if _, err := c.waitTxSucceed(ctx, updateExpirationTx); err != nil {
		return fmt.Errorf("failed to wait for bookingToken.SetMinExpirationTimestampDiff tx to succeed: %w", err)
	}
	if _, err := c.waitTxSucceed(ctx, setServiceFeeTokenTx); err != nil {
		return fmt.Errorf("failed to wait for cmAccountManager.SetServiceFeeToken tx to succeed: %w", err)
	}

	c.bookingTokenContractAddress = bookingTokenProxyAddress

	nonce, err := c.ethClient.PendingNonceAt(ctx, adminAddress)
	if err != nil {
		return fmt.Errorf("failed to get nonce for admin address: %w", err)
	}
	c.nonces[adminAddress] = nonce

	return nil
}

func (c *Client) nextNonce(ctx context.Context, addr common.Address) (uint64, error) {
	c.noncesMutex.Lock()
	defer c.noncesMutex.Unlock()

	nonce, ok := c.nonces[addr]
	if !ok {
		var err error
		nonce, err = c.ethClient.PendingNonceAt(ctx, addr)
		if err != nil {
			return 0, fmt.Errorf("failed to get nonce for address %s: %w", addr.Hex(), err)
		}
	}

	c.nonces[addr] = nonce + 1

	return nonce, nil
}

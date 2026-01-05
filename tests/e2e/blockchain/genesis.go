// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/chain4travel/caminogoeth-compat/caminoethvm/core"
	"github.com/chain4travel/caminogoeth-compat/caminoethvm/params"
	"github.com/chain4travel/caminogoeth-compat/caminogo/address"
	"github.com/chain4travel/caminogoeth-compat/caminogo/constants"
	"github.com/chain4travel/caminogoeth-compat/caminogo/genesis"
	"github.com/chain4travel/caminogoeth-compat/caminogo/ids"
	"github.com/chain4travel/caminogoeth-compat/caminogo/secp256k1"
	"github.com/chain4travel/caminogoeth-compat/caminogo/units"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/chain4travel/camino-messenger-bot/v12/pkg/conversion"
	e2eCommon "github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/common"
)

var evmAdminFunds = big.NewInt(0).Mul(big.NewInt(1_000_000), e2eCommon.CAM)

type allocation struct {
	owner      *secp256k1.PrivateKey
	addrStates genesis.AddressStates
	p          uint64
	x          uint64
	c          *big.Int
}

type validator struct {
	owner  *secp256k1.PrivateKey
	nodeID ids.NodeID
}

// validators will be allocated with MaxValidatorStake
func generateGenesis(
	startTime time.Time,
	initialAdmin *secp256k1.PrivateKey,
	validators []validator,
	funds []allocation,
	networkID uint32,
) (string, error) {
	evmGenesis, err := generateEVMGenesis(startTime, initialAdmin, evmChainIDs[networkID], funds)
	if err != nil {
		return "", fmt.Errorf("failed to generate evm genesis: %w", err)
	}

	params := genesis.GetStakingConfig(networkID)

	hrp := constants.GetHRP(networkID)
	pAdminAddr, err := address.Format("P", hrp, initialAdmin.Address().Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to format admin address: %w", err)
	}

	memoCounter := 0
	allocations := make([]genesis.UnparsedCaminoAllocation, len(validators))
	for i, validator := range validators {
		validatorAddrStr, err := address.Format("P", hrp, validator.owner.Address().Bytes())
		if err != nil {
			return "", fmt.Errorf("failed to format validator address: %w", err)
		}

		allocations[i] = genesis.UnparsedCaminoAllocation{
			ETHAddr:  crypto.PubkeyToAddress(validator.owner.ToECDSA().PublicKey).Hex(),
			AVAXAddr: validatorAddrStr,
			AddressStates: genesis.AddressStates{
				ConsortiumMember: true,
				KYCVerified:      true,
			},
			PlatformAllocations: []genesis.UnparsedPlatformAllocation{{
				Amount:            params.MaxValidatorStake,
				NodeID:            validator.nodeID.String(),
				ValidatorDuration: conversion.MustDurationToUInt64(params.MaxStakeDuration / time.Second),
				Memo:              strconv.Itoa(memoCounter),
			}},
		}
		memoCounter++
	}

	for _, fund := range funds {
		ownerAddrStr, err := address.Format("P", hrp, fund.owner.Address().Bytes())
		if err != nil {
			return "", fmt.Errorf("failed to format owner address: %w", err)
		}

		ownerCAddr := crypto.PubkeyToAddress(fund.owner.ToECDSA().PublicKey)

		allocations = append(allocations, genesis.UnparsedCaminoAllocation{
			ETHAddr:  ownerCAddr.Hex(),
			AVAXAddr: ownerAddrStr,
			XAmount:  fund.x,
			PlatformAllocations: []genesis.UnparsedPlatformAllocation{{
				Amount: fund.p,
				Memo:   strconv.Itoa(memoCounter),
			}},
		})
		memoCounter++
	}

	allocations = append(allocations, genesis.UnparsedCaminoAllocation{
		ETHAddr:  crypto.PubkeyToAddress(initialAdmin.ToECDSA().PublicKey).Hex(),
		AVAXAddr: pAdminAddr,
		XAmount:  1_000_000 * units.Avax,
		AddressStates: genesis.AddressStates{
			ConsortiumMember: true,
			KYCVerified:      true,
		},
		PlatformAllocations: []genesis.UnparsedPlatformAllocation{{
			Amount: 1_000_000 * units.Avax,
			Memo:   strconv.Itoa(memoCounter),
		}},
	})

	genesis := &genesis.UnparsedConfig{
		NetworkID: networkID,
		Camino: &genesis.UnparsedCamino{
			VerifyNodeSignature: true,
			LockModeBondDeposit: true,
			InitialAdmin:        pAdminAddr,
			Allocations:         allocations,
		},
		StartTime:     conversion.MustInt64ToUInt64(startTime.Unix()),
		CChainGenesis: evmGenesis,
	}

	genesisBytes, err := json.Marshal(genesis)
	if err != nil {
		return "", fmt.Errorf("failed to marshal genesis: %w", err)
	}

	genesisStr := base64.StdEncoding.EncodeToString(genesisBytes)

	return genesisStr, nil
}

func generateEVMGenesis(
	startTime time.Time,
	initialAdmin *secp256k1.PrivateKey,
	chainID *big.Int,
	funds []allocation,
) (string, error) {
	cAdminAddr := crypto.PubkeyToAddress(initialAdmin.ToECDSA().PublicKey)

	allocations := map[common.Address]core.GenesisAccount{
		common.HexToAddress("0x0100000000000000000000000000000000000000"): {
			Balance: common.Big0,
			Code:    common.FromHex("0x7300000000000000000000000000000000000000003014608060405260043610603d5760003560e01c80631e010439146042578063b6510bb314606e575b600080fd5b605c60048036036020811015605657600080fd5b503560b1565b60408051918252519081900360200190f35b818015607957600080fd5b5060af60048036036080811015608e57600080fd5b506001600160a01b03813516906020810135906040810135906060013560b6565b005b30cd90565b836001600160a01b031681836108fc8690811502906040516000604051808303818888878c8acf9550505050505015801560f4573d6000803e3d6000fd5b505050505056fea26469706673582212201eebce970fe3f5cb96bf8ac6ba5f5c133fc2908ae3dcd51082cfee8f583429d064736f6c634300060a0033"),
		},
		cAdminAddr: {Balance: evmAdminFunds},
	}

	for _, fund := range funds {
		ownerCAddr := crypto.PubkeyToAddress(fund.owner.ToECDSA().PublicKey)
		allocations[ownerCAddr] = core.GenesisAccount{
			Balance: fund.c,
		}
	}

	evmGenesis := &core.Genesis{
		Config:       &params.ChainConfig{ChainID: chainID},
		InitialAdmin: cAdminAddr,
		Timestamp:    conversion.MustInt64ToUInt64(startTime.Unix()),
		ExtraData:    []byte{0},        // 0x00
		GasLimit:     100000000,        // 0x5f5e100
		Difficulty:   big.NewInt(0),    // 0x0
		Mixhash:      common.Hash{},    // 0x0000000000000000000000000000000000000000000000000000000000000000
		Coinbase:     common.Address{}, // 0x0000000000000000000000000000000000000000
		Alloc:        allocations,      //
		ParentHash:   common.Hash{},    // 0x0000000000000000000000000000000000000000000000000000000000000000
	}

	evmGenesisBytes, err := evmGenesis.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("failed to marshal evm genesis: %w", err)
	}

	return string(evmGenesisBytes), nil
}

// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package cmaccounts

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Cancellation interface {
	InitiateCancellationProposal(
		ctx context.Context,
		botKey *ecdsa.PrivateKey,
		cmAccountAddress common.Address,
		tokenID *big.Int,
		refundAmount *big.Int,
		reason uint16,
		reasonVersion uint16,
	) (*types.Receipt, error)

	CounterCancellation(
		ctx context.Context,
		botKey *ecdsa.PrivateKey,
		cmAccountAddress common.Address,
		tokenID *big.Int,
		refundAmount *big.Int,
		reason uint16,
		reasonVersion uint16,
	) (*types.Receipt, error)

	AcceptCancellationProposal(
		ctx context.Context,
		botKey *ecdsa.PrivateKey,
		cmAccountAddress common.Address,
		tokenID *big.Int,
		refundAmount *big.Int,
	) (*types.Receipt, error)

	RejectCancellationProposal(
		ctx context.Context,
		botKey *ecdsa.PrivateKey,
		cmAccountAddress common.Address,
		tokenID *big.Int,
		reason uint16,
		reasonVersion uint16,
	) (*types.Receipt, error)

	WithdrawCancellation(
		ctx context.Context,
		botKey *ecdsa.PrivateKey,
		cmAccountAddress common.Address,
		tokenID *big.Int,
		reason uint16,
		reasonVersion uint16,
	) (*types.Receipt, error)

	FinalizeCancellation(
		ctx context.Context,
		botKey *ecdsa.PrivateKey,
		cmAccountAddress common.Address,
		tokenID *big.Int,
		refundAmount *big.Int,
	) (*types.Receipt, error)
}

func (s *service) InitiateCancellationProposal(
	ctx context.Context,
	botKey *ecdsa.PrivateKey,
	cmAccountAddress common.Address,
	tokenID *big.Int,
	refundAmount *big.Int,
	reason uint16,
	reasonVersion uint16,
) (*types.Receipt, error) {
	cmAccount, err := s.CMAccount(cmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get cmAccount contract instance: %w", err)
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(botKey, s.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := cmAccount.InitiateCancellation(
		transactor,
		tokenID,
		refundAmount,
		reason,
		reasonVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate cancellation: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, s.ethClient, tx)
	if err != nil {
		return nil, err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction failed: %v", receipt)
	}

	return receipt, nil
}

func (s *service) CounterCancellation(
	ctx context.Context,
	botKey *ecdsa.PrivateKey,
	cmAccountAddress common.Address,
	tokenID *big.Int,
	refundAmount *big.Int,
	reason uint16,
	reasonVersion uint16,
) (*types.Receipt, error) {
	cmAccount, err := s.CMAccount(cmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get cmAccount contract instance: %w", err)
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(botKey, s.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := cmAccount.CounterCancellation(
		transactor,
		tokenID,
		refundAmount,
		reason,
		reasonVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to counter cancellation proposal: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, s.ethClient, tx)
	if err != nil {
		return nil, err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction failed: %v", receipt)
	}

	return receipt, nil
}

func (s *service) AcceptCancellationProposal(
	ctx context.Context,
	botKey *ecdsa.PrivateKey,
	cmAccountAddress common.Address,
	tokenID *big.Int,
	refundAmount *big.Int,
) (*types.Receipt, error) {
	cmAccount, err := s.CMAccount(cmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get cmAccount contract instance: %w", err)
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(botKey, s.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := cmAccount.AcceptCancellation(
		transactor,
		tokenID,
		refundAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to accept cancellation: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, s.ethClient, tx)
	if err != nil {
		return nil, err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction failed: %v", receipt)
	}

	return receipt, nil
}

func (s *service) RejectCancellationProposal(
	ctx context.Context,
	botKey *ecdsa.PrivateKey,
	cmAccountAddress common.Address,
	tokenID *big.Int,
	reason uint16,
	reasonVersion uint16,
) (*types.Receipt, error) {
	cmAccount, err := s.CMAccount(cmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get cmAccount contract instance: %w", err)
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(botKey, s.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := cmAccount.RejectCancellation(
		transactor,
		tokenID,
		reason,
		reasonVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reject cancellation: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, s.ethClient, tx)
	if err != nil {
		return nil, err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction failed: %v", receipt)
	}

	return receipt, nil
}

func (s *service) WithdrawCancellation(
	ctx context.Context,
	botKey *ecdsa.PrivateKey,
	cmAccountAddress common.Address,
	tokenID *big.Int,
	reason uint16,
	reasonVersion uint16,
) (*types.Receipt, error) {
	cmAccount, err := s.CMAccount(cmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get cmAccount contract instance: %w", err)
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(botKey, s.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := cmAccount.WithdrawCancellation(
		transactor,
		tokenID,
		reason,
		reasonVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to withdraw cancellation: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, s.ethClient, tx)
	if err != nil {
		return nil, err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction failed: %v", receipt)
	}

	return receipt, nil
}

func (s *service) FinalizeCancellation(
	ctx context.Context,
	botKey *ecdsa.PrivateKey,
	cmAccountAddress common.Address,
	tokenID *big.Int,
	refundAmount *big.Int,
) (*types.Receipt, error) {
	cmAccount, err := s.CMAccount(cmAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get cmAccount contract instance: %w", err)
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(botKey, s.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := cmAccount.FinalizeCancellation(
		transactor,
		tokenID,
		refundAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize cancellation: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, s.ethClient, tx)
	if err != nil {
		return nil, err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction failed: %v", receipt)
	}

	return receipt, nil
}

// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.
package cancellation

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/v12/internal/common"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/version"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v12/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/conversion"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v2/cancellationv2grpc"
	cancellationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v2"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

const (
	cancellationReasonVersion = 1
	withdrawReasonVersion     = 1
	rejectReasonVersion       = 1
	counterReasonVersion      = 1
)

var _ cancellationv2grpc.CancellationServiceServer = (*cancellationV2Service)(nil)

func NewService(
	logger *zap.SugaredLogger,
	botKey *ecdsa.PrivateKey,
	cmAccountAddr ethCommon.Address,
	cmAccounts cmaccounts.Service,
	priceHandler common.PriceHandler,
) cancellationv2grpc.CancellationServiceServer {
	return &cancellationV2Service{
		botKey:        botKey,
		cmAccountAddr: cmAccountAddr,
		logger:        logger,
		priceHandler:  priceHandler,
		cmAccounts:    cmAccounts,
	}
}

type cancellationV2Service struct {
	botKey        *ecdsa.PrivateKey
	cmAccountAddr ethCommon.Address
	logger        *zap.SugaredLogger
	priceHandler  common.PriceHandler
	cmAccounts    cmaccounts.Service
}

func (s *cancellationV2Service) InitiateCancellation(
	ctx context.Context,
	request *cancellationv2.InitiateCancellationRequest,
) (*cancellationv2.InitiateCancellationResponse, error) {
	response := &cancellationv2.InitiateCancellationResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: version.VersionV4},
		},
	}

	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV4(ctx, request.RefundAmount)
	if err != nil {
		err := fmt.Errorf("error getting price and token: %w", err)
		s.logger.Error(err)
		return response, err
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		err := fmt.Errorf("error converting reason to uint16: %w", err)
		s.logger.Error(err)
		return response, err
	}

	receipt, err := s.cmAccounts.InitiateCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount, reasonValue, cancellationReasonVersion)
	if err != nil {
		err := fmt.Errorf("error initiating cancellation proposal: %w", err)
		s.logger.Error(err)
		return response, err
	}

	response.TransactionId = &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	s.logger.Infof("Initiated cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV2Service) CounterCancellation(
	ctx context.Context,
	request *cancellationv2.CounterCancellationRequest,
) (*cancellationv2.CounterCancellationResponse, error) {
	response := &cancellationv2.CounterCancellationResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: version.VersionV4},
		},
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV4(ctx, request.RefundAmount)
	if err != nil {
		err := fmt.Errorf("error getting price and token: %w", err)
		s.logger.Error(err)
		return response, err
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		err := fmt.Errorf("error converting reason to uint16: %w", err)
		s.logger.Error(err)
		return response, err
	}

	receipt, err := s.cmAccounts.CounterCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount, reasonValue, counterReasonVersion)
	if err != nil {
		err := fmt.Errorf("error countering cancellation proposal: %w", err)
		s.logger.Error(err)
		return response, err
	}

	response.TransactionId = &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	s.logger.Infof("Countered cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV2Service) AcceptCancellation(
	ctx context.Context,
	request *cancellationv2.AcceptCancellationRequest,
) (*cancellationv2.AcceptCancellationResponse, error) {
	response := &cancellationv2.AcceptCancellationResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: version.VersionV4},
		},
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV4(ctx, request.RefundAmount)
	if err != nil {
		err := fmt.Errorf("error getting price and payment token: %w", err)
		s.logger.Error(err)
		return response, err
	}

	receipt, err := s.cmAccounts.AcceptCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount)
	if err != nil {
		err := fmt.Errorf("error accepting cancellation proposal: %w", err)
		s.logger.Error(err)
		return response, err
	}

	response.TransactionId = &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	s.logger.Infof("Accepted cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV2Service) RejectCancellation(
	ctx context.Context,
	request *cancellationv2.RejectCancellationRequest,
) (*cancellationv2.RejectCancellationResponse, error) {
	response := &cancellationv2.RejectCancellationResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: version.VersionV4},
		},
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		err := fmt.Errorf("error converting reason to uint16: %w", err)
		s.logger.Error(err)
		return response, err
	}

	receipt, err := s.cmAccounts.RejectCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, reasonValue, rejectReasonVersion)
	if err != nil {
		err := fmt.Errorf("error rejecting cancellation proposal: %w", err)
		s.logger.Error(err)
		return response, err
	}

	response.TransactionId = &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	s.logger.Infof("Rejected cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV2Service) WithdrawCancellation(
	ctx context.Context,
	request *cancellationv2.WithdrawCancellationRequest,
) (*cancellationv2.WithdrawCancellationResponse, error) {
	response := &cancellationv2.WithdrawCancellationResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: version.VersionV4},
		},
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		err := fmt.Errorf("error converting reason to uint16: %w", err)
		s.logger.Error(err)
		return response, err
	}

	receipt, err := s.cmAccounts.WithdrawCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, reasonValue, withdrawReasonVersion)
	if err != nil {
		err := fmt.Errorf("error withdrawing cancellation proposal: %w", err)
		s.logger.Error(err)
		return response, err
	}

	response.TransactionId = &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	s.logger.Infof("Withdrawn cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV2Service) FinalizeCancellation(
	ctx context.Context,
	request *cancellationv2.FinalizeCancellationRequest,
) (*cancellationv2.FinalizeCancellationResponse, error) {
	response := &cancellationv2.FinalizeCancellationResponse{
		Header: &typesv4.ResponseHeader{
			BaseHeader: &typesv4.Header{Version: version.VersionV4},
		},
	}

	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV4(ctx, request.RefundAmount)
	if err != nil {
		err := fmt.Errorf("error getting price and token: %w", err)
		s.logger.Error(err)
		return response, err
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.FinalizeCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount)
	if err != nil {
		err := fmt.Errorf("error finalizing cancellation proposal: %w", err)
		s.logger.Error(err)
		return response, err
	}

	response.TransactionId = &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	s.logger.Infof("Finalized cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

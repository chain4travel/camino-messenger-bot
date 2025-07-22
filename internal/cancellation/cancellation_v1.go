// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.
package cancellation

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/v11/internal/common"
	"github.com/chain4travel/camino-messenger-bot/v11/internal/version"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v11/pkg/cm_accounts"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v1/cancellationv1grpc"
	cancellationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	cancellationReasonVersion = 1
	withdrawReasonVersion     = 1
	rejectReasonVersion       = 1
	counterReasonVersion      = 1
)

var _ cancellationv1grpc.CancellationServiceServer = (*cancellationV1Service)(nil)

func NewService(
	logger *zap.SugaredLogger,
	botKey *ecdsa.PrivateKey,
	cmAccountAddr ethCommon.Address,
	cmAccounts cmaccounts.Service,
	priceHandler common.PriceHandler,
) cancellationv1grpc.CancellationServiceServer {
	return &cancellationV1Service{
		botKey:        botKey,
		cmAccountAddr: cmAccountAddr,
		logger:        logger,
		priceHandler:  priceHandler,
		cmAccounts:    cmAccounts,
	}
}

type cancellationV1Service struct {
	botKey        *ecdsa.PrivateKey
	cmAccountAddr ethCommon.Address
	logger        *zap.SugaredLogger
	priceHandler  common.PriceHandler
	cmAccounts    cmaccounts.Service
}

func (s *cancellationV1Service) InitiateCancellation(
	ctx context.Context,
	request *cancellationv1.InitiateCancellationRequest,
) (*cancellationv1.InitiateCancellationResponse, error) {
	response := &cancellationv1.InitiateCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.Version},
		},
	}

	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV3(ctx, request.RefundAmount)
	if err != nil {
		err := fmt.Errorf("error getting price and token: %w", err)
		s.logger.Error(err)
		return response, err
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	reasonValue, err := uint16FromProtoEnumNumber(request.Reason.Number())
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

	response.TransactionId = &typesv3.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	s.logger.Infof("Initiated cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV1Service) CounterCancellation(
	ctx context.Context,
	request *cancellationv1.CounterCancellationRequest,
) (*cancellationv1.CounterCancellationResponse, error) {
	response := &cancellationv1.CounterCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.Version},
		},
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV3(ctx, request.RefundAmount)
	if err != nil {
		err := fmt.Errorf("error getting price and token: %w", err)
		s.logger.Error(err)
		return response, err
	}

	reasonValue, err := uint16FromProtoEnumNumber(request.Reason.Number())
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

	response.TransactionId = &typesv3.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	s.logger.Infof("Countered cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV1Service) AcceptCancellation(
	ctx context.Context,
	request *cancellationv1.AcceptCancellationRequest,
) (*cancellationv1.AcceptCancellationResponse, error) {
	response := &cancellationv1.AcceptCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.Version},
		},
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV3(ctx, request.RefundAmount)
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

	response.TransactionId = &typesv3.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	s.logger.Infof("Accepted cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV1Service) RejectCancellation(
	ctx context.Context,
	request *cancellationv1.RejectCancellationRequest,
) (*cancellationv1.RejectCancellationResponse, error) {
	response := &cancellationv1.RejectCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.Version},
		},
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	reasonValue, err := uint16FromProtoEnumNumber(request.Reason.Number())
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

	response.TransactionId = &typesv3.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	s.logger.Infof("Rejected cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV1Service) WithdrawCancellation(
	ctx context.Context,
	request *cancellationv1.WithdrawCancellationRequest,
) (*cancellationv1.WithdrawCancellationResponse, error) {
	response := &cancellationv1.WithdrawCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.Version},
		},
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	reasonValue, err := uint16FromProtoEnumNumber(request.Reason.Number())
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

	response.TransactionId = &typesv3.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	s.logger.Infof("Withdrawn cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV1Service) FinalizeCancellation(
	ctx context.Context,
	request *cancellationv1.FinalizeCancellationRequest,
) (*cancellationv1.FinalizeCancellationResponse, error) {
	response := &cancellationv1.FinalizeCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.Version},
		},
	}

	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV3(ctx, request.RefundAmount)
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

	response.TransactionId = &typesv3.EVMTransactionID{Hash: receipt.TxHash.Hex()}

	s.logger.Infof("Finalized cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

// Safely converts an int32 to uint16, returning an error if out of range.
func uint16FromProtoEnumNumber(value protoreflect.EnumNumber) (uint16, error) {
	if value < 0 || value > math.MaxUint16 {
		return 0, fmt.Errorf("value out of range for uint16: %d", value)
	}
	// TODO @VjeraTurk check if false positives are removed in newer versions
	// https://github.com/securego/gosec/issues/1212
	// https://github.com/securego/gosec/pull/1194

	// Otherwise lint.sh fails with G115: integer overflow conversion int32 -> uint16 (gosec)
	// nolint:gosec
	return uint16(value), nil
}

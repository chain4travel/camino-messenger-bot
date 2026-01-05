// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.
package cancellation

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/v12/internal/price"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/version"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v12/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/conversion"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v1/cancellationv1grpc"
	cancellationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
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
	priceHandler price.Handler,
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
	priceHandler  price.Handler
	cmAccounts    cmaccounts.Service
}

func (s *cancellationV1Service) InitiateCancellation(
	ctx context.Context,
	request *cancellationv1.InitiateCancellationRequest,
) (*cancellationv1.InitiateCancellationResponse, error) {
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV3(ctx, request.RefundAmount)
	if err != nil {
		errMsg := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMsg)
		return initiateCancellationErrorResponse(errMsg), nil
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMsg := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMsg)
		return initiateCancellationErrorResponse(errMsg), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.InitiateCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount, reasonValue, cancellationReasonVersion)
	if err != nil {
		errMsg := fmt.Sprintf("error initiating cancellation proposal: %v", err)
		s.logger.Error(errMsg)
		return initiateCancellationErrorResponse(errMsg), nil
	}

	response := &cancellationv1.InitiateCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		TransactionId: &typesv3.EVMTransactionID{Hash: receipt.TxHash.Hex()},
	}

	s.logger.Infof("Initiated cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV1Service) CounterCancellation(
	ctx context.Context,
	request *cancellationv1.CounterCancellationRequest,
) (*cancellationv1.CounterCancellationResponse, error) {
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV3(ctx, request.RefundAmount)
	if err != nil {
		errMsg := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMsg)
		return counterCancellationErrorResponse(errMsg), nil
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMsg := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMsg)
		return counterCancellationErrorResponse(errMsg), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.CounterCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount, reasonValue, counterReasonVersion)
	if err != nil {
		errMsg := fmt.Sprintf("error countering cancellation proposal: %v", err)
		s.logger.Error(errMsg)
		return counterCancellationErrorResponse(errMsg), nil
	}

	response := &cancellationv1.CounterCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		TransactionId: &typesv3.EVMTransactionID{Hash: receipt.TxHash.Hex()},
	}

	s.logger.Infof("Countered cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV1Service) AcceptCancellation(
	ctx context.Context,
	request *cancellationv1.AcceptCancellationRequest,
) (*cancellationv1.AcceptCancellationResponse, error) {
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV3(ctx, request.RefundAmount)
	if err != nil {
		errMsg := fmt.Sprintf("error getting price and payment token: %v", err)
		s.logger.Error(errMsg)
		return acceptCancellationErrorResponse(errMsg), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.AcceptCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount)
	if err != nil {
		errMsg := fmt.Sprintf("error accepting cancellation proposal: %v", err)
		s.logger.Error(errMsg)
		return acceptCancellationErrorResponse(errMsg), nil
	}

	response := &cancellationv1.AcceptCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		TransactionId: &typesv3.EVMTransactionID{Hash: receipt.TxHash.Hex()},
	}

	s.logger.Infof("Accepted cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV1Service) RejectCancellation(
	ctx context.Context,
	request *cancellationv1.RejectCancellationRequest,
) (*cancellationv1.RejectCancellationResponse, error) {
	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMsg := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMsg)
		return rejectCancellationErrorResponse(errMsg), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.RejectCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, reasonValue, rejectReasonVersion)
	if err != nil {
		errMsg := fmt.Sprintf("error rejecting cancellation proposal: %v", err)
		s.logger.Error(errMsg)
		return rejectCancellationErrorResponse(errMsg), nil
	}

	response := &cancellationv1.RejectCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		TransactionId: &typesv3.EVMTransactionID{Hash: receipt.TxHash.Hex()},
	}

	s.logger.Infof("Rejected cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV1Service) WithdrawCancellation(
	ctx context.Context,
	request *cancellationv1.WithdrawCancellationRequest,
) (*cancellationv1.WithdrawCancellationResponse, error) {
	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMsg := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMsg)
		return withdrawCancellationErrorResponse(errMsg), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.WithdrawCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, reasonValue, withdrawReasonVersion)
	if err != nil {
		errMsg := fmt.Sprintf("error withdrawing cancellation proposal: %v", err)
		s.logger.Error(errMsg)
		return withdrawCancellationErrorResponse(errMsg), nil
	}

	response := &cancellationv1.WithdrawCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		TransactionId: &typesv3.EVMTransactionID{Hash: receipt.TxHash.Hex()},
	}

	s.logger.Infof("Withdrawn cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func (s *cancellationV1Service) FinalizeCancellation(
	ctx context.Context,
	request *cancellationv1.FinalizeCancellationRequest,
) (*cancellationv1.FinalizeCancellationResponse, error) {
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV3(ctx, request.RefundAmount)
	if err != nil {
		errMsg := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMsg)
		return finalizeCancellationErrorResponse(errMsg), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.FinalizeCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount)
	if err != nil {
		errMsg := fmt.Sprintf("error finalizing cancellation proposal: %v", err)
		s.logger.Error(errMsg)
		return finalizeCancellationErrorResponse(errMsg), nil
	}

	response := &cancellationv1.FinalizeCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		TransactionId: &typesv3.EVMTransactionID{Hash: receipt.TxHash.Hex()},
	}

	s.logger.Infof("Finalized cancellation for token %s with tx: %s", tokenID.String(), response.TransactionId.Hash)

	return response, nil
}

func initiateCancellationErrorResponse(errorMessage string) *cancellationv1.InitiateCancellationResponse {
	return &cancellationv1.InitiateCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_FAILURE,
			Alerts: []*typesv1.Alert{{
				Message: errorMessage,
				Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
			}},
		},
	}
}

func counterCancellationErrorResponse(errorMessage string) *cancellationv1.CounterCancellationResponse {
	return &cancellationv1.CounterCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_FAILURE,
			Alerts: []*typesv1.Alert{{
				Message: errorMessage,
				Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
			}},
		},
	}
}

func acceptCancellationErrorResponse(errorMessage string) *cancellationv1.AcceptCancellationResponse {
	return &cancellationv1.AcceptCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_FAILURE,
			Alerts: []*typesv1.Alert{{
				Message: errorMessage,
				Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
			}},
		},
	}
}

func rejectCancellationErrorResponse(errorMessage string) *cancellationv1.RejectCancellationResponse {
	return &cancellationv1.RejectCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_FAILURE,
			Alerts: []*typesv1.Alert{{
				Message: errorMessage,
				Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
			}},
		},
	}
}

func withdrawCancellationErrorResponse(errorMessage string) *cancellationv1.WithdrawCancellationResponse {
	return &cancellationv1.WithdrawCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_FAILURE,
			Alerts: []*typesv1.Alert{{
				Message: errorMessage,
				Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
			}},
		},
	}
}

func finalizeCancellationErrorResponse(errorMessage string) *cancellationv1.FinalizeCancellationResponse {
	return &cancellationv1.FinalizeCancellationResponse{
		Header: &typesv1.ResponseHeader{
			BaseHeader: &typesv1.Header{Version: version.VersionV1},
			Status:     typesv1.StatusType_STATUS_TYPE_FAILURE,
			Alerts: []*typesv1.Alert{{
				Message: errorMessage,
				Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
			}},
		},
	}
}

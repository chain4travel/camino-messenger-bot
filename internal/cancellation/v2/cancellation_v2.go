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
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV4(ctx, request.RefundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMessage)
		return initiateCancellationErrResponse(errMessage), nil
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMessage := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMessage)
		return initiateCancellationErrResponse(errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.InitiateCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount, reasonValue, cancellationReasonVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error initiating cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return initiateCancellationErrResponse(errMessage), nil
	}

	response := &cancellationv2.InitiateCancellationResponse{
		Response: &cancellationv2.InitiateCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv2.InitiateCancellationSuccessResponse{
				Header: &typesv4.SuccessResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
				},
				TransactionId: &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()},
			},
		},
	}

	s.logger.Infof("Initiated cancellation for token %s with tx: %s", tokenID.String(), response.GetSuccessResponse().TransactionId.Hash)

	return response, nil
}

func (s *cancellationV2Service) CounterCancellation(
	ctx context.Context,
	request *cancellationv2.CounterCancellationRequest,
) (*cancellationv2.CounterCancellationResponse, error) {
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV4(ctx, request.RefundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMessage)
		return counterCancellationErrResponse(errMessage), nil
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMessage := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMessage)
		return counterCancellationErrResponse(errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.CounterCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount, reasonValue, counterReasonVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error countering cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return counterCancellationErrResponse(errMessage), nil
	}

	response := &cancellationv2.CounterCancellationResponse{
		Response: &cancellationv2.CounterCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv2.CounterCancellationSuccessResponse{
				Header: &typesv4.SuccessResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
				},
				TransactionId: &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()},
			},
		},
	}

	s.logger.Infof("Countered cancellation for token %s with tx: %s", tokenID.String(), response.GetSuccessResponse().TransactionId.Hash)

	return response, nil
}

func (s *cancellationV2Service) AcceptCancellation(
	ctx context.Context,
	request *cancellationv2.AcceptCancellationRequest,
) (*cancellationv2.AcceptCancellationResponse, error) {
	tokenID := new(big.Int).SetUint64(request.TokenId)
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV4(ctx, request.RefundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMessage)
		return acceptCancellationErrResponse(errMessage), nil
	}

	receipt, err := s.cmAccounts.AcceptCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error accepting cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return acceptCancellationErrResponse(errMessage), nil
	}

	response := &cancellationv2.AcceptCancellationResponse{
		Response: &cancellationv2.AcceptCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv2.AcceptCancellationSuccessResponse{
				Header: &typesv4.SuccessResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
				},
				TransactionId: &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()},
			},
		},
	}

	s.logger.Infof("Accepted cancellation for token %s with tx: %s", tokenID.String(), response.GetSuccessResponse().TransactionId.Hash)

	return response, nil
}

func (s *cancellationV2Service) RejectCancellation(
	ctx context.Context,
	request *cancellationv2.RejectCancellationRequest,
) (*cancellationv2.RejectCancellationResponse, error) {

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMessage := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMessage)
		return rejectCancellationErrResponse(errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.RejectCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, reasonValue, rejectReasonVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error rejecting cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return rejectCancellationErrResponse(errMessage), nil
	}

	response := &cancellationv2.RejectCancellationResponse{
		Response: &cancellationv2.RejectCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv2.RejectCancellationSuccessResponse{
				Header: &typesv4.SuccessResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
				},
				TransactionId: &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()},
			},
		},
	}

	s.logger.Infof("Rejected cancellation for token %s with tx: %s", tokenID.String(), response.GetSuccessResponse().TransactionId.Hash)

	return response, nil
}

func (s *cancellationV2Service) WithdrawCancellation(
	ctx context.Context,
	request *cancellationv2.WithdrawCancellationRequest,
) (*cancellationv2.WithdrawCancellationResponse, error) {
	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMessage := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMessage)
		return withdrawCancellationErrResponse(errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.WithdrawCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, reasonValue, withdrawReasonVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error withdrawing cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return withdrawCancellationErrResponse(errMessage), nil
	}

	response := &cancellationv2.WithdrawCancellationResponse{
		Response: &cancellationv2.WithdrawCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv2.WithdrawCancellationSuccessResponse{
				Header: &typesv4.SuccessResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
				},
				TransactionId: &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()},
			},
		},
	}

	s.logger.Infof("Withdrawn cancellation for token %s with tx: %s", tokenID.String(), response.GetSuccessResponse().TransactionId.Hash)

	return response, nil
}

func (s *cancellationV2Service) FinalizeCancellation(
	ctx context.Context,
	request *cancellationv2.FinalizeCancellationRequest,
) (*cancellationv2.FinalizeCancellationResponse, error) {
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV4(ctx, request.RefundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMessage)
		return finalizeCancellationErrResponse(errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.FinalizeCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error finalizing cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return finalizeCancellationErrResponse(errMessage), nil
	}

	response := &cancellationv2.FinalizeCancellationResponse{
		Response: &cancellationv2.FinalizeCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv2.FinalizeCancellationSuccessResponse{
				Header: &typesv4.SuccessResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
				},
				TransactionId: &typesv4.EVMTransactionID{Hash: receipt.TxHash.Hex()},
			},
		},
	}

	s.logger.Infof("Finalized cancellation for token %s with tx: %s", tokenID.String(), response.GetSuccessResponse().TransactionId.Hash)

	return response, nil
}

func initiateCancellationErrResponse(errorMessage string) *cancellationv2.InitiateCancellationResponse {
	return &cancellationv2.InitiateCancellationResponse{
		Response: &cancellationv2.InitiateCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv2.InitiateCancellationErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors:     []*typesv4.Error{{Message: errorMessage}},
				},
			},
		},
	}
}

func counterCancellationErrResponse(errorMessage string) *cancellationv2.CounterCancellationResponse {
	return &cancellationv2.CounterCancellationResponse{
		Response: &cancellationv2.CounterCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv2.CounterCancellationErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors:     []*typesv4.Error{{Message: errorMessage}},
				},
			},
		},
	}
}

func acceptCancellationErrResponse(errorMessage string) *cancellationv2.AcceptCancellationResponse {
	return &cancellationv2.AcceptCancellationResponse{
		Response: &cancellationv2.AcceptCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv2.AcceptCancellationErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors:     []*typesv4.Error{{Message: errorMessage}},
				},
			},
		},
	}
}

func rejectCancellationErrResponse(errorMessage string) *cancellationv2.RejectCancellationResponse {
	return &cancellationv2.RejectCancellationResponse{
		Response: &cancellationv2.RejectCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv2.RejectCancellationErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors:     []*typesv4.Error{{Message: errorMessage}},
				},
			},
		},
	}
}

func withdrawCancellationErrResponse(errorMessage string) *cancellationv2.WithdrawCancellationResponse {
	return &cancellationv2.WithdrawCancellationResponse{
		Response: &cancellationv2.WithdrawCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv2.WithdrawCancellationErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors:     []*typesv4.Error{{Message: errorMessage}},
				},
			},
		},
	}
}

func finalizeCancellationErrResponse(errorMessage string) *cancellationv2.FinalizeCancellationResponse {
	return &cancellationv2.FinalizeCancellationResponse{
		Response: &cancellationv2.FinalizeCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv2.FinalizeCancellationErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors:     []*typesv4.Error{{Message: errorMessage}},
				},
			},
		},
	}
}

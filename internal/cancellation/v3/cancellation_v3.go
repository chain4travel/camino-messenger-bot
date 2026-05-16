// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.
package cancellation

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/chain4travel/camino-messenger-bot/v13/internal/price"
	"github.com/chain4travel/camino-messenger-bot/v13/internal/version"
	cmaccounts "github.com/chain4travel/camino-messenger-bot/v13/pkg/cm_accounts"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/conversion"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v3/cancellationv3grpc"
	cancellationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"buf.build/go/protovalidate"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

const (
	cancellationReasonVersion = 1
	withdrawReasonVersion     = 1
	rejectReasonVersion       = 1
	counterReasonVersion      = 1
)

var _ cancellationv3grpc.CancellationServiceServer = (*cancellationV3Service)(nil)

func NewServiceV3(
	logger *zap.SugaredLogger,
	botKey *ecdsa.PrivateKey,
	cmAccountAddr ethCommon.Address,
	cmAccounts cmaccounts.Service,
	priceHandler price.Handler,
) cancellationv3grpc.CancellationServiceServer {
	return &cancellationV3Service{
		botKey:        botKey,
		cmAccountAddr: cmAccountAddr,
		logger:        logger,
		priceHandler:  priceHandler,
		cmAccounts:    cmAccounts,
	}
}

type cancellationV3Service struct {
	botKey        *ecdsa.PrivateKey
	cmAccountAddr ethCommon.Address
	logger        *zap.SugaredLogger
	priceHandler  price.Handler
	cmAccounts    cmaccounts.Service
}

func (s *cancellationV3Service) InitiateCancellation(
	ctx context.Context,
	request *cancellationv3.InitiateCancellationRequest,
) (*cancellationv3.InitiateCancellationResponse, error) {
	if err := protovalidate.Validate(request); err != nil {
		return initiateCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INVALID_PROTO, fmt.Sprintf("request validation failed: %v", err)), nil
	}

	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV5(ctx, request.RefundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMessage)
		return initiateCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMessage := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMessage)
		return initiateCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.InitiateCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount, reasonValue, cancellationReasonVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error initiating cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return initiateCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMessage), nil
	}

	response := &cancellationv3.InitiateCancellationResponse{
		Response: &cancellationv3.InitiateCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv3.InitiateCancellationSuccessResponse{
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

func (s *cancellationV3Service) CounterCancellation(
	ctx context.Context,
	request *cancellationv3.CounterCancellationRequest,
) (*cancellationv3.CounterCancellationResponse, error) {
	if err := protovalidate.Validate(request); err != nil {
		return counterCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INVALID_PROTO, fmt.Sprintf("request validation failed: %v", err)), nil
	}

	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV5(ctx, request.RefundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMessage)
		return counterCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMessage := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMessage)
		return counterCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.CounterCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount, reasonValue, counterReasonVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error countering cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return counterCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMessage), nil
	}

	response := &cancellationv3.CounterCancellationResponse{
		Response: &cancellationv3.CounterCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv3.CounterCancellationSuccessResponse{
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

func (s *cancellationV3Service) AcceptCancellation(
	ctx context.Context,
	request *cancellationv3.AcceptCancellationRequest,
) (*cancellationv3.AcceptCancellationResponse, error) {
	if err := protovalidate.Validate(request); err != nil {
		return acceptCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INVALID_PROTO, fmt.Sprintf("request validation failed: %v", err)), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV5(ctx, request.RefundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMessage)
		return acceptCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	receipt, err := s.cmAccounts.AcceptCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error accepting cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return acceptCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMessage), nil
	}

	response := &cancellationv3.AcceptCancellationResponse{
		Response: &cancellationv3.AcceptCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv3.AcceptCancellationSuccessResponse{
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

func (s *cancellationV3Service) RejectCancellation(
	ctx context.Context,
	request *cancellationv3.RejectCancellationRequest,
) (*cancellationv3.RejectCancellationResponse, error) {
	if err := protovalidate.Validate(request); err != nil {
		return rejectCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INVALID_PROTO, fmt.Sprintf("request validation failed: %v", err)), nil
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMessage := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMessage)
		return rejectCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.RejectCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, reasonValue, rejectReasonVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error rejecting cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return rejectCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMessage), nil
	}

	response := &cancellationv3.RejectCancellationResponse{
		Response: &cancellationv3.RejectCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv3.RejectCancellationSuccessResponse{
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

func (s *cancellationV3Service) WithdrawCancellation(
	ctx context.Context,
	request *cancellationv3.WithdrawCancellationRequest,
) (*cancellationv3.WithdrawCancellationResponse, error) {
	if err := protovalidate.Validate(request); err != nil {
		return withdrawCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INVALID_PROTO, fmt.Sprintf("request validation failed: %v", err)), nil
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMessage := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMessage)
		return withdrawCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.WithdrawCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, reasonValue, withdrawReasonVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error withdrawing cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return withdrawCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMessage), nil
	}

	response := &cancellationv3.WithdrawCancellationResponse{
		Response: &cancellationv3.WithdrawCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv3.WithdrawCancellationSuccessResponse{
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

func (s *cancellationV3Service) FinalizeCancellation(
	ctx context.Context,
	request *cancellationv3.FinalizeCancellationRequest,
) (*cancellationv3.FinalizeCancellationResponse, error) {
	if err := protovalidate.Validate(request); err != nil {
		return finalizeCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INVALID_PROTO, fmt.Sprintf("request validation failed: %v", err)), nil
	}

	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV5(ctx, request.RefundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMessage)
		return finalizeCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.FinalizeCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error finalizing cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return finalizeCancellationErrResponseV3(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMessage), nil
	}

	response := &cancellationv3.FinalizeCancellationResponse{
		Response: &cancellationv3.FinalizeCancellationResponse_SuccessResponse{
			SuccessResponse: &cancellationv3.FinalizeCancellationSuccessResponse{
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

func initiateCancellationErrResponseV3(code typesv4.ErrorCode, errorMessage string) *cancellationv3.InitiateCancellationResponse {
	return &cancellationv3.InitiateCancellationResponse{
		Response: &cancellationv3.InitiateCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv3.InitiateCancellationErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors: []*typesv4.Error{{
						Code:    code,
						Message: errorMessage,
					}},
				},
			},
		},
	}
}

func counterCancellationErrResponseV3(code typesv4.ErrorCode, errorMessage string) *cancellationv3.CounterCancellationResponse {
	return &cancellationv3.CounterCancellationResponse{
		Response: &cancellationv3.CounterCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv3.CounterCancellationErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors: []*typesv4.Error{{
						Code:    code,
						Message: errorMessage,
					}},
				},
			},
		},
	}
}

func acceptCancellationErrResponseV3(code typesv4.ErrorCode, errorMessage string) *cancellationv3.AcceptCancellationResponse {
	return &cancellationv3.AcceptCancellationResponse{
		Response: &cancellationv3.AcceptCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv3.AcceptCancellationErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors: []*typesv4.Error{{
						Code:    code,
						Message: errorMessage,
					}},
				},
			},
		},
	}
}

func rejectCancellationErrResponseV3(code typesv4.ErrorCode, errorMessage string) *cancellationv3.RejectCancellationResponse {
	return &cancellationv3.RejectCancellationResponse{
		Response: &cancellationv3.RejectCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv3.RejectCancellationErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors: []*typesv4.Error{{
						Code:    code,
						Message: errorMessage,
					}},
				},
			},
		},
	}
}

func withdrawCancellationErrResponseV3(code typesv4.ErrorCode, errorMessage string) *cancellationv3.WithdrawCancellationResponse {
	return &cancellationv3.WithdrawCancellationResponse{
		Response: &cancellationv3.WithdrawCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv3.WithdrawCancellationErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors: []*typesv4.Error{{
						Code:    code,
						Message: errorMessage,
					}},
				},
			},
		},
	}
}

func finalizeCancellationErrResponseV3(code typesv4.ErrorCode, errorMessage string) *cancellationv3.FinalizeCancellationResponse {
	return &cancellationv3.FinalizeCancellationResponse{
		Response: &cancellationv3.FinalizeCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv3.FinalizeCancellationErrorResponse{
				Header: &typesv4.ErrorResponseHeader{
					BaseHeader: &typesv4.Header{Version: version.VersionV4},
					Errors: []*typesv4.Error{{
						Code:    code,
						Message: errorMessage,
					}},
				},
			},
		},
	}
}

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

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v2/cancellationv2grpc"
	cancellationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v2"
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

var _ cancellationv2grpc.CancellationServiceServer = (*cancellationV2Service)(nil)

func NewService(
	logger *zap.SugaredLogger,
	botKey *ecdsa.PrivateKey,
	cmAccountAddr ethCommon.Address,
	cmAccounts cmaccounts.Service,
	priceHandler price.Handler,
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
	priceHandler  price.Handler
	cmAccounts    cmaccounts.Service
}

func (s *cancellationV2Service) InitiateCancellation(
	ctx context.Context,
	request *cancellationv2.InitiateCancellationRequest,
) (*cancellationv2.InitiateCancellationResponse, error) {
	if err := protovalidate.Validate(request); err != nil {
		return initiateCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INVALID_PROTO, fmt.Sprintf("request validation failed: %v", err)), nil
	}

	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV4(ctx, request.RefundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMessage)
		return initiateCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMessage := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMessage)
		return initiateCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.InitiateCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount, reasonValue, cancellationReasonVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error initiating cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return initiateCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMessage), nil
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
	if err := protovalidate.Validate(request); err != nil {
		return counterCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INVALID_PROTO, fmt.Sprintf("request validation failed: %v", err)), nil
	}

	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV4(ctx, request.RefundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMessage)
		return counterCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMessage := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMessage)
		return counterCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.CounterCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount, reasonValue, counterReasonVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error countering cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return counterCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMessage), nil
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
	if err := protovalidate.Validate(request); err != nil {
		return acceptCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INVALID_PROTO, fmt.Sprintf("request validation failed: %v", err)), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)
	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV4(ctx, request.RefundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMessage)
		return acceptCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	receipt, err := s.cmAccounts.AcceptCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error accepting cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return acceptCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMessage), nil
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
	if err := protovalidate.Validate(request); err != nil {
		return rejectCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INVALID_PROTO, fmt.Sprintf("request validation failed: %v", err)), nil
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMessage := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMessage)
		return rejectCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.RejectCancellationProposal(ctx, s.botKey, s.cmAccountAddr, tokenID, reasonValue, rejectReasonVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error rejecting cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return rejectCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMessage), nil
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
	if err := protovalidate.Validate(request); err != nil {
		return withdrawCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INVALID_PROTO, fmt.Sprintf("request validation failed: %v", err)), nil
	}

	reasonValue, err := conversion.ProtoEnumNumberToUInt16(request.Reason.Number())
	if err != nil {
		errMessage := fmt.Sprintf("error converting reason to uint16: %v", err)
		s.logger.Error(errMessage)
		return withdrawCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.WithdrawCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, reasonValue, withdrawReasonVersion)
	if err != nil {
		errMessage := fmt.Sprintf("error withdrawing cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return withdrawCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMessage), nil
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
	if err := protovalidate.Validate(request); err != nil {
		return finalizeCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INVALID_PROTO, fmt.Sprintf("request validation failed: %v", err)), nil
	}

	refundAmount, _, _, err := s.priceHandler.GetPriceAndTokenV4(ctx, request.RefundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error getting price and token: %v", err)
		s.logger.Error(errMessage)
		return finalizeCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_INTERNAL, errMessage), nil
	}

	tokenID := new(big.Int).SetUint64(request.TokenId)

	receipt, err := s.cmAccounts.FinalizeCancellation(ctx, s.botKey, s.cmAccountAddr, tokenID, refundAmount)
	if err != nil {
		errMessage := fmt.Sprintf("error finalizing cancellation proposal: %v", err)
		s.logger.Error(errMessage)
		return finalizeCancellationErrResponse(typesv4.ErrorCode_ERROR_CODE_BLOCKCHAIN_ERROR, errMessage), nil
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

func initiateCancellationErrResponse(code typesv4.ErrorCode, errorMessage string) *cancellationv2.InitiateCancellationResponse {
	return &cancellationv2.InitiateCancellationResponse{
		Response: &cancellationv2.InitiateCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv2.InitiateCancellationErrorResponse{
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

func counterCancellationErrResponse(code typesv4.ErrorCode, errorMessage string) *cancellationv2.CounterCancellationResponse {
	return &cancellationv2.CounterCancellationResponse{
		Response: &cancellationv2.CounterCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv2.CounterCancellationErrorResponse{
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

func acceptCancellationErrResponse(code typesv4.ErrorCode, errorMessage string) *cancellationv2.AcceptCancellationResponse {
	return &cancellationv2.AcceptCancellationResponse{
		Response: &cancellationv2.AcceptCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv2.AcceptCancellationErrorResponse{
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

func rejectCancellationErrResponse(code typesv4.ErrorCode, errorMessage string) *cancellationv2.RejectCancellationResponse {
	return &cancellationv2.RejectCancellationResponse{
		Response: &cancellationv2.RejectCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv2.RejectCancellationErrorResponse{
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

func withdrawCancellationErrResponse(code typesv4.ErrorCode, errorMessage string) *cancellationv2.WithdrawCancellationResponse {
	return &cancellationv2.WithdrawCancellationResponse{
		Response: &cancellationv2.WithdrawCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv2.WithdrawCancellationErrorResponse{
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

func finalizeCancellationErrResponse(code typesv4.ErrorCode, errorMessage string) *cancellationv2.FinalizeCancellationResponse {
	return &cancellationv2.FinalizeCancellationResponse{
		Response: &cancellationv2.FinalizeCancellationResponse_ErrorResponse{
			ErrorResponse: &cancellationv2.FinalizeCancellationErrorResponse{
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

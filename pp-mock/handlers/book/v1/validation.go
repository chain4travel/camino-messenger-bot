// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v1

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"

	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/google/uuid"
)

var _ bookv1grpc.ValidationServiceServer = (*validationServiceV1Server)(nil)

type validationServiceV1Server struct{}

func NewValidationServiceServer() bookv1grpc.ValidationServiceServer {
	return &validationServiceV1Server{}
}

func (s *validationServiceV1Server) Validation(_ context.Context, req *bookv1.ValidationRequest) (*bookv1.ValidationResponse, error) {
	if req.ValidationObject == nil ||
		req.ValidationObject.SearchIdentifier == nil ||
		req.ValidationObject.SearchIdentifier.ResultId == 0 ||
		req.ValidationObject.SearchIdentifier.SearchId == nil {
		response := bookv1.ValidationResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Invalid validation request: missing validation object or search identifier",
					Type:    typesv1.AlertType_ALERT_TYPE_INFO,
				}},
			},
		}
		return &response, nil
	}

	response := bookv1.ValidationResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		ValidationId:     &typesv1.UUID{Value: uuid.New().String()},
		ValidationObject: req.ValidationObject,
		PriceDetail: &typesv1.PriceDetail{
			Price: &typesv1.Price{
				Value:    common.DefaultPricePerNightStr,
				Decimals: common.DefaultPricePerNightDecimals,

				Currency: &typesv1.Currency{
					Currency: &typesv1.Currency_NativeToken{},
				},
			},
			Description: "price per night",
		},
	}

	return &response, nil
}

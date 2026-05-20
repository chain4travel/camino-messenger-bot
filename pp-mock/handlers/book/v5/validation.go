// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v5

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v5/bookv5grpc"
	bookv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v5"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	typesv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v5"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/price"
	"github.com/chain4travel/camino-messenger-bot/v13/pp-mock/common"
)

var _ bookv5grpc.ValidationServiceServer = (*validationV5Server)(nil)

type validationV5Server struct{}

func NewValidationServiceServer() bookv5grpc.ValidationServiceServer {
	return &validationV5Server{}
}

func (s *validationV5Server) Validation(_ context.Context, req *bookv5.ValidationRequest) (*bookv5.ValidationResponse, error) {
	return &bookv5.ValidationResponse{
		Response: &bookv5.ValidationResponse_SuccessResponse{
			SuccessResponse: &bookv5.ValidationSuccessResponse{
				Header:           common.SuccessHeaderV4(),
				ValidationId:     common.NewExpiringUUID(),
				ValidationObject: req.ValidationObject,
				TotalPrice: &typesv5.TotalPrice{
					Value: &typesv5.Price{
						Value:    "100000000000000000000",
						Decimals: uint32(price.NativeTokenDecimals),
						Currency: &typesv4.Currency{
							Currency: &typesv4.Currency_NativeToken{},
						},
					},
				},
			},
		},
	}, nil
}

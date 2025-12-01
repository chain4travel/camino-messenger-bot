// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v3

import (
	"context"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v3/infov3grpc"
	infov3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/info/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
	"github.com/google/uuid"
)

var _ infov3grpc.CountryEntryRequirementsServiceServer = (*countryEntryRequirementsServiceV3Server)(nil)

type countryEntryRequirementsServiceV3Server struct{}

func NewCountryEntryRequirementsServiceServer() infov3grpc.CountryEntryRequirementsServiceServer {
	return &countryEntryRequirementsServiceV3Server{}
}

func (s *countryEntryRequirementsServiceV3Server) CountryEntryRequirements(_ context.Context, _ *infov3.CountryEntryRequirementsRequest) (*infov3.CountryEntryRequirementsResponse, error) {
	return &infov3.CountryEntryRequirementsResponse{
		Response: &infov3.CountryEntryRequirementsResponse_SuccessResponse{
			SuccessResponse: &infov3.CountryEntryRequirementsSuccessResponse{
				Header:     common.SuccessHeaderV4(),
				ResponseId: &typesv4.UUID{Value: uuid.NewString()},
			},
		},
	}, nil
}

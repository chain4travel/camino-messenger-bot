// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package server

import (
	"context"

	"github.com/chain4travel/camino-messenger-bot/v12/proto/pb/readiness"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const StatusReady = "ready"

func (s *server) Readiness(context.Context, *emptypb.Empty) (*readiness.ReadinessResponse, error) {
	return &readiness.ReadinessResponse{Status: StatusReady}, nil
}

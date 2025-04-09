// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package partnerplugin

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/chain4travel/camino-messenger-bot/pp-mock/proto/pb/events"
	"github.com/chain4travel/camino-messenger-bot/tests/e2e/process"
)

const requestTickerInterval = 500 * time.Millisecond

type PartnerPlugin struct {
	logger       *zap.SugaredLogger
	pid          int
	host         *url.URL
	pingClient   pingv1grpc.PingServiceClient
	eventsClient events.MyEventsServiceClient
	logFile      *os.File
}

func (pp *PartnerPlugin) Host() string {
	if pp == nil {
		return ""
	}
	return pp.host.Host
}

func (pp *PartnerPlugin) Stop(ctx context.Context) error {
	if pp == nil {
		return nil
	}
	if err := process.StopProcess(ctx, pp.pid); err != nil {
		return fmt.Errorf("failed to stop partner plugin process with pid %d: %w", pp.pid, err)
	}
	pp.logger.Debugf("Partner plugin (pid %d) stopped", pp.pid)
	if err := pp.logFile.Close(); err != nil {
		return fmt.Errorf("failed to close partner plugin logfile: %w", err)
	}
	return nil
}

func (pp *PartnerPlugin) SubscribeForEvents(ctx context.Context) (events.MyEventsService_SubscribeClient, error) {
	return pp.eventsClient.Subscribe(ctx, &emptypb.Empty{})
}

func (pp *PartnerPlugin) awaitReady(ctx context.Context) error {
	ticker := time.NewTicker(requestTickerInterval)
	defer ticker.Stop()

	for {
		res, err := pp.pingClient.Ping(ctx, &pingv1.PingRequest{
			Header: &typesv1.RequestHeader{
				BaseHeader: &typesv1.Header{
					Version:              &typesv1.Version{},
					EndUserWalletAddress: common.Address{}.Hex(),
				},
			},
			PingMessage: "ping",
			Timestamp:   timestamppb.Now(),
		})
		if err == nil && res.Header.Status == typesv1.StatusType_STATUS_TYPE_SUCCESS {
			return nil
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

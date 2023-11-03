package client

import (
	"fmt"
	"sync"

	"camino-messenger-bot/config"
	"camino-messenger-bot/internal/proto/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type RPCClient struct {
	Gsc    pb.GreetingServiceClient
	cfg    *config.PartnerPluginConfig
	logger *zap.SugaredLogger
	cc     *grpc.ClientConn
	mu     sync.Mutex
}

func NewClient(cfg *config.PartnerPluginConfig, logger *zap.SugaredLogger) *RPCClient {
	return &RPCClient{
		cfg:    cfg,
		logger: logger,
	}
}
func (rc *RPCClient) Start() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	opts := grpc.WithInsecure() //todo add TLS
	cc, err := grpc.Dial(fmt.Sprintf("%s:%d", rc.cfg.PartnerPluginHost, rc.cfg.PartnerPluginPort), opts)
	if err != nil {
		return nil
	}
	rc.Gsc = pb.NewGreetingServiceClient(cc)
	rc.cc = cc
	return nil
}

func (rc *RPCClient) Shutdown() error {
	rc.logger.Info("Shutting down gRPC client...")
	return rc.cc.Close()
}

package client

import (
	"fmt"
	"sync"

	"google.golang.org/grpc/credentials/insecure"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	utils "github.com/chain4travel/camino-messenger-bot/utils/tls"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var _ metadata.Checkpoint = (*RPCClient)(nil)

type RPCClient struct {
	cfg        *config.PartnerPluginConfig
	logger     *zap.SugaredLogger
	ClientConn *grpc.ClientConn
	mu         sync.Mutex
}

func NewClient(cfg *config.PartnerPluginConfig, logger *zap.SugaredLogger) *RPCClient {
	return &RPCClient{
		cfg:    cfg,
		logger: logger,
	}
}

func (rc *RPCClient) Checkpoint() string {
	return "ext-system-client"
}

func (rc *RPCClient) Start() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	var opts []grpc.DialOption
	if rc.cfg.Unencrypted {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		tlsCreds, err := utils.LoadCATLSCredentials(rc.cfg.CACertFile)
		if err != nil {
			return fmt.Errorf("could not load TLS keys: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(tlsCreds))
	}

	cc, err := grpc.NewClient(fmt.Sprintf("%s:%d", rc.cfg.Host, rc.cfg.Port), opts...)
	if err != nil {
		return nil
	}
	rc.ClientConn = cc
	return nil
}

func (rc *RPCClient) Shutdown() error {
	rc.logger.Info("Shutting down gRPC client...")
	return rc.ClientConn.Close()
}

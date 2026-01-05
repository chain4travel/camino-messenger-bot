// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package server

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/chain4travel/camino-messenger-bot/v12/config"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/messaging"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/messaging/message"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/rpc"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/rpc/generated"
	"github.com/chain4travel/camino-messenger-bot/v12/internal/utils/tls"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v12/proto/pb/readiness"
	"github.com/google/uuid"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v1/cancellationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/cancellation/v2/cancellationv2grpc"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	grpcMetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	_ Server                           = (*server)(nil)
	_ rpc.RequestHandler               = (*server)(nil)
	_ readiness.ReadinessServiceServer = (*server)(nil)
)

type Server interface {
	Start(ctx context.Context) (chan error, error)
	Stop()
}

func NewServer(
	cfg config.RPCServerConfig,
	logger *zap.SugaredLogger,
	processor messaging.MessageProcessor,
	serviceRegistry messaging.ServiceRegistry,
	cancellationV1Service cancellationv1grpc.CancellationServiceServer,
	cancellationV2Service cancellationv2grpc.CancellationServiceServer,
	developerMode bool,
) (Server, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	var opts []grpc.ServerOption
	if cfg.Unencrypted {
		logger.Warn("Running gRPC server without TLS!")
	} else {
		creds, err := tls.LoadTLSCredentials(cfg.ServerCertFile, cfg.ServerKeyFile)
		if err != nil {
			return nil, fmt.Errorf("could not load TLS keys: %w", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}

	s := &server{
		cfg:             cfg,
		logger:          logger,
		processor:       processor,
		serviceRegistry: serviceRegistry,
	}

	opts = append(opts, grpc.ChainUnaryInterceptor(
		s.unaryRecoverInterceptor,
		selector.UnaryServerInterceptor( // for all cancellation v1/v2 methods
			s.tracingInterceptor,
			selector.MatchFunc(func(_ context.Context, callMeta interceptors.CallMeta) bool {
				return cancellationv1grpc.CancellationService_ServiceDesc.ServiceName == callMeta.Service ||
					cancellationv2grpc.CancellationService_ServiceDesc.ServiceName == callMeta.Service
			}),
		),
	))

	s.grpcServer = grpc.NewServer(opts...)
	generated.RegisterServerServices(s.grpcServer, s)
	cancellationv1grpc.RegisterCancellationServiceServer(s.grpcServer, cancellationV1Service)
	cancellationv2grpc.RegisterCancellationServiceServer(s.grpcServer, cancellationV2Service)
	readiness.RegisterReadinessServiceServer(s.grpcServer, s)

	// Register reflection service on gRPC server in developerMode.
	if developerMode {
		reflection.Register(s.grpcServer)
	}
	return s, nil
}

type server struct {
	grpcServer      *grpc.Server
	cfg             config.RPCServerConfig
	logger          *zap.SugaredLogger
	processor       messaging.MessageProcessor
	serviceRegistry messaging.ServiceRegistry

	readiness.UnimplementedReadinessServiceServer
}

func (s *server) Start(ctx context.Context) (chan error, error) {
	listenCfg := net.ListenConfig{}
	lis, err := listenCfg.Listen(ctx, "tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	errChan := make(chan error)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("gRPC server panicked: %v", r)
				s.logger.Errorf("recovered from panic: %v", err)
				errChan <- err
			}
			close(errChan)
		}()

		s.logger.Infof("gRPC server listening on %s", lis.Addr().String())

		if err := s.grpcServer.Serve(lis); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, grpc.ErrServerStopped) {
			s.logger.Errorf("gRPC server stopped serving with error: %w", err)
			errChan <- err
		}
	}()

	return errChan, nil
}

func (s *server) Stop() {
	s.grpcServer.Stop()
}

func (s *server) HandleMessageRequest(ctx context.Context, requestType message.Type, request protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
	recipientCMAccountAddress, err := s.getRecipientAddress(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipient cm account address from request context: %w", err)
	}

	requestMsg := &message.Message{
		Type:       requestType,
		Content:    request,
		RequestID:  uuid.New().String(),
		Timestamps: metadata.Timestamps{},
	}

	requestMsg.Timestamps.Stamp(metadata.CheckpointGRPCRequestReceived)

	responseMsg, err := s.processor.SendRequestMessage(ctx, requestMsg, recipientCMAccountAddress)
	if err != nil {
		return nil, fmt.Errorf("error sending request message: %w", err)
	}

	responseMsg.Timestamps.Stamp(metadata.CheckpointGRPCResponseSent)

	timestampsStr, err := responseMsg.Timestamps.MarshalToString()
	if err != nil {
		return nil, fmt.Errorf("error marshalling timestamps: %w", err)
	}

	return responseMsg.Content, grpc.SendHeader(ctx, grpcMetadata.Pairs(
		metadata.KeyRequestID, responseMsg.RequestID,
		metadata.KeyTimestamps, timestampsStr,
	))
}

func (s *server) getRecipientAddress(ctx context.Context) (ethCommon.Address, error) {
	mdPairs, ok := grpcMetadata.FromIncomingContext(ctx)
	if !ok {
		return ethCommon.Address{}, fmt.Errorf("metadata not found in incoming context")
	}

	recipient := mdPairs[metadata.KeyRecipientCMAccount]
	if len(recipient) != 1 || !ethCommon.IsHexAddress(recipient[0]) {
		return ethCommon.Address{}, fmt.Errorf("invalid recipient address: %s", recipient)
	}

	return ethCommon.HexToAddress(recipient[0]), nil
}

func (s *server) unaryRecoverInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response any, err error) {
	defer func() {
		if r := recover(); r != nil {
			var recipientCMAccountAddress ethCommon.Address
			recipientCMAccountAddress, err = s.getRecipientAddress(ctx)
			if err != nil {
				s.logger.Errorf("failed to get recipient cm account address from request context: %v", err)
			}
			err = fmt.Errorf("gRPC %s (recipient %s) handler panicked: %v", info.FullMethod, recipientCMAccountAddress.Hex(), r) // we return this error to the client
			s.logger.Errorf("recovered from panic: %v", err)
		}
	}()

	return handler(ctx, req)
}

// Not intended to be used with p2p services, as they are managing headers themselves.
func (s *server) tracingInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	timestamps := metadata.Timestamps{}
	timestamps.Stamp(metadata.CheckpointGRPCRequestReceived)

	defer func() {
		timestamps.Stamp(metadata.CheckpointGRPCResponseSent)

		timestampsStr, err := timestamps.MarshalToString()
		if err != nil {
			s.logger.Errorf("error marshalling timestamps: %v", err)
		}

		if err := grpc.SendHeader(ctx, grpcMetadata.Pairs(
			metadata.KeyTimestamps, timestampsStr,
		)); err != nil {
			s.logger.Errorf("failed to send header: %v", err)
		}
	}()

	return handler(ctx, req)
}

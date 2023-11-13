package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chain4travel/camino-messenger-bot/proto/pb/messages"
	utils "github.com/chain4travel/camino-messenger-bot/utils/tls"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
)

func main() {
	var logger *zap.Logger
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, _ = cfg.Build()
	sLogger := logger.Sugar()
	logger.Sync()

	var opts []grpc.DialOption
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) == 1 {
		tlsCreds, err := utils.LoadCATLSCredentials(argsWithoutProg[0])
		if err != nil {
			log.Fatalf("could not load TLS keys: %s", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(tlsCreds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	c := client.NewClient(&config.PartnerPluginConfig{
		PartnerPluginHost: "localhost",
		PartnerPluginPort: 9090,
	}, sLogger, opts)
	request := &messages.FlightSearchRequest{
		Market:   "MUC",
		Currency: "EUR",
	}

	err := c.Start()
	if err != nil {
		panic(err)
	}
	// ralf sending request to OAG
	md := metadata.New(map[string]string{
		"sender":    "@nikostest1:localhost",
		"recipient": "@nikostest2:localhost",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	begin := time.Now()
	resp, err := c.Sc.Search(ctx, request)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Received response after %s => %s\n", time.Since(begin), resp.Context)

	c.Shutdown()

}

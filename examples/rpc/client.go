package main

import (
	"context"
	"fmt"
	"log"

	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"camino-messenger-bot/config"
	"camino-messenger-bot/internal/proto/pb"
	"camino-messenger-bot/internal/rpc/client"
)

func main() {

	// create logger
	var logger *zap.Logger
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, _ = cfg.Build()
	sLogger := logger.Sugar()
	logger.Sync()

	c := client.NewClient(&config.PartnerPluginConfig{
		PartnerPluginHost: "localhost",
		PartnerPluginPort: 9090,
	}, sLogger)
	request := &pb.GreetingServiceRequest{Name: "Gophers"}

	err := c.Start()
	if err != nil {
		panic(err)
	}
	// ralf sending request to OAG
	md := metadata.New(map[string]string{
		"sender":  "@t-kopernikus15r59v8u6uc4d5cgzscyeskkq8ydndukp5jucg2:matrix.chain4travel.com",
		"room_id": "!hlEisVaujNqfJEBuzR:matrix.chain4travel.com",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := c.Gsc.Greeting(ctx, request)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("StartReceiver response => %s ", resp.Message)

	c.Shutdown()

}

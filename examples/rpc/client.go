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

	c := client.NewClient(&config.PartnerPluginConfig{
		PartnerPluginHost: "localhost",
		PartnerPluginPort: 9090,
	}, &zap.SugaredLogger{})
	request := &pb.GreetingServiceRequest{Name: "Gophers"}

	err := c.Start()
	if err != nil {
		panic(err)
	}
	// ralf sending request to a3m
	md := metadata.New(map[string]string{
		"sender":  "0x028f455c1f95e1ec24bfafb81cb2d1e76118944931e3a2599241a457d7d5b8399b2ba5bb39",
		"room_id": "!VZwjBnxEukQOzOjoYt:matrix.chain4travel.com",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := c.Gsc.Greeting(ctx, request)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("StartReceiver response => %s ", resp.Message)

	c.Shutdown()

}

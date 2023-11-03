package main

import (
	"context"
	"fmt"
	"log"

	"go.uber.org/zap"

	"camino-messenger-bot/config"
	"camino-messenger-bot/internal/proto/pb"
	"camino-messenger-bot/internal/rpc/client"
)

func main() {

	c := client.NewClient(&config.PartnerPluginConfig{
		PartnerPluginHost: "localhost",
		PartnerPluginPort: 50051,
	}, &zap.SugaredLogger{})
	request := &pb.GreetingServiceRequest{Name: "Gophers"}

	resp, err := c.Gsc.Greeting(context.Background(), request)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("StartReceiver response => %s ", resp.Message)
}

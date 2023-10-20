package client

import (
	"camino-messenger-provider/config"
	"camino-messenger-provider/internal/proto/pb"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
)

type RPCClient struct {
	Gsc pb.GreetingServiceClient
	cfg *config.PartnerPluginConfig
	cc  *grpc.ClientConn
}

func NewClient(cfg *config.PartnerPluginConfig) RPCClient {
	opts := grpc.WithInsecure()
	cc, err := grpc.Dial(fmt.Sprintf("%s:%d", cfg.PartnerPluginHost, cfg.PartnerPluginPort), opts)
	if err != nil {
		log.Fatal(err)
	}

	return RPCClient{
		Gsc: pb.NewGreetingServiceClient(cc),
		cc:  cc,
		cfg: cfg,
	}
}

func (r *RPCClient) Shutdown() error {
	return r.cc.Close()
}

func main() {

	c := NewClient(&config.PartnerPluginConfig{
		PartnerPluginHost: "localhost",
		PartnerPluginPort: 50051,
	})
	request := &pb.GreetingServiceRequest{Name: "Gophers"}

	resp, err := c.Gsc.Greeting(context.Background(), request)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Receive response => %s ", resp.Message)
}

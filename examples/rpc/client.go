package main

import (
	typesv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1alpha"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha/accommodationv1alphagrpc"
	accommodationv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha"
	internalmetadata "github.com/chain4travel/camino-messenger-bot/internal/metadata"
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

	argsWithoutProg := os.Args[1:]
	unencrypted := len(argsWithoutProg) == 0
	ppConfig := config.PartnerPluginConfig{
		Host:        "localhost",
		Port:        9092,
		Unencrypted: unencrypted,
	}
	if !unencrypted {
		ppConfig.CACertFile = argsWithoutProg[0]
	}
	c := client.NewClient(&ppConfig, sLogger)
	err := c.Start()
	if err != nil {
		panic(err)
	}
	request := &accommodationv1alpha.AccommodationSearchRequest{
		Header: nil,
		SearchParametersGeneric: &typesv1alpha.SearchParameters{
			Currency:   typesv1alpha.Currency_CURRENCY_EUR,
			Language:   typesv1alpha.Language_LANGUAGE_UG,
			Market:     1,
			MaxOptions: 2,
		},
		Queries: []*accommodationv1alpha.AccommodationSearchQuery{
			{
				SearchParametersAccommodation: &accommodationv1alpha.AccommodationSearchParameters{
					RatePlan: []*typesv1alpha.RatePlan{{RatePlan: "economy"}},
				},
			},
		},
	}

	err = c.Start()
	if err != nil {
		panic(err)
	}
	md := metadata.New(map[string]string{
		"recipient": "@t-kopernikus1tyewqsap6v8r8wghg7qn7dyfzg2prtcrw04ke3:matrix.camino.network",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	ass := accommodationv1alphagrpc.NewAccommodationSearchServiceClient(c.ClientConn)
	begin := time.Now()
	var header metadata.MD
	resp, err := ass.AccommodationSearch(ctx, request, grpc.Header(&header))
	if err != nil {
		log.Fatal(err)
	}
	metadata := &internalmetadata.Metadata{}
	err = metadata.FromGrpcMD(header)
	if err != nil {
		fmt.Print("error extracting metadata")
	}
	fmt.Printf("Received response after %s => ID: %s\n", time.Since(begin), resp.Metadata.SearchId)
	c.Shutdown()

}

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha1/accommodationv1alpha1grpc"
	accommodationv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha1"
	typesv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1alpha1"
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
	request := &accommodationv1alpha1.AccommodationSearchRequest{
		Header: nil,
		SearchParametersGeneric: &typesv1alpha1.SearchParameters{
			Currency:   typesv1alpha1.Currency_CURRENCY_EUR,
			Language:   typesv1alpha1.Language_LANGUAGE_UG,
			Market:     1,
			MaxOptions: 2,
		},
		SearchParametersAccommodation: &accommodationv1alpha1.AccommodationSearchParameters{
			RatePlan: []*typesv1alpha1.RatePlan{
				{
					RatePlan: "economy",
				},
			},
		},
	}

	err = c.Start()
	if err != nil {
		panic(err)
	}
	md := metadata.New(map[string]string{
		"recipient": "@t-kopernikus1dry573dcz6jefshfxgya68jd6s07xezpm27ng9:matrix.camino.network",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	ass := accommodationv1alpha1grpc.NewAccommodationSearchServiceClient(c.ClientConn)
	var header metadata.MD
	begin := time.Now()
	resp, err := ass.AccommodationSearch(ctx, request, grpc.Header(&header))
	if err != nil {
		log.Fatal(err)
	}
	totalTime := time.Since(begin)
	fmt.Printf("Total time|%s|%s\n", resp.Metadata.SearchId, totalTime)
	metadata := &internalmetadata.Metadata{}
	err = metadata.FromGrpcMD(header)
	if err != nil {
		fmt.Print("error extracting metadata")
	}

	var entries []struct {
		Key   string
		Value int64
	}
	// Populate the slice with map entries
	for key, value := range metadata.Timestamps {
		entries = append(entries, struct {
			Key   string
			Value int64
		}{Key: key, Value: value})
	}

	// Sort the slice based on values
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Value < entries[j].Value
	})
	lastValue := int64(0)
	for _, entry := range entries {
		fmt.Printf("%s|%s|%d|%d|%f\n", entry.Key, resp.Metadata.SearchId, entry.Value, entry.Value-lastValue, float32(entry.Value-lastValue)/float32(totalTime.Milliseconds()))
		lastValue = entry.Value
	}
	c.Shutdown()

}

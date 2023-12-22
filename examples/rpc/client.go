package main

import (
	typesv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1alpha1"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha1/accommodationv1alpha1grpc"
	accommodationv1alpha1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha1"
	internalmetadata "github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
)

func main() {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var logger *zap.Logger
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, _ = cfg.Build()
	sLogger := logger.Sugar()
	logger.Sync()

	times := flag.Int("requests", 1, "Repeat the request n times")
	host := flag.String("host", "127.0.0.1", "Distributor bot host")
	port := flag.Int("port", 9092, "Distributor bot port")
	recipient := flag.String("recipient", "@t-kopernikus1tyewqsap6v8r8wghg7qn7dyfzg2prtcrw04ke3:matrix.camino.network", "Recipient address (format: @t-kopernikus[...]:matrix.camino.network")
	caCertFile := flag.String("ca-cert-file", "", "CA certificate file (optional)")
	flag.Parse()

	ppConfig := config.PartnerPluginConfig{
		Host:        *host,
		Port:        *port,
		Unencrypted: *caCertFile == "",
	}
	ppConfig.CACertFile = *caCertFile

	loadTestData := make([][]string, *times)
	for i := 0; i < *times; i++ {
		loadTestData[i] = make([]string, 6)
		wg.Add(1)
		go func(counter int) {
			defer wg.Done()
			createClientAndRunRequest(counter, ppConfig, sLogger, *recipient, loadTestData, mu)
		}(i)
	}

	wg.Wait()

	if len(loadTestData) > 1 || len(loadTestData) == 1 && loadTestData[0][0] != "" { // otherwise no data have been recorded
		persistToCSV(loadTestData)
	}
}

func createClientAndRunRequest(i int, ppConfig config.PartnerPluginConfig, sLogger *zap.SugaredLogger, recipient string, loadTestData [][]string, mu sync.Mutex) {
	c := client.NewClient(&ppConfig, sLogger)
	err := c.Start()
	if err != nil {
		fmt.Errorf("error starting client: %v", err)
		return
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

	md := metadata.New(map[string]string{
		"recipient": recipient,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	ass := accommodationv1alpha1grpc.NewAccommodationSearchServiceClient(c.ClientConn)
	var header metadata.MD
	begin := time.Now()
	resp, err := ass.AccommodationSearch(ctx, request, grpc.Header(&header))
	if err != nil {
		sLogger.Errorf("error when performing search: %v", err)
		return
	}
	totalTime := time.Since(begin)
	fmt.Println(totalTime.Milliseconds())
	//fmt.Printf("Total time(ms)|%s|%d\n", resp.Metadata.SearchId.GetValue(), totalTime.Milliseconds())
	metadata := &internalmetadata.Metadata{}
	err = metadata.FromGrpcMD(header)
	if err != nil {
		sLogger.Errorf("error extracting metadata: %v", err)
	}

	addToDataset(int64(i), totalTime.Milliseconds(), resp, metadata, loadTestData, mu)

	c.Shutdown()
}

func addToDataset(counter int64, totalTime int64, resp *accommodationv1alpha1.AccommodationSearchResponse, metadata *internalmetadata.Metadata, loadTestData [][]string, mu sync.Mutex) {
	var data []string
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
	data = append(data, strconv.FormatInt(counter+1, 10))
	data = append(data, strconv.FormatInt(totalTime, 10))
	for _, entry := range entries {

		if entry.Key == "request-gateway-request" {
			lastValue = entry.Value
			continue //skip
		}
		if entry.Key == "processor-request" {

			//lastValue = entry.Value
			continue //skip
		}
		fmt.Printf("%d|%s|%s|%d|%.2f\n", entry.Value, entry.Key, resp.Metadata.SearchId.GetValue(), entry.Value-lastValue, float32(entry.Value-lastValue)/float32(totalTime))

		data = append(data, strconv.FormatInt(entry.Value-lastValue, 10))
		lastValue = entry.Value
	}

	mu.Lock()
	loadTestData[counter] = data
	mu.Unlock()
}
func persistToCSV(dataset [][]string) {
	// Open a new CSV file
	file, err := os.Create("load_test_data.csv")
	if err != nil {
		fmt.Println("Error creating CSV file:", err)
		return
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header row
	header := []string{"Request ID", "Total Time", "distributor -> matrix", "matrix -> provider", "provider -> matrix", "matrix -> distributor", "process-response"}
	if err := writer.Write(header); err != nil {
		fmt.Println("Error writing header:", err)
		return
	}

	// Write the load test data rows
	for _, dataRow := range dataset {
		if err := writer.Write(dataRow); err != nil {
			fmt.Println("Error writing data row:", err)
			return
		}
	}

	fmt.Println("CSV file created successfully.")
}

package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	internalmetadata "github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
)

func main() {
	mu := &sync.Mutex{}
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
	recipient := flag.String("recipient", "@0xeb3D6560a5eCf3e00b68a4b2899FEc93419F06B9:", "Recipient c-chain address (format: @[...]:messenger.chain4travel.com")
	caCertFile := flag.String("ca-cert-file", "", "CA certificate file (optional)")
	flag.Parse()
	hostURL, _ := url.Parse(fmt.Sprintf("%s:%d", *host, *port))

	ppConfig := config.PartnerPluginConfig{
		HostURL:     *hostURL,
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

func createClientAndRunRequest(
	i int,
	ppConfig config.PartnerPluginConfig,
	sLogger *zap.SugaredLogger,
	recipient string,
	loadTestData [][]string,
	mu *sync.Mutex,
) {
	c, err := client.NewClient(ppConfig, sLogger)
	if err != nil {
		fmt.Printf("error creating client: %v", err)
		return
	}
	request := &accommodationv1.AccommodationSearchRequest{
		Header: nil,
		SearchParametersGeneric: &typesv1.SearchParameters{
			Language:   typesv1.Language_LANGUAGE_UG,
			Market:     1,
			MaxOptions: 2,
		},
		Queries: []*accommodationv1.AccommodationSearchQuery{
			{
				SearchParametersAccommodation: &accommodationv1.AccommodationSearchParameters{
					SupplierCodes: []*typesv1.SupplierProductCode{{SupplierCode: "supplier1"}},
				},
			},
		},
	}

	md := metadata.New(map[string]string{
		"recipient": recipient,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	ass := accommodationv1grpc.NewAccommodationSearchServiceClient(c.ClientConn)
	begin := time.Now()
	var header metadata.MD
	resp, err := ass.AccommodationSearch(ctx, request, grpc.Header(&header))
	if err != nil {
		sLogger.Errorf("error when performing search: %v", err)
		return
	}
	totalTime := time.Since(begin)
	fmt.Printf("Total time|%s|%s\n", resp.Metadata.SearchId, totalTime)
	metadata := &internalmetadata.Metadata{}
	err = metadata.FromGrpcMD(header)
	if err != nil {
		sLogger.Errorf("error extracting metadata: %v", err)
	}

	addToDataset(int64(i), totalTime.Milliseconds(), resp, metadata, loadTestData, mu)
	fmt.Printf("Received response after %s => ID: %s\n", time.Since(begin), resp.Metadata.SearchId)
	c.Shutdown()
}

func addToDataset(
	counter int64,
	totalTime int64,
	resp *accommodationv1.AccommodationSearchResponse,
	metadata *internalmetadata.Metadata,
	loadTestData [][]string,
	mu *sync.Mutex,
) {
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
			continue // skip
		}
		if entry.Key == "processor-request" {
			// lastValue = entry.Value
			continue // skip
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

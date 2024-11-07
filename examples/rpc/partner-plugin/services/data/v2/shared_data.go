package helpers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
)

type ValidationData struct {
	ValidationObject *bookv2.ValidationObject `json:"validation_object"`
	PriceDetail      *typesv2.PriceDetail     `json:"price_detail"`
}

var (
	properties []accommodationv2.PropertyExtendedInfo
	loadOnce   sync.Once
)

func LoadPropertiesMockData() []accommodationv2.PropertyExtendedInfo {
	loadOnce.Do(func() {
		// Assuming the JSON file is relative to the current file
		currentDir, err := os.Getwd()
		if err != nil {
			log.Printf("Error getting current directory: %v", err)
			return
		}

		filePath := filepath.Join(currentDir, "../../examples", "rpc", "partner-plugin", "mock_data", "accommodation", "v2", "properties.json")
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Error reading properties file: %v", err)
			return
		}

		if err := json.Unmarshal(data, &properties); err != nil {
			log.Printf("Error unmarshaling properties: %v", err)
			return
		}

		log.Printf("Successfully loaded %d properties", len(properties))
	})

	return properties
}

func LoadValidationMockData() (map[string]*ValidationData, error) {
	var (
		err           error
		validationMap map[string]*ValidationData
	)

	loadOnce.Do(func() {
		// Get the current working directory
		currentDir, err := os.Getwd()
		if err != nil {
			log.Printf("Error getting current directory: %v", err)
			err = fmt.Errorf("failed to get current directory: %w", err)
			return
		}

		filePath := filepath.Join(currentDir, "../../examples", "rpc", "partner-plugin", "mock_data", "book", "validation.json")

		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Error reading validation mock data file: %v", err)
			err = fmt.Errorf("failed to read validation mock data file: %w", err)
			return
		}

		var mockResp []*ValidationData
		if err := json.Unmarshal(data, &mockResp); err != nil {
			log.Printf("Error unmarshaling validation mock data: %v", err)
			err = fmt.Errorf("failed to unmarshal validation mock data: %w", err)
			return
		}

		validationMap = make(map[string]*ValidationData)
		for _, v := range mockResp {
			key := v.ValidationObject.SearchIdentifier.SearchId.Value
			validationMap[key] = v
		}
	})

	return validationMap, err
}

package helpers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	validationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1" // Adjust the import path based on your actual proto package
)

var (
	properties         []accommodationv1.PropertyExtendedInfo
	loadOnce           sync.Once
	validationResponse *validationv1.ValidationResponse
)

func LoadPropertiesMockData() []accommodationv1.PropertyExtendedInfo {
	loadOnce.Do(func() {
		// Assuming the JSON file is relative to the current file
		currentDir, err := os.Getwd()
		if err != nil {
			log.Printf("Error getting current directory: %v", err)
			return
		}

		filePath := filepath.Join(currentDir, "../../examples", "rpc", "partner-plugin", "mock_data", "accommodation", "v1", "properties.json")
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

func LoadValidationMockData() (*validationv1.ValidationResponse, error) {
	var err error
	loadOnce.Do(func() {
		// Get the current working directory
		currentDir, err := os.Getwd()
		if err != nil {
			log.Printf("Error getting current directory: %v", err)
			err = fmt.Errorf("failed to get current directory: %w", err)
			return
		}

		filePath := filepath.Join(currentDir, "../../examples", "rpc", "partner-plugin", "mock_data", "book", "validation_response.json")

		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Error reading validation mock data file: %v", err)
			return
		}

		var mockResp validationv1.ValidationResponse
		if err := json.Unmarshal(data, &mockResp); err != nil {
			log.Printf("Error unmarshaling validation mock data: %v", err)
			err = fmt.Errorf("failed to unmarshal validation mock data: %w", err)
			return
		}

		validationResponse = &mockResp
		log.Printf("Successfully loaded mock ValidationResponse with validation_id: %s", validationResponse.ValidationId.GetValue())
	})

	return validationResponse, err
}

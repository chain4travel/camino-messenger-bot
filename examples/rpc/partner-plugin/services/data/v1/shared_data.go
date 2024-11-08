package helpers

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
)

var (
	properties []accommodationv1.PropertyExtendedInfo
	loadOnce   sync.Once
)

func LoadPropertiesMockData() []accommodationv1.PropertyExtendedInfo {
	loadOnce.Do(func() {
		// Assuming the JSON file is relative to the current file
		currentDir, err := os.Getwd()
		if err != nil {
			log.Printf("Error getting current directory: %v", err)
			return
		}

		filePath := filepath.Join(currentDir, "../../examples", "rpc", "partner-plugin", "mock_data", "properties.json")
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

// ReloadPropertiesMockData reloads the properties data from the JSON file
func ReloadPropertiesMockData() error {
	// Reset the sync.Once to allow reloading
	loadOnce = sync.Once{}

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting current directory: %v", err)
		return err
	}

	// Read properties file
	filePath := filepath.Join(currentDir, "../../examples", "rpc", "partner-plugin", "mock_data", "properties.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading properties file: %v", err)
		return err
	}

	// Clear existing properties
	properties = nil

	// Unmarshal new data
	if err := json.Unmarshal(data, &properties); err != nil {
		log.Printf("Error unmarshaling properties: %v", err)
		return err
	}

	log.Printf("Successfully reloaded %d properties", len(properties))
	return nil
}

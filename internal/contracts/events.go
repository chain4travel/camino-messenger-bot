package contracts

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/chain4travel/camino-messenger-bot/config"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EventListener contains fields to listen for events
type EventListener struct {
	ethClient *ethclient.Client
	cfg       *config.EvmConfig
}

// NewEventListener initializes the event listener with the provided details
func NewEventListener(client *ethclient.Client, cfg *config.EvmConfig) (*EventListener, error) {
	return &EventListener{
		ethClient: client,
		cfg:       cfg,
	}, nil
}

type event struct {
	BlockNumber uint64
	Data        map[string]interface{}
}

func (el *EventListener) Listen(eventName string, contractAddress common.Address, ABIFilePath string) (*event, error) {
	// Load the ABI from file
	contractABI, err := loadABI(ABIFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load ABI: %w", err)
	}

	// Get the ABI event details
	eventABI, ok := contractABI.Events[eventName]
	if !ok {
		return nil, fmt.Errorf("event %s not found in ABI", eventName)
	}

	// Prepare a filter query to get logs from the contract
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics:    [][]common.Hash{{eventABI.ID}}, // Filter for the event using its ID (signature)
	}

	// Query the logs
	logs, err := el.ethClient.FilterLogs(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to filter logs: %w", err)
	}

	var mostRecentEvent *event

	// Iterate over the logs and decode each one
	for _, vLog := range logs {
		// Create a map to store event data
		dataMap := make(map[string]interface{})

		// Unpack the event data into the map
		err := contractABI.UnpackIntoMap(dataMap, eventName, vLog.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack event log: %w", err)
		}

		// Update the most recent event
		mostRecentEvent = &event{
			BlockNumber: vLog.BlockNumber,
			Data:        dataMap,
		}
	}

	// Return the most recent event
	if mostRecentEvent == nil {
		return nil, fmt.Errorf("no events found for event %s", eventName)
	}

	return mostRecentEvent, nil
}

// Loads an ABI file
func loadABI(filePath string) (*abi.ABI, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	abi, err := abi.JSON(strings.NewReader(string(file)))
	if err != nil {
		return nil, err
	}
	return &abi, nil
}

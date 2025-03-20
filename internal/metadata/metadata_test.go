// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package metadata

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimestampInGrpcMetadata(t *testing.T) {
	// Create a metadata instance with timestamps
	md := Metadata{
		RequestID: "test-request-id",
		Sender:    "sender-address",
		Recipient: "recipient-address",
	}

	// Add timestamps
	md.Stamp("test-checkpoint-1")
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	md.Stamp("test-checkpoint-2")

	// Convert to gRPC metadata
	grpcMD := md.ToGrpcMD()

	// Verify timestamps are in the gRPC metadata
	timestampsValue, ok := grpcMD["timestamps"]
	require.True(t, ok, "timestamps should be present in gRPC metadata")
	require.Len(t, timestampsValue, 1, "timestamps should have one value")

	// Parse the timestamps JSON
	var timestamps map[string]int64
	err := json.Unmarshal([]byte(timestampsValue[0]), &timestamps)
	require.NoError(t, err, "should be able to unmarshal timestamps JSON")

	// Verify we have the expected timestamps
	require.Len(t, timestamps, 2, "should have 2 timestamps")

	// Check that the timestamps contain our checkpoints
	foundCheckpoint1 := false
	foundCheckpoint2 := false
	for key := range timestamps {
		if key == "0-test-checkpoint-1" {
			foundCheckpoint1 = true
		}
		if key == "1-test-checkpoint-2" {
			foundCheckpoint2 = true
		}
	}
	assert.True(t, foundCheckpoint1, "should have checkpoint 1 in timestamps")
	assert.True(t, foundCheckpoint2, "should have checkpoint 2 in timestamps")

	// Create a new metadata instance and extract from gRPC metadata
	newMD := Metadata{}
	err = newMD.FromGrpcMD(grpcMD)
	require.NoError(t, err, "should be able to extract metadata from gRPC metadata")

	// Verify the timestamps were correctly extracted
	require.Equal(t, md.Timestamps, newMD.Timestamps, "timestamps should be preserved through gRPC metadata conversion")
}

func TestStampAddsTimestampWithIndex(t *testing.T) {
	md := Metadata{}

	// Add multiple timestamps
	md.Stamp("checkpoint-1")
	md.Stamp("checkpoint-2")
	md.Stamp("checkpoint-3")

	// Verify timestamps have the expected format with index
	require.Len(t, md.Timestamps, 3, "should have 3 timestamps")

	// Check that keys have the expected format: "{index}-{checkpoint}"
	_, hasKey0 := md.Timestamps["0-checkpoint-1"]
	_, hasKey1 := md.Timestamps["1-checkpoint-2"]
	_, hasKey2 := md.Timestamps["2-checkpoint-3"]

	assert.True(t, hasKey0, "should have indexed checkpoint 1")
	assert.True(t, hasKey1, "should have indexed checkpoint 2")
	assert.True(t, hasKey2, "should have indexed checkpoint 3")

	// Verify timestamps are in ascending order
	assert.True(t, md.Timestamps["0-checkpoint-1"] <= md.Timestamps["1-checkpoint-2"],
		"timestamp 1 should be before or equal to timestamp 2")
	assert.True(t, md.Timestamps["1-checkpoint-2"] <= md.Timestamps["2-checkpoint-3"],
		"timestamp 2 should be before or equal to timestamp 3")
}

func TestStampOnAddsSpecificTimestamp(t *testing.T) {
	md := Metadata{}

	// Add timestamps with specific values
	timestamp1 := int64(1000)
	timestamp2 := int64(2000)

	md.StampOn("checkpoint-1", timestamp1)
	md.StampOn("checkpoint-2", timestamp2)

	// Verify timestamps have the expected values
	require.Len(t, md.Timestamps, 2, "should have 2 timestamps")
	assert.Equal(t, timestamp1, md.Timestamps["0-checkpoint-1"], "should have correct timestamp value for checkpoint 1")
	assert.Equal(t, timestamp2, md.Timestamps["1-checkpoint-2"], "should have correct timestamp value for checkpoint 2")
}

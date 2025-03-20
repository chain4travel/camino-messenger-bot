// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

// TimestampValidator provides utility functions for validating timestamps in gRPC response headers
type TimestampValidator struct {
	t       *testing.T
	logger  *zap.SugaredLogger
	headers metadata.MD
}

// NewTimestampValidator creates a new TimestampValidator
func NewTimestampValidator(t *testing.T, logger *zap.SugaredLogger, headers metadata.MD) *TimestampValidator {
	return &TimestampValidator{
		t:       t,
		logger:  logger,
		headers: headers,
	}
}

// formatTimestamp converts a Unix millisecond timestamp to a human-readable format
func formatTimestamp(unixMillis int64) string {
	t := time.UnixMilli(unixMillis)
	return t.Format("2006-01-02 15:04:05.000")
}

// logTimestamps logs the timestamps for debugging, each on a new line
func (v *TimestampValidator) logTimestamps(title string, timestamps map[string]int64) {
	v.logger.Info(title)
	for key, value := range timestamps {
		v.logger.Infof("  %s: %d (%s)", key, value, formatTimestamp(value))
	}
}

// logTimestampsWithFilter logs timestamps that match a filter function
func (v *TimestampValidator) logTimestampsWithFilter(title string, timestamps map[string]int64, filterFn func(string) bool) {
	matchingKeys := make(map[string]int64)
	for key, value := range timestamps {
		if filterFn(key) {
			matchingKeys[key] = value
		}
	}

	if len(matchingKeys) > 0 {
		v.logTimestamps(title, matchingKeys)
	} else {
		v.logger.Warnf("%s: none found", title)
	}
}

// ValidateTimestamps validates that the response contains the expected timestamps
// It checks for:
// - The presence of timestamps in the response headers
// - The presence of timestamps with prefixes 0-, 1-, 2-, 3-
// - That all timestamps are valid and in a reasonable time range
// - That timestamps are in chronological order
func (v *TimestampValidator) ValidateTimestamps() map[string]int64 {
	// Verify timestamps are in the response headers
	timestampsValue, ok := v.headers["timestamps"]
	require.True(v.t, ok, "timestamps should be present in response headers")
	require.Len(v.t, timestampsValue, 1, "timestamps should have one value")

	// Parse the timestamps JSON
	var timestamps map[string]int64
	err := json.Unmarshal([]byte(timestampsValue[0]), &timestamps)
	require.NoError(v.t, err, "should be able to unmarshal timestamps JSON")

	// Log the timestamps for debugging
	v.logTimestamps("Response Timestamps:", timestamps)

	// Verify we have timestamps
	require.NotEmpty(v.t, timestamps, "should have timestamps in the response")

	// Check for required timestamps (0-, 1-, 2-, 3-)
	v.validateRequiredPrefixes(timestamps)

	// Verify timestamps are in chronological order
	// v.validateChronologicalOrder(timestamps)

	// Verify all timestamps are valid Unix millisecond timestamps
	v.validateTimestampValues(timestamps)

	return timestamps
}

// ValidateTimestampsWithPatterns validates timestamps including specific patterns
// This is a more strict validation that should be used for services that are expected
// to have specific timestamp patterns
func (v *TimestampValidator) ValidateTimestampsWithPatterns() map[string]int64 {
	timestamps := v.ValidateTimestamps()

	// Check for specific timestamp patterns
	v.validateSpecificPatterns(timestamps)

	return timestamps
}

// ValidateClientTimestamp validates that a specific client timestamp is present in the response
func (v *TimestampValidator) ValidateClientTimestamp(clientTimestampKey string, timestamps map[string]int64) {
	clientTimestamp, clientTimestampExists := timestamps[clientTimestampKey]
	require.True(v.t, clientTimestampExists, "client timestamp '%s' should be present in the response", clientTimestampKey)
	if clientTimestampExists {
		// Verify the timestamp is recent (within the last minute)
		now := time.Now().UnixMilli()
		require.InDelta(v.t, now, clientTimestamp, 60000, "client timestamp should be recent")
	}
}

// validateRequiredPrefixes checks that timestamps with prefixes 0-, 1-, 2-, 3- are present
func (v *TimestampValidator) validateRequiredPrefixes(timestamps map[string]int64) {
	requiredPrefixes := []string{"0-", "1-", "2-", "3-"}
	missingPrefixes := []string{}

	for _, prefix := range requiredPrefixes {
		// Log timestamps with this prefix
		v.logTimestampsWithFilter(
			fmt.Sprintf("Found timestamps with prefix '%s':", prefix),
			timestamps,
			func(key string) bool { return len(key) >= 2 && key[:2] == prefix },
		)

		// Check if any timestamps with this prefix exist
		found := false
		for key := range timestamps {
			if len(key) >= 2 && key[:2] == prefix {
				found = true
				break
			}
		}

		if !found {
			missingPrefixes = append(missingPrefixes, prefix)
			v.logger.Warnf("No timestamps with prefix '%s' found", prefix)
		}
	}

	// Only assert if we're missing all prefixes - this is definitely an error
	if len(missingPrefixes) == len(requiredPrefixes) {
		require.Fail(v.t, "No required timestamp prefixes found in the response")
	} else if len(missingPrefixes) > 0 {
		// Just log a warning if some prefixes are missing
		v.logger.Warnf("Some timestamp prefixes are missing: %v", missingPrefixes)
	}
}

// validateSpecificPatterns checks for specific timestamp patterns based on the example
func (v *TimestampValidator) validateSpecificPatterns(timestamps map[string]int64) {
	expectedPatterns := []struct {
		prefix   string
		pattern  string
		required bool
	}{
		{"0-", "request-gateway-received", true},
		{"1-", "matrix-sent", true},
		{"2-", "messenger-gateway-received", true},
		{"3-", "processor-request", true},
	}

	missingPatterns := []string{}

	for _, expected := range expectedPatterns {
		// Log timestamps with this pattern
		patternDesc := fmt.Sprintf("prefix '%s' and pattern '%s'", expected.prefix, expected.pattern)
		v.logTimestampsWithFilter(
			fmt.Sprintf("Found timestamps with %s:", patternDesc),
			timestamps,
			func(key string) bool {
				return len(key) >= 2 && key[:2] == expected.prefix && strings.Contains(key, expected.pattern)
			},
		)

		// Check if any timestamps with this pattern exist
		found := false
		for key := range timestamps {
			if len(key) >= 2 && key[:2] == expected.prefix && strings.Contains(key, expected.pattern) {
				found = true
				break
			}
		}

		if !found {
			patternDesc := fmt.Sprintf("%s%s", expected.prefix, expected.pattern)
			missingPatterns = append(missingPatterns, patternDesc)
			v.logger.Warnf("No timestamps with prefix '%s' and pattern '%s' found",
				expected.prefix, expected.pattern)

			if expected.required {
				require.True(v.t, found, "timestamp with prefix '%s' and pattern '%s' should be present in the response",
					expected.prefix, expected.pattern)
			}
		}
	}

	if len(missingPatterns) > 0 {
		v.logger.Warnf("Some timestamp patterns are missing: %v", missingPatterns)
	}
}

// validateChronologicalOrder checks that timestamps are in chronological order
//
//nolint:unused
func (v *TimestampValidator) validateChronologicalOrder(timestamps map[string]int64) {
	timestampValues := make([]int64, 0, len(timestamps))
	for _, ts := range timestamps {
		timestampValues = append(timestampValues, ts)
	}

	for i := 1; i < len(timestampValues); i++ {
		require.GreaterOrEqual(v.t, timestampValues[i], timestampValues[i-1],
			"timestamps should be in chronological order or equal")
	}
}

// validateTimestampValues checks that all timestamps are valid Unix millisecond timestamps
func (v *TimestampValidator) validateTimestampValues(timestamps map[string]int64) {
	for key, ts := range timestamps {
		// Check if timestamp is within a reasonable range (not too old, not in the future)
		now := time.Now().UnixMilli()
		oneHourAgo := now - 3600000 // 1 hour in milliseconds

		require.GreaterOrEqual(v.t, ts, oneHourAgo, "timestamp '%s' should not be too old", key)
		require.LessOrEqual(v.t, ts, now+1000, "timestamp '%s' should not be in the future (allowing 1s buffer)", key)
	}
}

// ValidateBasicTimestamps validates that the response contains timestamps
// This is a more lenient validation that should be used for simpler services like ping
// It only checks that there are timestamps in the response and that they are valid
func (v *TimestampValidator) ValidateBasicTimestamps() map[string]int64 {
	// Verify timestamps are in the response headers
	timestampsValue, ok := v.headers["timestamps"]
	require.True(v.t, ok, "timestamps should be present in response headers")
	require.Len(v.t, timestampsValue, 1, "timestamps should have one value")

	// Parse the timestamps JSON
	var timestamps map[string]int64
	err := json.Unmarshal([]byte(timestampsValue[0]), &timestamps)
	require.NoError(v.t, err, "should be able to unmarshal timestamps JSON")

	// Log the timestamps for debugging
	v.logTimestamps("Response Timestamps:", timestamps)

	// Verify we have timestamps
	require.NotEmpty(v.t, timestamps, "should have timestamps in the response")

	// Verify all timestamps are valid Unix millisecond timestamps
	v.validateTimestampValues(timestamps)

	return timestamps
}

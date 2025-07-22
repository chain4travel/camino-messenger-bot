// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package metadata

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimestamps(t *testing.T) {
	timestamps := Timestamps{
		"0-started":   1633072800000,
		"1-processed": 1633072801000,
		"2-ended":     1633072802000,
	}
	timestampsStr, err := timestamps.MarshalToString()
	require.NoError(t, err)
	timestampsFromStr, err := TimestampsFromString(timestampsStr)
	require.NoError(t, err)
	require.Equal(t, timestamps, timestampsFromStr)

	before := time.Now().UnixMilli()
	time.Sleep(time.Millisecond)
	timestamps.Stamp(CheckpointGRPCResponseSent)
	time.Sleep(time.Millisecond)
	after := time.Now().UnixMilli()

	actual := timestamps[fmt.Sprintf("3-%s", CheckpointGRPCResponseSent)]

	require.Greater(t, actual, before)
	require.Less(t, actual, after)
}

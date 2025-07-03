// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package metadata

import (
	"maps"
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

	now := time.Now().Unix()
	nowPlus100 := now + 100

	expectedTimestamps := Timestamps{}
	maps.Copy(expectedTimestamps, timestamps)
	expectedTimestamps["3-now"] = now
	expectedTimestamps["4-now+100"] = nowPlus100

	timestamps.StampOn("now", now)
	timestamps.StampOn("now+100", nowPlus100)

	require.Equal(t, expectedTimestamps, timestamps)
}

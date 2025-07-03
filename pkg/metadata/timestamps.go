// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package metadata

import (
	"encoding/json"
	"fmt"
	"time"
)

type Timestamps map[string]int64

func TimestampsFromString(s string) (Timestamps, error) {
	var timestamps Timestamps
	if err := json.Unmarshal([]byte(s), &timestamps); err != nil {
		return nil, fmt.Errorf("error unmarshalling timestamps: %w", err)
	}
	return timestamps, nil
}

func (t Timestamps) Stamp(checkpoint string) {
	t.StampOn(checkpoint, time.Now().UnixMilli())
}

func (t Timestamps) StampOn(checkpoint string, timestamp int64) {
	idx := len(t) // for analysis' sake, we want to know the order of the checkpoints
	t[fmt.Sprintf("%d-%s", idx, checkpoint)] = timestamp
}

func (t Timestamps) MarshalToString() (string, error) {
	timestampsJSON, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("error marshalling timestamps: %w", err)
	}
	return string(timestampsJSON), nil
}

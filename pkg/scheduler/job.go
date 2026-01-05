// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package scheduler

import "time"

type Job struct {
	Name           string
	LastExecutedAt time.Time
	Period         time.Duration
}

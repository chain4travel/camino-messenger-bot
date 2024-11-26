package scheduler

import "time"

type Job struct {
	Name           string
	LastExecutedAt time.Time
	Period         time.Duration
}

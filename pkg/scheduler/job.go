package scheduler

import "time"

type Job struct {
	Name      string
	ExecuteAt time.Time
	Period    time.Duration
}

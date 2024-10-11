package scheduler

import (
	"time"
)

func newTimer() *timer {
	t := time.NewTimer(time.Second)
	t.Stop()
	return &timer{
		Timer:  t,
		stopCh: make(chan struct{}, 1),
	}
}

type timer struct {
	*time.Timer
	stopCh chan struct{}
}

// StartOnce starts the timer once and starts goroutine with [f] call when the timer expires.
// Returns a channel that is closed upon completion.
//
// Should not be called on already running timer.
func (t *timer) StartOnce(d time.Duration, f func()) chan struct{} {
	t.Reset(d)
	stopSignalCh := make(chan struct{})
	go func() {
		defer close(stopSignalCh)
		for {
			select {
			case <-t.stopCh:
				return
			case <-t.C:
				go f()
				t.Stop()
			}
		}
	}()
	return stopSignalCh
}

// Start starts the timer and starts goroutine with [f] call when the timer ticks.
// Returns a channel that is closed after timer is stopped.
//
// Should not be called on already running timer.
func (t *timer) Start(d time.Duration, f func()) chan struct{} {
	t.Reset(d)
	stopSignalCh := make(chan struct{})
	go func() {
		defer close(stopSignalCh)
		for {
			select {
			case <-t.stopCh:
				return
			case <-t.C:
				go f()
				t.Reset(d)
			}
		}
	}()
	return stopSignalCh
}

// Stop stops the timer.
//
// Stop should not be called on already stopped timer.
func (t *timer) Stop() {
	t.stopCh <- struct{}{}
	t.Timer.Stop()
}

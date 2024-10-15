package scheduler

import (
	"sync/atomic"
	"time"

	"github.com/jonboulle/clockwork"
)

func newTimer(clock clockwork.Clock) *timer {
	t := clock.NewTimer(time.Second)
	t.Stop()
	return &timer{
		timer:  t,
		stopCh: make(chan struct{}, 1),
	}
}

type timer struct {
	timer   clockwork.Timer
	stopped atomic.Bool
	stopCh  chan struct{}
}

// StartOnce starts the timer once and starts goroutine with [f] call when the timer expires.
// Returns a channel that is closed upon completion.
//
// Should not be called on already running timer.
func (t *timer) StartOnce(d time.Duration, f func()) chan struct{} {
	t.timer.Reset(d)
	stopSignalCh := make(chan struct{})
	go func() {
		defer close(stopSignalCh)
		for {
			select {
			case <-t.stopCh:
				t.stopped.Store(true) // TODO@ test
				return
			case <-t.timer.Chan():
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
	t.timer.Reset(d)
	stopSignalCh := make(chan struct{})
	go func() {
		defer close(stopSignalCh)
		for {
			select {
			case <-t.stopCh:
				t.stopped.Store(true) // TODO@ test
				return
			case <-t.timer.Chan():
				go f()
				t.timer.Reset(d)
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
	t.timer.Stop()
}

// TODO@ test
func (t *timer) IsStopped() bool {
	return t.stopped.Load()
}

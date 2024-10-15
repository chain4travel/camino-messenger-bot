package scheduler

import (
	"sync/atomic"
	"time"

	"github.com/jonboulle/clockwork"
)

func NewTimer(clock clockwork.Clock) *Timer {
	timer := &Timer{
		clock:    clock,
		timer:    clock.NewTimer(time.Second),
		stopChan: make(chan time.Time),
	}
	timer.timer.Stop()
	timer.stopped.Store(true)
	return timer
}

type Timer struct {
	clock    clockwork.Clock
	timer    clockwork.Timer
	stopped  atomic.Bool
	stopChan chan time.Time
	once     bool
}

// StartOnce starts the timer once and starts goroutine with [f] call when the timer expires.
// Returns a channel that will receive stop-time and will be closed after timer is stopped.
//
// Should not be called on already running timer, will panic.
func (t *Timer) StartOnce(d time.Duration, f func()) chan time.Time {
	if !t.stopped.Load() {
		panic("timer is already running")
	}
	t.once = true
	return t.start(d, f)
}

// Start starts the timer and starts goroutine with [f] call when the timer ticks.
// Returns a channel that will receive stop-time and will be closed after timer is stopped.
//
// Should not be called on already running timer, will panic.
func (t *Timer) Start(d time.Duration, f func()) chan time.Time {
	if !t.stopped.Load() {
		panic("timer is already running")
	}
	t.once = false
	return t.start(d, f)
}

// Stop stops the timer. If the timer is already stopped, it panics.
func (t *Timer) Stop() {
	if t.stopped.Load() {
		panic("timer is already stopped")
	}
	t.stopChan <- t.clock.Now()
}

// IsStopped returns true if the timer is stopped.
func (t *Timer) IsStopped() bool {
	return t.stopped.Load()
}

func (t *Timer) start(d time.Duration, f func()) chan time.Time {
	t.stopped.Store(false)
	t.timer.Reset(d)
	stopTimeChan := make(chan time.Time)

	go func() {
		for {
			select {
			case stopTime := <-t.stopChan:
				t.timer.Stop()
				t.stopped.Store(true)
				stopTimeChan <- stopTime
				close(stopTimeChan) // will be blocked until stopTimeCh read, but its fine
				return
			case <-t.timer.Chan():
				go f()
				if t.once {
					t.Stop()
				} else {
					t.timer.Reset(d)
				}
			}
		}
	}()

	return stopTimeChan
}

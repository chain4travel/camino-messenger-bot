package scheduler

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jonboulle/clockwork"
)

func NewTimer(clock clockwork.Clock) *Timer {
	timer := &Timer{
		clock:    clock,
		timer:    clock.NewTimer(time.Second),
		stopChan: make(chan time.Time, 1),
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

// StartOnce starts with goroutine that will call [f] when the timer expires. The timer will be stopped after the first expiration.
// Returns a channel that will receive stop-time and will be closed after timer is stopped.
//
// Should not be called on already running timer, will panic.
func (t *Timer) StartOnce(d time.Duration, f func(time.Time)) chan time.Time {
	if !t.stopped.Load() {
		panic("timer is already running")
	}
	t.once = true
	return t.start(d, f)
}

// Starts timer with goroutine that will call [f] when timer ticks. The timer will continue to tick until stopped.
// Returns a channel that will receive stop-time and will be closed after timer is stopped.
//
// Should not be called on already running timer, will panic.
func (t *Timer) Start(d time.Duration, f func(time.Time)) chan time.Time {
	if !t.stopped.Load() {
		panic("timer is already running")
	}
	t.once = false
	return t.start(d, f)
}

// Stops the timer. If the timer is already stopped, it panics.
func (t *Timer) Stop() {
	if t.stopped.Load() {
		panic("timer is already stopped")
	}
	t.stopped.Store(true)
	t.stopChan <- t.clock.Now()
}

// Returns true if the timer is stopped.
func (t *Timer) IsStopped() bool {
	return t.stopped.Load()
}

func (t *Timer) start(d time.Duration, f func(time.Time)) chan time.Time {
	t.stopped.Store(false)
	t.timer.Reset(d)
	stopTimeChan := make(chan time.Time)

	go func() {
		for {
			select {
			case stopTime := <-t.stopChan:
				fmt.Printf("Received stopTime %d from t.stopChan\n", stopTime.UnixNano())
				fmt.Printf("Stopping timer\n")
				t.timer.Stop()
				fmt.Printf("Sending stopTime %d to stopTimeChan\n", stopTime.UnixNano())
				stopTimeChan <- stopTime
				fmt.Printf("closing stopTimeChan\n")
				close(stopTimeChan) // will be blocked until stopTimeCh read, but its fine
				fmt.Printf("returning\n")
				return
			case tickTime := <-t.timer.Chan():
				fmt.Printf("Received tickTime %d from t.stopChan\n", tickTime.UnixNano())
				fmt.Printf("Starting handler goroutine\n")
				go f(tickTime)
				if t.once {
					fmt.Printf("Stopping timer\n")
					t.Stop()
				} else {
					fmt.Printf("Resetting timer\n")
					t.timer.Reset(d)
				}
				fmt.Printf("looping\n")
			}
		}
	}()

	return stopTimeChan
}

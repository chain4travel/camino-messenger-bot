package scheduler

import (
	"sync/atomic"
	"time"

	"github.com/jonboulle/clockwork"
)

func NewTimer(clock clockwork.Clock) *Timer {
	timer := &Timer{
		clock:    clock,
		stopChan: make(chan time.Time, 1),
	}
	timer.stopped.Store(true)
	return timer
}

type Timer struct {
	clock    clockwork.Clock
	stopped  atomic.Bool
	isTicker atomic.Bool
	stopChan chan time.Time
}

// StartOnce starts timer with goroutine that will call [f] when the timer expires. The timer will be stopped after the first expiration.
// Returns a channel that will receive stop-time and will be closed after timer is stopped.
//
// Should not be called on already running timer, will panic.
func (t *Timer) StartOnce(timerDuration time.Duration, f func(time.Time)) (stopChan chan time.Time) {
	if !t.stopped.Load() {
		panic("timer is already running")
	}

	t.stopped.Store(false)
	t.isTicker.Store(false)
	timer := t.clock.NewTimer(timerDuration)
	stopTimeChan := make(chan time.Time)

	go func() {
		for i := 0; ; i++ {
			select {
			case stopTime := <-t.stopChan:
				t.stopped.Store(true)
				_ = timer.Stop()
				stopTimeChan <- stopTime
				close(stopTimeChan) // will be blocked until stopTimeCh read, but its fine
				return
			case tickTime := <-timer.Chan():
				// TODO@ don't tick if atomic stop?
				go f(tickTime)
				t.stopChan <- tickTime
			}
		}
	}()

	return stopTimeChan
}

// Starts ticker with goroutine that will call [f] on each tick. The ticker will continue to tick until stopped.
// Returns a channel that will receive stop-time and will be closed after ticker is stopped.
//
// Should not be called on already running ticker, will panic.
func (t *Timer) Start(timerDuration time.Duration, f func(time.Time)) (stopChan chan time.Time) {
	if !t.stopped.Load() {
		panic("timer is already running")
	}
	if timerDuration == 0 {
		panic("cannot start non-once timer with zero duration")
	}

	t.stopped.Store(false)
	t.isTicker.Store(true)
	ticker := t.clock.NewTicker(timerDuration)
	stopTimeChan := make(chan time.Time)

	go func() {
		for i := 0; ; i++ {
			select {
			case stopTime := <-t.stopChan:
				t.stopped.Store(true)
				ticker.Stop()
				stopTimeChan <- stopTime
				close(stopTimeChan) // will be blocked until stopTimeCh read, but its fine
				return
			case tickTime := <-ticker.Chan():
				// TODO@ don't tick if atomic stop?
				go f(tickTime)
			}
		}
	}()

	return stopTimeChan
}

// Stops the timer. If the timer is already stopped, it panics.
func (t *Timer) Stop() {
	if t.stopped.Load() {
		panic("timer is already stopped")
	}
	// TODO@ stop atomic should be there
	stopTime := t.clock.Now()
	t.stopChan <- stopTime
}

// Returns true if the timer is stopped.
func (t *Timer) IsStopped() bool {
	return t.stopped.Load()
}

// Returns true if the timer is running as ticker or if timer is stopped and was running as ticker before that.
func (t *Timer) IsTicker() bool {
	return t.isTicker.Load()
}

package scheduler

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/jonboulle/clockwork"
)

func NewTimer(clock clockwork.Clock, name string) *Timer {
	timer := &Timer{
		clock:    clock,
		name:     name,
		stopChan: make(chan time.Time, 1),
	}
	timer.stopped.Store(true)
	return timer
}

type Timer struct {
	name     string
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

	fmt.Printf("%s-%d (goID %d): timer.StartOnce\n", t.name, timerDuration, goID())
	t.stopped.Store(false)
	t.isTicker.Store(false)
	timer := t.clock.NewTimer(timerDuration)
	stopTimeChan := make(chan time.Time)

	go func() {
		for i := 0; ; i++ {
			fmt.Printf("%s-%d (goID %d) [%d]: timer.StartOnce.go: Select\n", t.name, timerDuration, goID(), i)
			select {
			case stopTime := <-t.stopChan:
				fmt.Printf("%s-%d (goID %d) [%d]: timer.StartOnce.go: Received stopTime %d from t.stopChan\n",
					t.name, timerDuration, goID(), i, stopTime.UnixNano())
				t.stopped.Store(true)
				fmt.Printf("%s-%d (goID %d) [%d]: timer.StartOnce.go: Stopping internal timer\n",
					t.name, timerDuration, goID(), i)
				_ = timer.Stop()
				fmt.Printf("%s-%d (goID %d) [%d]: timer.StartOnce.go: Sending stopTime %d to stopTimeChan\n",
					t.name, timerDuration, goID(), i, stopTime.UnixNano())
				stopTimeChan <- stopTime
				fmt.Printf("%s-%d (goID %d) [%d]: timer.StartOnce.go: Closing stopTimeChan\n",
					t.name, timerDuration, goID(), i)
				close(stopTimeChan) // will be blocked until stopTimeCh read, but its fine
				fmt.Printf("%s-%d (goID %d) [%d]: timer.StartOnce.go: Returning\n",
					t.name, timerDuration, goID(), i)
				return
			case tickTime := <-timer.Chan():
				// TODO@ don't tick if atomic stop?
				fmt.Printf("%s-%d (goID %d) [%d]: timer.StartOnce.go: Received tickTime %d from t.timer.Chan()\n",
					t.name, timerDuration, goID(), i, tickTime.UnixNano())
				fmt.Printf("%s-%d (goID %d) [%d]: timer.StartOnce.go: Starting handler goroutine\n",
					t.name, timerDuration, goID(), i)
				go f(tickTime)
				fmt.Printf("%s-%d (goID %d) [%d]: timer.StartOnce.go: Sending tickTime to t.stopChan\n",
					t.name, timerDuration, goID(), i)
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

	fmt.Printf("%s-%d (goID %d): timer.Start\n", t.name, timerDuration, goID())
	t.stopped.Store(false)
	t.isTicker.Store(true)
	ticker := t.clock.NewTicker(timerDuration)
	stopTimeChan := make(chan time.Time)

	go func() {
		for i := 0; ; i++ {
			fmt.Printf("%s-%d (goID %d) [%d]: timer.Start.go: Select\n", t.name, timerDuration, goID(), i)
			select {
			case stopTime := <-t.stopChan:
				fmt.Printf("%s-%d (goID %d) [%d]: timer.Start.go: Received stopTime %d from t.stopChan\n",
					t.name, timerDuration, goID(), i, stopTime.UnixNano())
				t.stopped.Store(true)
				fmt.Printf("%s-%d (goID %d) [%d]: timer.Start.go: Stopping internal timer\n",
					t.name, timerDuration, goID(), i)
				ticker.Stop()
				fmt.Printf("%s-%d (goID %d) [%d]: timer.Start.go: Sending stopTime %d to stopTimeChan\n",
					t.name, timerDuration, goID(), i, stopTime.UnixNano())
				stopTimeChan <- stopTime
				fmt.Printf("%s-%d (goID %d) [%d]: timer.Start.go: Closing stopTimeChan\n",
					t.name, timerDuration, goID(), i)
				close(stopTimeChan) // will be blocked until stopTimeCh read, but its fine
				fmt.Printf("%s-%d (goID %d) [%d]: timer.Start.go: Returning\n",
					t.name, timerDuration, goID(), i)
				return
			case tickTime := <-ticker.Chan():
				// TODO@ don't tick if atomic stop?
				fmt.Printf("%s-%d (goID %d) [%d]: timer.Start.go: Received tickTime %d from t.timer.Chan()\n",
					t.name, timerDuration, goID(), i, tickTime.UnixNano())
				fmt.Printf("%s-%d (goID %d) [%d]: timer.Start.go: Starting handler goroutine\n",
					t.name, timerDuration, goID(), i)
				go f(tickTime)
			}
		}
	}()

	return stopTimeChan
}

// Stops the timer. If the timer is already stopped, it panics.
func (t *Timer) Stop() {
	fmt.Printf("%s (goID %d): timer.Stop\n", t.name, goID())
	if t.stopped.Load() {
		panic("timer is already stopped")
	}
	// TODO@ stop atomic should be there
	stopTime := t.clock.Now()
	fmt.Printf("%s (goID %d): timer.Stop: Sending stopTime %d to t.stopChan\n", t.name, goID(), stopTime.UnixNano())
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

func goID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := bytes.Fields(buf[:n])[1]
	id, _ := strconv.Atoi(string(idField))
	return id
}

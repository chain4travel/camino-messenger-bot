package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

// TODO@ update tests after implementation changes

func TestNewTimer(t *testing.T) {
	require := require.New(t)
	clock := clockwork.NewFakeClockAt(time.Unix(0, 100))
	timer := NewTimer(clock, "")

	require.Equal(clock, timer.clock)
	require.Equal(true, timer.stopped.Load())
	require.Equal(true, timer.IsStopped())

	timer.stopped.Store(false)
	require.Equal(false, timer.IsStopped())
}

func TestTimer_StartOnce(t *testing.T) {
	t.Run("timer expires", func(t *testing.T) {
		require := require.New(t)
		clock := clockwork.NewFakeClockAt(time.Unix(0, 100))
		timer := NewTimer(clock, "")
		duration := time.Millisecond
		timeout := 10 * time.Millisecond
		epsilon := time.Millisecond
		called := make(chan struct{})
		startTime := clock.Now()

		require.Greater(duration, time.Duration(1))
		require.Less(duration, timeout-epsilon)

		stopSignalCh := timer.StartOnce(duration, func(time.Time) {
			close(called)
		})

		clock.Advance(duration - 1)

		select {
		case <-time.After(timeout):
		case <-stopSignalCh:
			require.FailNow("timer should not have stopped before the expected duration")
		}

		clock.Advance(1)

		select {
		case <-stopSignalCh:
		case <-time.After(timeout):
			require.FailNow("timer did not stop within the expected duration")
		}

		select {
		case <-called:
		case <-time.After(timeout):
			require.FailNow("function was not called within the expected duration")
		}

		require.Equal(clock.Since(startTime), duration)
	})

	t.Run("timer is stopped manually", func(t *testing.T) {
		require := require.New(t)
		clock := clockwork.NewFakeClockAt(time.Unix(0, 100))
		timer := NewTimer(clock, "")
		duration := time.Millisecond
		timeout := 10 * time.Millisecond
		epsilon := time.Millisecond
		called := make(chan struct{})
		startTime := clock.Now()

		require.Greater(duration, time.Duration(1))
		require.Less(duration, timeout-epsilon)

		stopSignalCh := timer.StartOnce(duration, func(time.Time) {
			close(called)
		})

		runDuration := duration - 1
		clock.Advance(runDuration)

		timer.Stop()

		select {
		case <-stopSignalCh:
		case <-called:
			require.FailNow("function should not have been called after timer was stopped")
		case <-time.After(timeout):
			require.FailNow("timer did not stop within the expected duration")
		}

		require.Equal(clock.Since(startTime), runDuration)
		close(called) // ensure the function is not called after the timer is stopped
	})
}

func TestTimer_Start(t *testing.T) {
	require := require.New(t)
	clock := clockwork.NewFakeClockAt(time.Unix(0, 100))
	timer := NewTimer(clock, "")

	duration := time.Millisecond
	timeout := 10 * time.Millisecond
	epsilon := time.Millisecond
	startTime := clock.Now()
	maxCallCount := 5
	callCh := make(chan struct{})

	require.Greater(duration, time.Duration(1))
	require.Less(duration, timeout-epsilon)

	stopCh := timer.Start(duration, func(time.Time) {
		fmt.Printf("\n\nHandler called, goroutineID %d\n", goID())
		callCh <- struct{}{}
	})

	for i := 0; i < maxCallCount; i++ {
		require.Equal(duration*time.Duration(i), clock.Since(startTime))

		advanceDuration := duration - 1
		// advanceDuration := startTime.Add(duration*time.Duration(i+1)).Sub(clock.Now()) - 1
		nextTime := clock.Now().Add(advanceDuration)
		fmt.Printf("[%d] advancing %d time by %d to %d\n", i, clock.Now().UnixNano(), advanceDuration, nextTime.UnixNano())
		clock.Advance(advanceDuration)

		fmt.Printf("[%d] checking that handler isn't called within next %d\n", i, timeout)
		select {
		case <-callCh:
			require.FailNow("function should not have been called before the expected duration", "%d", timeout)
		case <-time.After(timeout):
		}

		advanceDuration = 1
		// advanceDuration = 2
		nextTime = clock.Now().Add(advanceDuration)
		fmt.Printf("[%d] advancing %d time by %d to %d\n", i, clock.Now().UnixNano(), advanceDuration, nextTime.UnixNano())
		clock.Advance(advanceDuration)

		fmt.Printf("[%d] checking that handler is called within next %d\n", i, timeout)
		select {
		case <-callCh:
		case <-time.After(timeout):
			require.FailNowf("function was not called within the expected duration", "%d", timeout*100)
		}

		require.Equal(duration*time.Duration(i+1), clock.Since(startTime))
	}

	timer.Stop()

	select {
	case <-stopCh:
	case <-time.After(timeout * 100):
		require.FailNow("timer did not stop within the expected duration")
	}

	require.Equal(duration*time.Duration(maxCallCount), clock.Since(startTime))
}

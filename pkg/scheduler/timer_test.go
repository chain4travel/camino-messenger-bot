package scheduler

import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestTimer_StartOnce(t *testing.T) {
	t.Run("function is called after duration", func(t *testing.T) {
		require := require.New(t)
		clock := clockwork.NewFakeClockAt(time.Unix(0, 100))
		timer := newTimer(clock)
		duration := time.Millisecond
		timeout := 10 * time.Millisecond
		epsilon := time.Millisecond
		called := make(chan struct{})
		startTime := clock.Now()

		require.Greater(duration, time.Duration(1))
		require.Less(duration, timeout-epsilon)

		stopSignalCh := timer.StartOnce(duration, func() {
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
		timer := newTimer(clock)
		duration := time.Millisecond
		timeout := 10 * time.Millisecond
		epsilon := time.Millisecond
		called := make(chan struct{})
		startTime := clock.Now()

		require.Greater(duration, time.Duration(1))
		require.Less(duration, timeout-epsilon)

		stopSignalCh := timer.StartOnce(duration, func() {
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
	timer := newTimer(clock)

	duration := time.Millisecond
	timeout := 10 * time.Millisecond
	epsilon := time.Millisecond
	startTime := clock.Now()
	callCount := 0
	lastCallCount := 0
	maxCallCount := 5
	callCh := make(chan struct{})

	require.Greater(duration, time.Duration(1))
	require.Less(duration, timeout-epsilon)

	stopSignalCh := timer.Start(duration, func() {
		callCount++
		callCh <- struct{}{}
	})

	for i := 0; i < maxCallCount; i++ {
		require.Equal(lastCallCount, callCount)
		require.Equal(duration*time.Duration(i), clock.Since(startTime))

		clock.Advance(duration - 1)

		select {
		case <-callCh:
			require.FailNow("function should not have been called before the expected duration")
		case <-time.After(timeout):
		}

		clock.Advance(1)

		select {
		case <-callCh:
		case <-time.After(timeout):
			require.FailNow("function was not called within the expected duration")
		}

		require.Equal(duration*time.Duration(i+1), clock.Since(startTime))
		require.Equal(lastCallCount+1, callCount)

		lastCallCount = callCount
	}

	timer.Stop()

	select {
	case <-stopSignalCh:
	case <-time.After(timeout):
		require.FailNow("timer did not stop within the expected duration")
	}

	require.Equal(duration*time.Duration(maxCallCount), clock.Since(startTime))
	require.Equal(maxCallCount, callCount)
}

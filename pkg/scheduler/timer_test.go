package scheduler

import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestNewTimer(t *testing.T) {
	require := require.New(t)
	clock := clockwork.NewFakeClockAt(time.Unix(0, 100))
	timer := NewTimer(clock)

	require.Equal(clock, timer.clock)

	require.Equal(false, timer.IsTicker())
	timer.isTicker.Store(true)
	require.Equal(true, timer.IsTicker())

	require.Equal(true, timer.IsStopped())
	timer.stopped.Store(false)
	require.Equal(false, timer.IsStopped())
}

func TestTimer_StartOnce(t *testing.T) {
	t.Run("timer expires", func(t *testing.T) {
		require := require.New(t)
		clock := clockwork.NewFakeClockAt(time.Unix(0, 100))
		timer := NewTimer(clock)
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

		require.False(timer.IsTicker())

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

		require.False(timer.IsTicker())
		require.Equal(clock.Since(startTime), duration)
	})

	t.Run("timer is stopped manually", func(t *testing.T) {
		require := require.New(t)
		clock := clockwork.NewFakeClockAt(time.Unix(0, 100))
		timer := NewTimer(clock)
		duration := time.Millisecond
		timeout := 10 * time.Millisecond
		epsilon := time.Millisecond
		callCh := make(chan struct{})
		startTime := clock.Now()

		require.Greater(duration, time.Duration(1))
		require.Less(duration, timeout-epsilon)

		stopSignalCh := timer.StartOnce(duration, func(time.Time) {
			close(callCh)
		})

		require.False(timer.IsTicker())

		runDuration := duration - 1
		clock.Advance(runDuration)

		timer.Stop()

		select {
		case <-stopSignalCh:
		case <-time.After(timeout):
			require.FailNow("timer did not stop within the expected duration")
		}

		require.False(timer.IsTicker())
		require.Equal(clock.Since(startTime), runDuration)

		select {
		case <-callCh:
			require.FailNow("function should not have been called after timer was stopped")
		case <-time.After(timeout):
		}
	})
}

func TestTimer_Start(t *testing.T) {
	require := require.New(t)
	clock := clockwork.NewFakeClockAt(time.Unix(0, 100))
	timer := NewTimer(clock)

	duration := time.Millisecond
	timeout := 10 * time.Millisecond
	epsilon := time.Millisecond
	startTime := clock.Now()
	maxCallCount := 5
	callCh := make(chan struct{})

	require.Greater(duration, time.Duration(1))
	require.Less(duration, timeout-epsilon)

	stopCh := timer.Start(duration, func(time.Time) {
		callCh <- struct{}{}
	})

	require.True(timer.IsTicker())

	for i := 0; i < maxCallCount; i++ {
		require.Equal(duration*time.Duration(i), clock.Since(startTime))

		clock.Advance(duration - 1)

		select {
		case <-callCh:
			require.FailNow("function should not have been called before the expected duration", "%d", timeout)
		case <-time.After(timeout):
		}

		clock.Advance(1)

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

	require.True(timer.IsTicker())
	require.Equal(duration*time.Duration(maxCallCount), clock.Since(startTime))

	select {
	case <-callCh:
		require.FailNow("function should not have been called after timer was stopped")
	case <-time.After(timeout):
	}
}

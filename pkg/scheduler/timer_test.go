package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimer_StartOnce(t *testing.T) {
	t.Run("function is called after duration", func(t *testing.T) {
		timer := newTimer()
		duration := 100 * time.Millisecond
		epsilon := time.Millisecond
		called := make(chan struct{})
		startTime := time.Now()

		stopSignalCh := timer.StartOnce(duration, func() {
			close(called)
		})

		select {
		case <-stopSignalCh:
			require.GreaterOrEqual(t, time.Since(startTime), duration)
		case timeout := <-time.After(duration + epsilon):
			require.FailNow(t, "timer did not stop within the expected duration", "expected less than %v+%v, got %v",
				duration, epsilon, timeout.Sub(startTime))
		}

		select {
		case <-called:
		case <-time.After(epsilon):
			require.FailNow(t, "function was not called within the expected duration")
		}
	})

	t.Run("timer is stopped manually", func(t *testing.T) {
		timer := newTimer()
		duration := 100 * time.Millisecond
		epsilon := time.Millisecond
		called := make(chan struct{})
		startTime := time.Now()
		runDuration := 50 * time.Millisecond

		stopSignalCh := timer.StartOnce(duration, func() {
			close(called)
		})

		time.Sleep(runDuration)
		timer.Stop()

		select {
		case <-stopSignalCh:
		case <-called:
			require.FailNow(t, "function should not have been called after timer was stopped")
		case timeout := <-time.After(epsilon):
			require.FailNow(t, "timer did not stop within the expected duration", "expected less than %v+%v, got %v",
				runDuration, epsilon, timeout.Sub(startTime))
		}

		close(called) // ensure the function is not called after the timer is stopped
	})
}

func TestTimer_Start(t *testing.T) {
	timer := newTimer()
	duration := 100 * time.Millisecond
	called := make(chan struct{})
	epsilon := time.Millisecond
	startTime := time.Now()
	lastCallTime := startTime
	callCount := 0
	lastCallCount := 0
	maxCallCount := 5
	totalDuration := duration * time.Duration(maxCallCount)
	totalEpsilon := epsilon * time.Duration(maxCallCount)
	callTimeCh := make(chan time.Time)

	stopSignalCh := timer.Start(duration, func() {
		callTimeCh <- time.Now()
		callCount++
	})

	go func() {
		for callTime := range callTimeCh {
			require.InEpsilon(t, duration, callTime.Sub(lastCallTime), float64(epsilon))
			require.Equal(t, callCount, lastCallCount+1)
			lastCallTime = callTime
			lastCallCount = callCount
			if callCount >= maxCallCount {
				timer.Stop()
				close(called)
			}
		}
	}()

	select {
	case <-stopSignalCh:
		require.GreaterOrEqual(t, time.Since(startTime), totalDuration)
	case timeout := <-time.After(totalDuration + totalEpsilon):
		require.FailNow(t, "timer did not stop within the expected duration", "expected less than %v+%v, got %v",
			totalDuration, totalEpsilon, timeout.Sub(startTime))
		require.FailNow(t, "timer did not stop within the expected duration")
	}

	require.Equal(t, maxCallCount, callCount)
}

package scheduler

import (
	"context"
	reflect "reflect"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestScheduler_Start(t *testing.T) {
	// *** base setup

	require := require.New(t)
	ctx := context.Background()
	clock := clockwork.NewFakeClockAt(time.Unix(0, 100))
	ctrl := gomock.NewController(t)
	storage := NewMockStorage(ctrl)
	epsilon := time.Millisecond
	timeout := 100 * time.Millisecond

	earlyJobExecuted := make(chan string)
	nowJobExecuted := make(chan string)
	lateJobExecuted := make(chan string)

	earlyJob := Job{
		Name:      "early_job",
		ExecuteAt: clock.Now().Add(-1),
		Period:    1000,
	}
	nowJob := Job{
		Name:      "now_job",
		ExecuteAt: clock.Now(),
		Period:    1003,
	}
	lateJob := Job{
		Name:      "late_job",
		ExecuteAt: clock.Now().Add(1),
		Period:    1007,
	}
	jobs := []*Job{&earlyJob, &nowJob, &lateJob}
	jobsExecChansMap := map[string]chan string{
		earlyJob.Name: earlyJobExecuted,
		nowJob.Name:   nowJobExecuted,
		lateJob.Name:  lateJobExecuted,
	}

	// this is needed for correct time-advancement sequence

	require.Less(earlyJob.ExecuteAt, clock.Now())
	require.Equal(nowJob.ExecuteAt, clock.Now())

	require.Less(earlyJob.Period, nowJob.Period)
	require.Less(nowJob.Period, lateJob.Period)
	require.Less(lateJob.Period, timeout-epsilon)

	// *** mock & executionSequence setup

	numberOfFullCycles := 4                // number of how many times each job will be executed
	require.Greater(numberOfFullCycles, 1) // at least more than initial startOnce execution

	type executionStep struct {
		time         time.Time
		jobs         []Job
		initialTimer bool
	}
	executionSequence := []executionStep{}

	// main goroutine
	storageSession := &dummySession{}
	storage.EXPECT().NewSession(ctx).Return(storageSession, nil)
	storage.EXPECT().GetAllJobs(ctx, storageSession).Return(jobs, nil)
	storage.EXPECT().Abort(storageSession)

	// startOnce and periodic start goroutines

	// its clock.Now().Add(-1), but we need real execution time for next mock setup steps
	// it will be corrected after
	earlyJob.ExecuteAt = clock.Now() // real execution time

	for i := 0; i < numberOfFullCycles; i++ {
		for _, originalJob := range jobs {
			currentJob := Job{
				Name:      originalJob.Name,
				ExecuteAt: originalJob.ExecuteAt.Add(originalJob.Period * time.Duration(i)),
				Period:    originalJob.Period,
			}

			newJob := &Job{
				Name:      originalJob.Name,
				ExecuteAt: currentJob.ExecuteAt.Add(originalJob.Period),
				Period:    originalJob.Period,
			}

			if len(executionSequence) == 0 || executionSequence[len(executionSequence)-1].time != currentJob.ExecuteAt {
				executionSequence = append(executionSequence, executionStep{
					time:         currentJob.ExecuteAt,
					jobs:         []Job{currentJob},
					initialTimer: i == 0,
				})
			} else {
				executionSequence[len(executionSequence)-1].jobs = append(executionSequence[len(executionSequence)-1].jobs, currentJob)
			}

			storageSession := &dummySession{}
			storage.EXPECT().NewSession(ctx).Return(storageSession, nil)
			storage.EXPECT().GetJobByName(ctx, storageSession, currentJob.Name).Return(&currentJob, nil)
			storage.EXPECT().UpsertJob(ctx, storageSession, newJob).Return(nil)
			storage.EXPECT().Commit(storageSession).Return(nil)
			storage.EXPECT().Abort(storageSession)
		}
	}

	// correct earlyJob.ExecuteAt
	earlyJob.ExecuteAt = clock.Now().Add(-1)

	// *** scheduler

	sch := New(zap.NewNop().Sugar(), storage, clock).(*scheduler)
	sch.RegisterJobHandler(earlyJob.Name, func() {
		earlyJobExecuted <- earlyJob.Name + " executed"
	})
	sch.RegisterJobHandler(nowJob.Name, func() {
		nowJobExecuted <- nowJob.Name + " executed"
	})
	sch.RegisterJobHandler(lateJob.Name, func() {
		lateJobExecuted <- lateJob.Name + " executed"
	})

	// *** test

	require.NoError(sch.Start(ctx))
	require.Len(sch.timers, len(jobs))

	for _, step := range executionSequence {
		jobNames := make([]string, len(step.jobs))
		for jobIndex, job := range step.jobs {
			jobNames[jobIndex] = job.Name
		}

		advanceDuration := step.time.Sub(clock.Now())
		clock.Advance(advanceDuration) // first execution step will advance time by 0
		require.Equal(step.time, clock.Now())

		jobsExecuteChans := make([]chan struct{}, len(step.jobs))
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		for jobIndex, job := range step.jobs {
			executeChan := make(chan struct{})
			jobsExecuteChans[jobIndex] = executeChan

			go func(jobName string) {
				select {
				case <-ctx.Done():
				case <-jobsExecChansMap[jobName]:
					executeChan <- struct{}{}
				}
				close(executeChan)
			}(job.Name)
		}

		for _, executeChan := range jobsExecuteChans {
			_, ok := <-executeChan
			require.True(ok, "some jobs weren't executed within timeout")
		}

		// if its first step for this timers, means that
		// those timers will be stopped after and replaced with tickers
		// we need to make sure that tickers are started before advancing time on next step
		if step.initialTimer {
			timersRearmChans := make([]chan struct{}, len(step.jobs))
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			for jobIndex, job := range step.jobs {
				rearmChan := make(chan struct{})
				timersRearmChans[jobIndex] = rearmChan

				// this starts goroutine with ticker that will periodically check if timer was replaced with ticker (rearmed)
				go func(jobName string) {
					ticker := time.NewTicker(epsilon)
					defer ticker.Stop()
					defer close(rearmChan)
					for {
						jobTimer, ok := sch.getJobTimer(jobName)
						require.True(ok)
						select {
						case <-ctx.Done():
							return
						case <-ticker.C:
							if _, ok := jobTimer.(clockwork.Ticker); ok {
								// sending signal, that timer was rearmed
								rearmChan <- struct{}{}
								return
							}
						}
					}
				}(job.Name)
			}

			// this gathers all rearm signals and checks that all timers were rearmed
			for _, rearmChan := range timersRearmChans {
				// [ok] true means that channel read returns actual written value
				// and not zero-value from closed channel
				// if its false, means that channel was closed before value was written, so timer wasn't rearmed within timeout
				_, ok := <-rearmChan
				require.True(ok, "some timers weren't rearmed within timeout")
			}
		}

		require.Equal(step.time, clock.Now())
	}

	require.NoError(sch.Stop())

	// checking, that all timers were stopped

	maxPeriod := time.Duration(0)
	for _, job := range jobs {
		if job.Period > maxPeriod {
			maxPeriod = job.Period
		}
	}

	clock.Advance(maxPeriod)

	selectCases := make([]reflect.SelectCase, len(jobs)+1)
	for i, job := range jobs {
		selectCases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(jobsExecChansMap[job.Name]),
		}
	}
	selectCases[len(jobs)] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(time.After(timeout)),
	}

	caseIndex, _, _ := reflect.Select(selectCases)
	require.Equal(len(jobs), caseIndex, "some jobs were executed after scheduler and job timers were stopped")
}

func TestScheduler_RegisterJobHandler(t *testing.T) {
	require := require.New(t)
	logger := zap.NewNop().Sugar()
	clock := clockwork.NewFakeClock()
	ctrl := gomock.NewController(t)
	storage := NewMockStorage(ctrl)
	jobExecuted := ""
	jobName1 := "job1"
	jobName2 := "job2"
	jobHandler1 := func() { jobExecuted = jobName1 }
	jobHandler2 := func() { jobExecuted = jobName2 }

	checkJobHandlerRegistered := func(sch *scheduler, jobName string) {
		t.Helper()
		require.Empty(jobExecuted)
		sch.registry[jobName]()
		require.Equal(jobName, jobExecuted)
		jobExecuted = ""
	}

	sch := New(logger, storage, clock).(*scheduler)

	require.Empty(sch.registry)

	// we cannot compare full scheduler structure, because it contains funcs map. Funcs cannot be compared with require.Equal
	// this can be changed in the future, if testify will support something like args for ignoring certain fields
	// so, we'll only check registered job handlers map by calling handlers

	sch.RegisterJobHandler(jobName1, jobHandler1)
	require.Len(sch.registry, 1)
	checkJobHandlerRegistered(sch, jobName1)

	sch.RegisterJobHandler(jobName2, jobHandler2)
	require.Len(sch.registry, 2)
	checkJobHandlerRegistered(sch, jobName1)
	checkJobHandlerRegistered(sch, jobName2)
}

func TestScheduler_Schedule(t *testing.T) {
	type testCase struct {
		storage     func(context.Context, *gomock.Controller, clockwork.Clock, *testCase) Storage
		existingJob *Job
		jobName     string
		period      time.Duration
		expectedErr error
	}

	tests := map[string]testCase{
		"OK: New job": {
			storage: func(ctx context.Context, ctrl *gomock.Controller, clock clockwork.Clock, tt *testCase) Storage {
				storage := NewMockStorage(ctrl)
				storageSession := &dummySession{}
				storage.EXPECT().NewSession(ctx).Return(storageSession, nil)
				storage.EXPECT().GetJobByName(ctx, storageSession, tt.jobName).Return(nil, ErrNotFound)
				storage.EXPECT().UpsertJob(ctx, storageSession, &Job{
					Name:      tt.jobName,
					ExecuteAt: clock.Now().Add(tt.period),
					Period:    tt.period,
				}).Return(nil)
				storage.EXPECT().Commit(storageSession).Return(nil)
				storage.EXPECT().Abort(storageSession)
				return storage
			},
			jobName: "new_job",
			period:  10 * time.Second,
		},
		"OK: Existing job": {
			storage: func(ctx context.Context, ctrl *gomock.Controller, _ clockwork.Clock, tt *testCase) Storage {
				storage := NewMockStorage(ctrl)
				storageSession := &dummySession{}
				storage.EXPECT().NewSession(ctx).Return(storageSession, nil)
				storage.EXPECT().GetJobByName(ctx, storageSession, tt.jobName).Return(tt.existingJob, nil)
				storage.EXPECT().UpsertJob(ctx, storageSession, &Job{
					Name:      tt.jobName,
					ExecuteAt: tt.existingJob.ExecuteAt,
					Period:    tt.period,
				}).Return(nil)
				storage.EXPECT().Commit(storageSession).Return(nil)
				storage.EXPECT().Abort(storageSession)
				return storage
			},
			existingJob: &Job{
				Name:      "existing_job",
				ExecuteAt: time.Now(),
				Period:    10 * time.Second,
			},
			jobName: "existing_job",
			period:  15 * time.Second,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			clock := clockwork.NewFakeClock()

			sch := New(
				zap.NewNop().Sugar(),
				tt.storage(ctx, gomock.NewController(t), clock, &tt),
				clock,
			).(*scheduler)

			err := sch.Schedule(ctx, tt.period, tt.jobName)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

type dummySession struct{}

func (d *dummySession) Commit() error {
	return nil
}

func (d *dummySession) Abort() error {
	return nil
}

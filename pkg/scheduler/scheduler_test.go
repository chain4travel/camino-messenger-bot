package scheduler

import (
	"context"
	"fmt"
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

	earlyJob := &Job{
		Name:      "early_job",
		ExecuteAt: clock.Now().Add(-1),
		Period:    13,
	}
	nowJob := &Job{
		Name:      "now_job",
		ExecuteAt: clock.Now(),
		Period:    17,
	}
	lateJob := &Job{
		Name:      "late_job",
		ExecuteAt: clock.Now().Add(1),
		Period:    19,
	}
	jobs := []*Job{earlyJob, nowJob, lateJob}
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
		time time.Time
		jobs []*Job
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
			currentJob := &Job{
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
					time: currentJob.ExecuteAt,
					jobs: []*Job{currentJob},
				})
			} else {
				executionSequence[len(executionSequence)-1].jobs = append(executionSequence[len(executionSequence)-1].jobs, currentJob)
			}

			fmt.Printf("Expecting update for %s (%d): %d -> %d\n",
				currentJob.Name, currentJob.Period, currentJob.ExecuteAt.UnixNano(), newJob.ExecuteAt.UnixNano())

			storageSession := &dummySession{}
			storage.EXPECT().NewSession(ctx).Return(storageSession, nil)
			storage.EXPECT().GetJobByName(ctx, storageSession, currentJob.Name).Return(&Job{ // we copy it, because execution will modify it
				Name:      currentJob.Name,
				ExecuteAt: currentJob.ExecuteAt,
				Period:    currentJob.Period,
			}, nil)
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
		fmt.Printf("%s (%d) executed at %d\n", earlyJob.Name, earlyJob.Period, clock.Now().UnixNano())
		earlyJobExecuted <- earlyJob.Name + " executed"
	})
	sch.RegisterJobHandler(nowJob.Name, func() {
		fmt.Printf("%s (%d) executed at %d\n", nowJob.Name, nowJob.Period, clock.Now().UnixNano())
		nowJobExecuted <- nowJob.Name + " executed"
	})
	sch.RegisterJobHandler(lateJob.Name, func() {
		fmt.Printf("%s (%d) executed at %d\n", lateJob.Name, lateJob.Period, clock.Now().UnixNano())
		lateJobExecuted <- lateJob.Name + " executed"
	})

	// *** test

	require.NoError(sch.Start(ctx))
	require.Len(sch.timers, len(jobs))

	for _, step := range executionSequence {
		fmt.Printf("Advance %d by %d", clock.Now().UnixNano(), step.time.Sub(clock.Now()))
		clock.Advance(step.time.Sub(clock.Now())) // first execution step will advance time by 0
		fmt.Printf(", now is %d\n", clock.Now().UnixNano())
		require.Equal(step.time, clock.Now())

		jobsExecChans := []chan string{}
		for _, job := range step.jobs {
			jobsExecChans = append(jobsExecChans, jobsExecChansMap[job.Name])
		}

		requireJobsToBeExecuted(t, clock.Now(), step.jobs, jobsExecChans, timeout)

		for _, job := range step.jobs {
			job.ExecuteAt = clock.Now().Add(job.Period)
		}

		require.Equal(step.time, clock.Now())
	}

	require.NoError(sch.Stop())
	time.Sleep(0)

	for _, timer := range sch.timers {
		require.True(timer.IsStopped())
	}
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

func requireJobsToBeExecuted(t *testing.T, execTime time.Time, jobs []*Job, executed []chan string, timeout time.Duration) { //nolint:unparam
	t.Helper()

	require.Len(t, jobs, len(executed))

	for i := 0; i < len(jobs); i++ {
		cases := make([]reflect.SelectCase, len(executed)+1)
		for i, ch := range executed {
			cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
		}
		cases[len(executed)] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(timeout))}

		if i, _, _ := reflect.Select(cases); i == len(executed) {
			require.FailNow(t, "some jobs wasn't executed within timeout")
		}
		require.Equal(t, execTime, jobs[i].ExecuteAt, "expected %s (%d) to be executed at %d, but got at %d",
			jobs[i].Name, jobs[i].Period, jobs[i].ExecuteAt.UnixNano(), execTime.UnixNano())
	}
}

// func requireJobsNotToBeExecuted(t *testing.T, jobs []*Job, executed []chan string, timeout time.Duration) {
// 	t.Helper()

// 	require.Len(t, jobs, len(executed))

// 	cases := make([]reflect.SelectCase, len(executed)+1)
// 	for i, ch := range executed {
// 		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
// 	}
// 	cases[len(executed)] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(timeout))}

// 	if i, _, _ := reflect.Select(cases); i < len(executed) {
// 		require.FailNowf(t, "job was executed", jobs[i].Name)
// 	}
// }

type dummySession struct{}

func (d *dummySession) Commit() error {
	return nil
}

func (d *dummySession) Abort() error {
	return nil
}

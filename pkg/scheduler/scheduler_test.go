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
	timeout := 10 * time.Millisecond
	numberOfFullCycles := 2

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
	jobsExecChans := []chan string{earlyJobExecuted, nowJobExecuted, lateJobExecuted}

	// this is needed for correct time-advancement sequence
	require.Less(earlyJob.Period, nowJob.Period)
	require.Less(nowJob.Period, lateJob.Period)
	require.Less(lateJob.Period, timeout-epsilon)

	// *** mock setup

	// main goroutine
	storageSession := &dummySession{}
	storage.EXPECT().NewSession(ctx).Return(storageSession, nil)
	storage.EXPECT().GetAllJobs(ctx, storageSession).Return(jobs, nil)
	storage.EXPECT().Abort(storageSession)

	// startOnce and periodic start goroutines

	// its clock.Now().Add(-1), but we need real execution time for next mock setup steps
	// it will be corrected after
	earlyJob.ExecuteAt = clock.Now() // real execution time

	for i := 0; i < numberOfFullCycles+1; i++ {
		for _, job := range jobs {
			expectUpdateJobExecutionTime(t, ctx, storage, job, job.ExecuteAt.Add(job.Period*time.Duration(i+1)))
		}
	}

	// correct earlyJob.ExecuteAt
	earlyJob.ExecuteAt = clock.Now().Add(-1)

	// *** scheduler

	sch, ok := New(zap.NewNop().Sugar(), storage, clock).(*scheduler)
	require.True(ok)
	sch.RegisterJobHandler(earlyJob.Name, func() {
		fmt.Printf("%s executed at %d\n", earlyJob.Name, clock.Now().UnixNano())
		earlyJobExecuted <- earlyJob.Name + " executed"
	})
	sch.RegisterJobHandler(nowJob.Name, func() {
		fmt.Printf("%s executed at %d\n", nowJob.Name, clock.Now().UnixNano())
		nowJobExecuted <- nowJob.Name + " executed"
	})
	sch.RegisterJobHandler(lateJob.Name, func() {
		fmt.Printf("%s executed at %d\n", lateJob.Name, clock.Now().UnixNano())
		lateJobExecuted <- lateJob.Name + " executed"
	})

	// *** test initial job execution

	fmt.Printf("Start scheduler\n")
	require.NoError(sch.Start(ctx))
	require.Len(sch.timers, len(jobs))

	// early and now jobs are executed, late job is not
	requireJobsNotToBeExecuted(t,
		[]*Job{lateJob},
		[]chan string{lateJobExecuted},
		timeout,
	)
	requireJobToBeExecuted(t, earlyJob.Name, earlyJobExecuted, timeout)
	requireJobToBeExecuted(t, nowJob.Name, nowJobExecuted, timeout)
	earlyJob.ExecuteAt = clock.Now().Add(earlyJob.Period)
	nowJob.ExecuteAt = clock.Now().Add(nowJob.Period)

	fmt.Printf("Advance %d by %d\n", clock.Now().UnixNano(), lateJob.ExecuteAt.Sub(clock.Now()))
	clock.Advance(lateJob.ExecuteAt.Sub(clock.Now()))
	fmt.Printf("clock.Now(): %d\n", clock.Now().UnixNano())

	// late job is executed, early and now jobs are not
	requireJobsNotToBeExecuted(t,
		[]*Job{earlyJob, nowJob},
		[]chan string{earlyJobExecuted, nowJobExecuted},
		timeout,
	)
	requireJobToBeExecuted(t, lateJob.Name, lateJobExecuted, timeout)
	lateJob.ExecuteAt = clock.Now().Add(lateJob.Period)

	// *** test job execution after period

	for i := 0; i < 2; i++ {
		for jobIndex, job := range jobs {
			fmt.Printf("Advance %d by %d\n", clock.Now().UnixNano(), job.ExecuteAt.Sub(clock.Now()))
			clock.Advance(job.ExecuteAt.Sub(clock.Now()))
			fmt.Printf("clock.Now(): %d\n", clock.Now().UnixNano())
			requireJobsNotToBeExecuted(t,
				excludeFromSlice(jobs, jobIndex),
				excludeFromSlice(jobsExecChans, jobIndex),
				timeout,
			)
			requireJobToBeExecuted(t, job.Name, jobsExecChans[jobIndex], timeout)
			job.ExecuteAt = clock.Now().Add(job.Period)
		}
	}

	require.NoError(sch.Stop())

	for _, timer := range sch.timers {
		<-timer.stopCh
	}
}

// func TestScheduler_Stop(t *testing.T) {
// 	logger := zap.NewNop().Sugar()
// 	clock := clockwork.NewFakeClock()
// 	ctrl := gomock.NewController(t)
// 	storage := NewMockStorage(ctrl)

// 	// storage := &mockStorage{
// 	// 	SessionHandler: &mockSessionHandler{},
// 	// 	jobs: []*Job{
// 	// 		{
// 	// 			Name:      "job1",
// 	// 			ExecuteAt: time.Now().Add(-time.Minute),
// 	// 			Period:    time.Minute,
// 	// 		},
// 	// 		{
// 	// 			Name:      "job2",
// 	// 			ExecuteAt: time.Now().Add(time.Minute),
// 	// 			Period:    time.Minute,
// 	// 		},
// 	// 	},
// 	// }

// 	s := New(logger, storage, clock)
// 	s.RegisterJobHandler("job1", func() {
// 		t.Log("job1 executed")
// 	})
// 	s.RegisterJobHandler("job2", func() {
// 		t.Log("job2 executed")
// 	})

// 	ctx := context.Background()
// 	err := s.Start(ctx)
// 	require.NoError(t, err)

// 	// Stop the scheduler
// 	err = s.Stop()
// 	require.NoError(t, err)

// 	// Check if all timers are stopped
// 	// s.(*scheduler).timersLock.RLock()
// 	// for _, timer := range s.(*scheduler).timers {
// 	// 	require.False(t, timer.IsRunning())
// 	// }
// 	// s.(*scheduler).timersLock.RUnlock()
// }
// func TestScheduler_Schedule(t *testing.T) {
// 	logger := zap.NewNop().Sugar()
// 	clock := clockwork.NewFakeClock()
// 	ctrl := gomock.NewController(t)
// 	storage := NewMockStorage(ctrl)

// 	// storage := &mockStorage{
// 	// 	SessionHandler: &mockSessionHandler{},
// 	// 	jobs: []*Job{
// 	// 		{
// 	// 			Name:      "job1",
// 	// 			ExecuteAt: time.Now().Add(time.Minute),
// 	// 			Period:    time.Minute,
// 	// 		},
// 	// 	},
// 	// }

// 	s := New(logger, storage, clock)
// 	s.RegisterJobHandler("job1", func() {
// 		t.Log("job1 executed")
// 	})

// 	ctx := context.Background()

// 	// Test scheduling a new job
// 	err := s.Schedule(ctx, time.Hour, "job2")
// 	require.NoError(t, err)

// 	// job, err := storage.GetJobByName(ctx, &mockSession{}, "job2")
// 	// require.NoError(t, err)
// 	// require.NotNil(t, job)
// 	// require.Equal(t, "job2", job.Name)
// 	// require.Equal(t, time.Hour, job.Period)

// 	// // Test updating an existing job
// 	// err = s.Schedule(ctx, time.Minute*30, "job1")
// 	// require.NoError(t, err)

// 	// job, err = storage.GetJobByName(ctx, &mockSession{}, "job1")
// 	// require.NoError(t, err)
// 	// require.NotNil(t, job)
// 	// require.Equal(t, "job1", job.Name)
// 	// require.Equal(t, time.Minute*30, job.Period)
// }

func requireJobToBeExecuted(t *testing.T, jobName string, executed chan string, timeout time.Duration) {
	t.Helper()
	select {
	case result := <-executed:
		require.Equal(t, jobName+" executed", result)
	case <-time.After(timeout):
		require.FailNowf(t, "job wasn't executed within timeout", jobName)
	}
}

func requireJobsNotToBeExecuted(t *testing.T, jobs []*Job, executed []chan string, timeout time.Duration) {
	t.Helper()

	cases := make([]reflect.SelectCase, len(executed)+1)
	for i, ch := range executed {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}
	cases[len(executed)] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(timeout))}

	i, v, c := reflect.Select(cases)
	if i < len(executed) {
		fmt.Printf("select {case i: %d, value: %s, closed: %t}\n", i, v.String(), c)
		require.FailNowf(t, "job was executed", jobs[i].Name)
	}
	tv := v.Interface().(time.Time)
	fmt.Printf("select {case i: %d, value: %d, closed: %t}\n", i, tv.UnixNano()/int64(time.Second), c)
}

func expectUpdateJobExecutionTime(t *testing.T, ctx context.Context, storage *MockStorage, job *Job, newExecuteAt time.Time) {
	t.Helper()
	storageSession := &dummySession{}

	storage.EXPECT().NewSession(ctx).Return(storageSession, nil)
	storage.EXPECT().GetJobByName(ctx, storageSession, job.Name).Return(job, nil)
	storage.EXPECT().UpsertJob(ctx, storageSession, &Job{
		Name:      job.Name,
		ExecuteAt: newExecuteAt,
		Period:    job.Period,
	}).Return(nil)
	storage.EXPECT().Commit(storageSession).Return(nil)
	storage.EXPECT().Abort(storageSession)
}

type dummySession struct{}

func (d *dummySession) Commit() error {
	return nil
}

func (d *dummySession) Abort() error {
	return nil
}

// will allocate new underlying array, so it will not affect the original slice
func excludeFromSlice[T any](slice []T, index int) []T {
	result := make([]T, 0, len(slice)-1)
	result = append(result, slice[:index]...)
	result = append(result, slice[index+1:]...)
	return result
}

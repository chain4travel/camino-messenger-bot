package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jonboulle/clockwork"
	"go.uber.org/zap"
)

var (
	_           Scheduler = (*scheduler)(nil)
	ErrNotFound           = errors.New("not found")
)

type Storage interface {
	SessionHandler

	GetAllJobs(ctx context.Context, session Session) ([]*Job, error)
	UpsertJob(ctx context.Context, session Session, job *Job) error
	GetJobByName(ctx context.Context, session Session, jobName string) (*Job, error)
}

type SessionHandler interface {
	NewSession(ctx context.Context) (Session, error)
	Commit(session Session) error
	Abort(session Session)
}

type Session interface {
	Commit() error
	Abort() error
}

type Scheduler interface {
	Start(ctx context.Context) error
	Stop() error
	Schedule(ctx context.Context, period time.Duration, jobName string) error
	RegisterJobHandler(jobName string, jobHandler func())
}

type Stopper interface {
	Stop()
}

func New(logger *zap.SugaredLogger, storage Storage, clock clockwork.Clock) Scheduler {
	return &scheduler{
		storage:  storage,
		logger:   logger,
		registry: make(map[string]func()),
		timers:   make(map[string]Stopper),
		clock:    clock,
	}
}

type scheduler struct {
	logger       *zap.SugaredLogger
	storage      Storage
	registry     map[string]func()
	timers       map[string]Stopper
	stopTimers   func()
	registryLock sync.RWMutex
	timersLock   sync.RWMutex
	clock        clockwork.Clock
}

// Start starts the scheduler. Jobs that are already due are executed immediately.
func (s *scheduler) Start(ctx context.Context) error {
	session, err := s.storage.NewSession(ctx)
	if err != nil {
		s.logger.Errorf("failed to create storage session: %v", err)
		return err
	}
	defer s.storage.Abort(session)

	jobs, err := s.storage.GetAllJobs(ctx, session)
	if err != nil {
		s.logger.Errorf("failed to get all jobs: %v", err)
		return err
	}

	timersCtx, cancel := context.WithCancel(ctx)
	s.stopTimers = cancel

	for _, job := range jobs {
		jobHandler, err := s.getJobHandler(job.Name)
		if err != nil {
			s.logger.Errorf("failed to get job handler: %v", err)
			return err
		}

		jobName := job.Name
		period := job.Period

		now := s.clock.Now()
		durationUntilFirstExecution := time.Duration(0)
		if job.ExecuteAt.After(now) {
			durationUntilFirstExecution = job.ExecuteAt.Sub(now)
		}

		onceDone := make(chan struct{})

		handler := func() {
			// TODO @evlekht panic handling?
			if err := s.updateJobExecutionTime(ctx, jobName); err != nil {
				s.logger.Errorf("failed to update job execution time: %v", err)
				return // TODO @evlekht handle error, maybe retry ?
			}
			jobHandler()
		}

		timer := s.clock.AfterFunc(durationUntilFirstExecution, func() {
			handler()
			close(onceDone)
		})
		s.setJobTimer(job.Name, &timerStopper{timer})

		go func() {
			<-onceDone
			ticker := s.clock.NewTicker(period)
			defer ticker.Stop()
			s.setJobTimer(job.Name, ticker)
			for {
				select {
				case <-ticker.Chan():
					handler()
				case <-timersCtx.Done():
					return
				}
			}
		}()
	}

	return nil
}

func (s *scheduler) Stop() error {
	s.stopTimers()
	s.timersLock.Lock()
	for jobName := range s.timers {
		delete(s.timers, jobName)
	}
	s.timersLock.Unlock()
	// TODO @evlekht await all ongoing job handlers to finish ?
	return nil
}

// Schedule schedules a job to be executed every period. If jobID is empty, a new job is created.
// Otherwise, the existing job period is updated and expiration time is set to min(current expiration time, now + period).
func (s *scheduler) Schedule(ctx context.Context, period time.Duration, jobName string) error {
	session, err := s.storage.NewSession(ctx)
	if err != nil {
		s.logger.Errorf("failed to create storage session: %v", err)
		return err
	}
	defer s.storage.Abort(session)

	job, err := s.storage.GetJobByName(ctx, session, jobName)
	if err != nil && !errors.Is(err, ErrNotFound) {
		s.logger.Errorf("failed to get job: %v", err)
		return err
	}

	executeAt := s.clock.Now().Add(period)

	if job != nil {
		job.Period = period
		if executeAt.Before(job.ExecuteAt) {
			job.ExecuteAt = executeAt
		}
	} else {
		job = &Job{
			Name:      jobName,
			ExecuteAt: executeAt,
			Period:    period,
		}
	}

	if err := s.storage.UpsertJob(ctx, session, job); err != nil {
		s.logger.Errorf("failed to store scheduled job: %v", err)
		return err
	}

	return s.storage.Commit(session)
}

func (s *scheduler) RegisterJobHandler(jobName string, jobHandler func()) {
	s.registryLock.Lock()
	s.registry[jobName] = jobHandler
	s.registryLock.Unlock()
}

func (s *scheduler) updateJobExecutionTime(ctx context.Context, jobName string) error {
	session, err := s.storage.NewSession(ctx)
	if err != nil {
		s.logger.Errorf("failed to create storage session: %v", err)
		return err
	}
	defer s.storage.Abort(session)

	job, err := s.storage.GetJobByName(ctx, session, jobName)
	if err != nil {
		s.logger.Errorf("failed to get job: %v", err)
		return err
	}

	job.ExecuteAt = s.clock.Now().Add(job.Period)

	if err := s.storage.UpsertJob(ctx, session, job); err != nil {
		s.logger.Errorf("failed to store scheduled job: %v", err)
		return err
	}

	if err := s.storage.Commit(session); err != nil {
		s.logger.Errorf("failed to commit session: %v", err)
		return err
	}

	return nil
}

func (s *scheduler) getJobHandler(jobName string) (func(), error) {
	s.registryLock.RLock()
	jobHandler, ok := s.registry[jobName]
	s.registryLock.RUnlock()
	if !ok {
		return nil, fmt.Errorf("job %s not found in registry", jobName)
	}
	return jobHandler, nil
}

func (s *scheduler) setJobTimer(jobName string, t Stopper) {
	s.timersLock.Lock()
	s.timers[jobName] = t
	s.timersLock.Unlock()
}

func (s *scheduler) getJobTimer(jobName string) (Stopper, bool) {
	s.timersLock.RLock()
	timer, ok := s.timers[jobName]
	s.timersLock.RUnlock()
	return timer, ok
}

type timerStopper struct {
	timer clockwork.Timer
}

func (t *timerStopper) Stop() {
	_ = t.timer.Stop()
}

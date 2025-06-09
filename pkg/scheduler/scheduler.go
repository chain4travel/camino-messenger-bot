// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

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
	Stop()
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
	wg           sync.WaitGroup
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

	safeGo := func(handler func()) {
		s.wg.Add(1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("scheduler panicked: %v", r)
					s.logger.Errorf("recovered from panic: %v", err)
				}
				s.wg.Done()
			}()
			handler()
		}()
	}

	for _, job := range jobs {
		jobHandler, err := s.getJobHandler(job.Name)
		if err != nil {
			s.logger.Errorf("failed to get job handler: %v", err)
			return err
		}

		jobName := job.Name
		period := job.Period

		now := s.clock.Now()
		durationUntilFirstExecution := max(job.LastExecutedAt.Add(job.Period).Sub(now), 0)

		handler := func(tickTime time.Time) {
			if err := s.updateJobExecutionTime(ctx, jobName, tickTime); err != nil {
				s.logger.Errorf("failed to update job execution time: %v", err)
				return
			}
			jobHandler()
		}

		// first execution
		onceDone := make(chan struct{})
		timer := s.clock.NewTimer(durationUntilFirstExecution)
		s.setJobTimer(job.Name, &timerStopper{timer})
		safeGo(func() {
			defer close(onceDone)
			select {
			case tickTime := <-timer.Chan():
				handler(tickTime)
			case <-timersCtx.Done():
			}
		})

		// periodic execution
		safeGo(func() {
			<-onceDone
			ticker := s.clock.NewTicker(period)
			defer ticker.Stop()
			s.setJobTimer(job.Name, ticker)
			for {
				select {
				case tickTime := <-ticker.Chan():
					handler(tickTime)
				case <-timersCtx.Done():
					return
				}
			}
		})
	}

	return nil
}

func (s *scheduler) Stop() {
	if s.stopTimers != nil { // if scheduler started correctly
		s.stopTimers()
	}
	s.wg.Wait()
	s.timersLock.Lock()
	for jobName := range s.timers {
		delete(s.timers, jobName)
	}
	s.timersLock.Unlock()
}

// Schedules a job to be executed every period.
// If there is already scheduled job with the same name, then its period is updated.
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

	lastExecutedAt := time.Time{}
	if job != nil {
		lastExecutedAt = job.LastExecutedAt
	}

	if err := s.storage.UpsertJob(ctx, session, &Job{
		Name:           jobName,
		LastExecutedAt: lastExecutedAt,
		Period:         period,
	}); err != nil {
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

func (s *scheduler) updateJobExecutionTime(ctx context.Context, jobName string, executionTime time.Time) error {
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

	job.LastExecutedAt = executionTime

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

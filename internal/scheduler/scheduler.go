package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/internal/models"
	"github.com/chain4travel/camino-messenger-bot/internal/storage"
	"go.uber.org/zap"
)

// TODO @evlekht its duplicate from asb, think of moving to common place
var _ Scheduler = (*scheduler)(nil)

type Scheduler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Schedule(ctx context.Context, period time.Duration, jobName string) error
	RegisterJobHandler(jobName string, jobHandler func())
}

func New(_ context.Context, logger *zap.SugaredLogger, storage storage.Storage) Scheduler {
	return &scheduler{
		storage:  storage,
		logger:   logger,
		registry: make(map[string]func()),
		timers:   make(map[string]*timer),
	}
}

type scheduler struct {
	logger       *zap.SugaredLogger
	storage      storage.Storage
	registry     map[string]func()
	timers       map[string]*timer
	registryLock sync.RWMutex
	timersLock   sync.RWMutex
}

// Start starts the scheduler. Jobs that are already due are executed immediately.
func (s *scheduler) Start(ctx context.Context) error {
	session, err := s.storage.NewSession(ctx)
	if err != nil {
		s.logger.Errorf("failed to create storage session: %v", err)
		return err
	}
	defer session.Abort()

	jobs, err := session.GetAllJobs(ctx)
	if err != nil {
		s.logger.Errorf("failed to get all jobs: %v", err)
		return err
	}

	for _, job := range jobs {
		jobHandler, err := s.getJobHandler(job.Name)
		if err != nil {
			s.logger.Errorf("failed to get job handler: %v", err)
			return err
		}

		jobName := job.Name
		period := job.Period

		now := time.Now()
		timeUntilFirstExecution := time.Duration(0)
		if job.ExecuteAt.After(now) {
			timeUntilFirstExecution = job.ExecuteAt.Sub(now)
		}

		handler := func() {
			// TODO @evlekht panic handling?
			if err := s.updateJobExecutionTime(ctx, jobName); err != nil {
				s.logger.Errorf("failed to update job execution time: %v", err)
				return // TODO @evlekht handle error, maybe retry ?
			}
			jobHandler()
		}

		timer := newTimer()
		doneCh := timer.StartOnce(timeUntilFirstExecution, handler)
		go func() {
			<-doneCh
			_ = timer.Start(period, handler)
		}()

		s.setJobTimer(job.Name, timer)
	}

	return nil
}

func (s *scheduler) Stop(_ context.Context) error {
	s.timersLock.RLock()
	for _, timer := range s.timers {
		timer.Stop()
	}
	s.timersLock.RUnlock()
	// TODO @evlekht await all ongoing job handlers to finish
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
	defer session.Abort()

	job, err := session.GetJobByName(ctx, jobName)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		s.logger.Errorf("failed to get job: %v", err)
		return err
	}

	executeAt := time.Now().Add(period)

	if job != nil {
		job.Period = period
		if executeAt.Before(job.ExecuteAt) {
			job.ExecuteAt = executeAt
		}
	} else {
		job = &models.Job{
			Name:      jobName,
			ExecuteAt: executeAt,
			Period:    period,
		}
	}

	if err := session.UpsertJob(ctx, job); err != nil {
		s.logger.Errorf("failed to store scheduled job: %v", err)
		return err
	}

	return session.Commit()
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
	defer session.Abort()

	job, err := session.GetJobByName(ctx, jobName)
	if err != nil {
		s.logger.Errorf("failed to get job: %v", err)
		return err
	}

	job.ExecuteAt = time.Now().Add(job.Period)

	if err := session.UpsertJob(ctx, job); err != nil {
		s.logger.Errorf("failed to store scheduled job: %v", err)
		return err
	}

	if err := session.Commit(); err != nil {
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

func (s *scheduler) setJobTimer(jobName string, t *timer) {
	s.timersLock.Lock()
	s.timers[jobName] = t
	s.timersLock.Unlock()
}

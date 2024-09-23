package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/chain4travel/camino-messenger-bot/internal/models"
	"github.com/jmoiron/sqlx"
)

const jobsTableName = "jobs"

var _ JobsStorage = (*session)(nil)

type JobsStorage interface {
	GetAllJobs(ctx context.Context) ([]*models.Job, error)
	UpsertJob(ctx context.Context, job *models.Job) error
	GetJobByName(ctx context.Context, jobName string) (*models.Job, error)
}

type job struct {
	Name      string `db:"name"`
	ExecuteAt int64  `db:"execute_at"`
	Period    int64  `db:"period"`
}

func (s *session) GetJobByName(ctx context.Context, jobName string) (*models.Job, error) {
	job := &job{}
	if err := s.tx.StmtxContext(ctx, s.storage.getJobByName).GetContext(ctx, job, jobName); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.logger.Error(err)
		}
		return nil, upgradeError(err)
	}
	return modelFromJob(job), nil
}

func (s *session) UpsertJob(ctx context.Context, job *models.Job) error {
	result, err := s.tx.NamedStmtContext(ctx, s.storage.upsertJob).
		ExecContext(ctx, jobFromModel(job))
	if err != nil {
		s.logger.Error(err)
		return upgradeError(err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		s.logger.Error(err)
		return upgradeError(err)
	} else if rowsAffected != 1 {
		return fmt.Errorf("failed to add cheque: expected to affect 1 row, but affected %d", rowsAffected)
	}
	return nil
}

func (s *session) GetAllJobs(ctx context.Context) ([]*models.Job, error) {
	jobs := []*models.Job{}
	rows, err := s.tx.StmtxContext(ctx, s.storage.getAllJobs).QueryxContext(ctx)
	if err != nil {
		s.logger.Error(err)
		return nil, upgradeError(err)
	}
	for rows.Next() {
		job := &job{}
		if err := rows.StructScan(job); err != nil {
			s.logger.Errorf("failed to get not cashed cheque from db: %v", err)
			continue
		}
		jobs = append(jobs, modelFromJob(job))
	}
	return jobs, nil
}

type jobsStatements struct {
	getAllJobs, getJobByName *sqlx.Stmt
	upsertJob                *sqlx.NamedStmt
}

func (s *storage) prepareJobsStmts(ctx context.Context) error {
	getJobByName, err := s.db.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE name = ?
	`, jobsTableName))
	if err != nil {
		s.logger.Error(err)
		return err
	}
	s.getJobByName = getJobByName

	upsertJob, err := s.db.PrepareNamedContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (
			name,
			execute_at,
			period
		) VALUES (
			:name,
			:execute_at,
			:period
		)
		ON CONFLICT(name)
		DO UPDATE SET period = excluded.period
	`, jobsTableName))
	if err != nil {
		s.logger.Error(err)
		return err
	}
	s.upsertJob = upsertJob

	getAllJobs, err := s.db.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
	`, jobsTableName))
	if err != nil {
		s.logger.Error(err)
		return err
	}
	s.getAllJobs = getAllJobs

	return nil
}

func modelFromJob(job *job) *models.Job {
	return &models.Job{
		Name:      job.Name,
		ExecuteAt: time.Unix(job.ExecuteAt, 0),
		Period:    time.Duration(job.Period) * time.Second,
	}
}

func jobFromModel(model *models.Job) *job {
	return &job{
		Name:      model.Name,
		ExecuteAt: model.ExecuteAt.Unix(),
		Period:    int64(model.Period / time.Second),
	}
}

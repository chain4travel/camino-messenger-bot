package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/chain4travel/camino-messenger-bot/pkg/scheduler"
	"github.com/jmoiron/sqlx"
)

const jobsTableName = "jobs"

var _ scheduler.Storage = (*storage)(nil)

type job struct {
	Name      string `db:"name"`
	ExecuteAt int64  `db:"execute_at"`
	Period    int64  `db:"period"`
}

func (s *storage) GetJobByName(ctx context.Context, session scheduler.Session, jobName string) (*scheduler.Job, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.baseDB.Logger.Error(err)
		return nil, err
	}

	job := &job{}
	if err := tx.StmtxContext(ctx, s.getJobByName).GetContext(ctx, job, jobName); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.baseDB.Logger.Error(err)
		}
		return nil, upgradeError(err)
	}
	return modelFromJob(job), nil
}

func (s *storage) UpsertJob(ctx context.Context, session scheduler.Session, job *scheduler.Job) error {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.baseDB.Logger.Error(err)
		return err
	}

	result, err := tx.NamedStmtContext(ctx, s.upsertJob).
		ExecContext(ctx, jobFromModel(job))
	if err != nil {
		s.baseDB.Logger.Error(err)
		return upgradeError(err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		s.baseDB.Logger.Error(err)
		return upgradeError(err)
	} else if rowsAffected != 1 {
		return fmt.Errorf("failed to add cheque: expected to affect 1 row, but affected %d", rowsAffected)
	}
	return nil
}

func (s *storage) GetAllJobs(ctx context.Context, session scheduler.Session) ([]*scheduler.Job, error) {
	tx, err := getSQLXTx(session)
	if err != nil {
		s.baseDB.Logger.Error(err)
		return nil, err
	}

	jobs := []*scheduler.Job{}
	rows, err := tx.StmtxContext(ctx, s.getAllJobs).QueryxContext(ctx)
	if err != nil {
		s.baseDB.Logger.Error(err)
		return nil, upgradeError(err)
	}
	for rows.Next() {
		job := &job{}
		if err := rows.StructScan(job); err != nil {
			s.baseDB.Logger.Errorf("failed to get not cashed cheque from db: %v", err)
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
	getJobByName, err := s.baseDB.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE name = ?
	`, jobsTableName))
	if err != nil {
		s.baseDB.Logger.Error(err)
		return err
	}
	s.getJobByName = getJobByName

	upsertJob, err := s.baseDB.DB.PrepareNamedContext(ctx, fmt.Sprintf(`
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
		s.baseDB.Logger.Error(err)
		return err
	}
	s.upsertJob = upsertJob

	getAllJobs, err := s.baseDB.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
	`, jobsTableName))
	if err != nil {
		s.baseDB.Logger.Error(err)
		return err
	}
	s.getAllJobs = getAllJobs

	return nil
}

func modelFromJob(job *job) *scheduler.Job {
	return &scheduler.Job{
		Name:      job.Name,
		ExecuteAt: time.Unix(job.ExecuteAt, 0),
		Period:    time.Duration(job.Period) * time.Second,
	}
}

func jobFromModel(model *scheduler.Job) *job {
	return &job{
		Name:      model.Name,
		ExecuteAt: model.ExecuteAt.Unix(),
		Period:    int64(model.Period / time.Second),
	}
}

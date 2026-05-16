// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v13/pkg/database/sqlite"
	"github.com/chain4travel/camino-messenger-bot/v13/pkg/scheduler"
	"github.com/jmoiron/sqlx"
)

const jobsTableName = "jobs"

var _ scheduler.Storage = (*storage)(nil)

type job struct {
	Name           string `db:"name"`
	LastExecutedAt int64  `db:"last_executed_at"`
	Period         int64  `db:"period"`
}

func (s *storage) GetJobByName(ctx context.Context, session scheduler.Session, jobName string) (*scheduler.Job, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	job := &job{}
	if err := tx.StmtxContext(ctx, s.getJobByName).GetContext(ctx, job, jobName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, scheduler.ErrNotFound
		}
		return nil, fmt.Errorf("failed to execute get job by name statement: %w", err)
	}
	return modelFromJob(job), nil
}

func (s *storage) UpsertJob(ctx context.Context, session scheduler.Session, job *scheduler.Job) error {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	result, err := tx.NamedStmtContext(ctx, s.upsertJob).
		ExecContext(ctx, jobFromModel(job))
	if err != nil {
		return fmt.Errorf("failed to execute upsert job statement: %w", err)
	}
	if rowsAffected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("failed to get rowsAffected from statement execution result: %w", err)
	} else if rowsAffected != 1 {
		return fmt.Errorf("unexpected number of rows affected: expected 1, but affected %d", rowsAffected)
	}
	return nil
}

func (s *storage) GetAllJobs(ctx context.Context, session scheduler.Session) ([]*scheduler.Job, error) {
	tx, err := sqlite.GetSQLXTx(session)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from db session: %w", err)
	}

	jobs := []*scheduler.Job{}
	rows, err := tx.StmtxContext(ctx, s.getAllJobs).QueryxContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get all jobs statement: %w", err)
	}
	for rows.Next() {
		job := &job{}
		if err := rows.StructScan(job); err != nil {
			s.base.Logger.Errorf("failed to scan row to job: %v", err)
			continue
		}
		jobs = append(jobs, modelFromJob(job))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred during rows iteration: %w", err)
	}
	return jobs, nil
}

type jobsStatements struct {
	getAllJobs, getJobByName *sqlx.Stmt
	upsertJob                *sqlx.NamedStmt
}

func (s *storage) prepareJobsStmts(ctx context.Context) error {
	getJobByName, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
		WHERE name = ?
	`, jobsTableName))
	if err != nil {
		return fmt.Errorf("failed to prepare get job by name statement: %w", err)
	}
	s.getJobByName = getJobByName

	upsertJob, err := s.base.DB.PrepareNamedContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (
			name,
			last_executed_at,
			period
		) VALUES (
			:name,
			:last_executed_at,
			:period
		)
		ON CONFLICT(name)
		DO UPDATE SET
			period = excluded.period,
			last_executed_at = excluded.last_executed_at
	`, jobsTableName))
	if err != nil {
		return fmt.Errorf("failed to prepare upsert job statement: %w", err)
	}
	s.upsertJob = upsertJob

	getAllJobs, err := s.base.DB.PreparexContext(ctx, fmt.Sprintf(`
		SELECT * FROM %s
	`, jobsTableName))
	if err != nil {
		return fmt.Errorf("failed to prepare get all jobs statement: %w", err)
	}
	s.getAllJobs = getAllJobs

	return nil
}

func modelFromJob(job *job) *scheduler.Job {
	return &scheduler.Job{
		Name:           job.Name,
		LastExecutedAt: time.Unix(job.LastExecutedAt, 0),
		Period:         time.Duration(job.Period) * time.Second,
	}
}

func jobFromModel(model *scheduler.Job) *job {
	return &job{
		Name:           model.Name,
		LastExecutedAt: model.LastExecutedAt.Unix(),
		Period:         int64(model.Period / time.Second),
	}
}

// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.19.1
// source: query.sql

package models

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const getWorkerPools = `-- name: GetWorkerPools :many
SELECT id, queue, workers, git_hash, job_types, created_at, updated_at
FROM arrower.gue_jobs_worker_pool
WHERE updated_at > NOW() - INTERVAL '2 minutes'
ORDER BY queue, id
`

func (q *Queries) GetWorkerPools(ctx context.Context) ([]ArrowerGueJobsWorkerPool, error) {
	rows, err := q.db.Query(ctx, getWorkerPools)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ArrowerGueJobsWorkerPool
	for rows.Next() {
		var i ArrowerGueJobsWorkerPool
		if err := rows.Scan(
			&i.ID,
			&i.Queue,
			&i.Workers,
			&i.GitHash,
			&i.JobTypes,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const insertHistory = `-- name: InsertHistory :exec
INSERT INTO arrower.gue_jobs_history (job_id, priority, run_at, job_type, args, run_count, run_error, queue, created_at,
                                      updated_at, success, finished_at)
VALUES ($1, $2, $3, $4, $5, $6, $8::text, $7, STATEMENT_TIMESTAMP(), STATEMENT_TIMESTAMP(), FALSE,
        NULL)
`

type InsertHistoryParams struct {
	JobID    string
	Priority int16
	RunAt    pgtype.Timestamptz
	JobType  string
	Args     []byte
	RunCount int32
	Queue    string
	RunError string
}

func (q *Queries) InsertHistory(ctx context.Context, arg InsertHistoryParams) error {
	_, err := q.db.Exec(ctx, insertHistory,
		arg.JobID,
		arg.Priority,
		arg.RunAt,
		arg.JobType,
		arg.Args,
		arg.RunCount,
		arg.Queue,
		arg.RunError,
	)
	return err
}

const updateHistory = `-- name: UpdateHistory :exec
UPDATE arrower.gue_jobs_history
SET run_error   = $3::text,
    finished_at = STATEMENT_TIMESTAMP(), -- now() or CURRENT_TIMESTAMP record the start of the transaction, this is more precise in case of a long running job.
    run_count   = $4,
    success     = $1
WHERE job_id = $2
  AND run_count = $4
  AND finished_at IS NULL
`

type UpdateHistoryParams struct {
	Success  bool
	JobID    string
	RunError string
	RunCount int32
}

func (q *Queries) UpdateHistory(ctx context.Context, arg UpdateHistoryParams) error {
	_, err := q.db.Exec(ctx, updateHistory,
		arg.Success,
		arg.JobID,
		arg.RunError,
		arg.RunCount,
	)
	return err
}

const upsertSchedule = `-- name: UpsertSchedule :exec
INSERT INTO arrower.gue_jobs_schedule (queue, spec, job_type, args, created_at, updated_at)
VALUES($1, $2, $3, $4, NOW(), $5)
ON CONFLICT (queue, spec, job_type, args) DO UPDATE SET updated_at = NOW()
`

type UpsertScheduleParams struct {
	Queue     string
	Spec      string
	JobType   string
	Args      []byte
	UpdatedAt pgtype.Timestamptz
}

func (q *Queries) UpsertSchedule(ctx context.Context, arg UpsertScheduleParams) error {
	_, err := q.db.Exec(ctx, upsertSchedule,
		arg.Queue,
		arg.Spec,
		arg.JobType,
		arg.Args,
		arg.UpdatedAt,
	)
	return err
}

const upsertWorkerToPool = `-- name: UpsertWorkerToPool :exec
INSERT INTO arrower.gue_jobs_worker_pool (id, queue, workers, git_hash, job_types, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW(), $6)
ON CONFLICT (id, queue) DO UPDATE SET updated_at = NOW(),
                                      workers    = $3,
                                      git_hash   = $4,
                                      job_types  = $5
`

type UpsertWorkerToPoolParams struct {
	ID        string
	Queue     string
	Workers   int16
	GitHash   string
	JobTypes  []string
	UpdatedAt pgtype.Timestamptz
}

func (q *Queries) UpsertWorkerToPool(ctx context.Context, arg UpsertWorkerToPoolParams) error {
	_, err := q.db.Exec(ctx, upsertWorkerToPool,
		arg.ID,
		arg.Queue,
		arg.Workers,
		arg.GitHash,
		arg.JobTypes,
		arg.UpdatedAt,
	)
	return err
}

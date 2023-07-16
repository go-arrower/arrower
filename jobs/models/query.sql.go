// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.17.2
// source: query.sql

package models

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const deleteJob = `-- name: DeleteJob :exec
DELETE FROM public.gue_jobs WHERE job_id = $1
`

func (q *Queries) DeleteJob(ctx context.Context, jobID string) error {
	_, err := q.db.Exec(ctx, deleteJob, jobID)
	return err
}

const getPendingJobs = `-- name: GetPendingJobs :many
SELECT job_id, priority, run_at, job_type, args, error_count, last_error, queue, created_at, updated_at FROM public.gue_jobs WHERE queue = $1 ORDER BY priority, run_at ASC LIMIT 100
`

func (q *Queries) GetPendingJobs(ctx context.Context, queue string) ([]GueJob, error) {
	rows, err := q.db.Query(ctx, getPendingJobs, queue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GueJob
	for rows.Next() {
		var i GueJob
		if err := rows.Scan(
			&i.JobID,
			&i.Priority,
			&i.RunAt,
			&i.JobType,
			&i.Args,
			&i.ErrorCount,
			&i.LastError,
			&i.Queue,
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

const getQueues = `-- name: GetQueues :many
SELECT queue FROM public.gue_jobs
UNION
SELECT queue FROM public.gue_jobs_history
`

func (q *Queries) GetQueues(ctx context.Context) ([]string, error) {
	rows, err := q.db.Query(ctx, getQueues)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var queue string
		if err := rows.Scan(&queue); err != nil {
			return nil, err
		}
		items = append(items, queue)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getWorkerPools = `-- name: GetWorkerPools :many
SELECT id, queue, workers, created_at, updated_at FROM public.gue_jobs_worker_pool WHERE updated_at > NOW() - INTERVAL '2 minutes' ORDER BY queue, id
`

func (q *Queries) GetWorkerPools(ctx context.Context) ([]GueJobsWorkerPool, error) {
	rows, err := q.db.Query(ctx, getWorkerPools)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GueJobsWorkerPool
	for rows.Next() {
		var i GueJobsWorkerPool
		if err := rows.Scan(
			&i.ID,
			&i.Queue,
			&i.Workers,
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

const statsAvgDurationOfJobs = `-- name: StatsAvgDurationOfJobs :one
SELECT AVG(EXTRACT(MICROSECONDS FROM (finished_at - created_at))) AS durration_in_microseconds FROM public.gue_jobs_history WHERE queue = $1
`

func (q *Queries) StatsAvgDurationOfJobs(ctx context.Context, queue string) (float64, error) {
	row := q.db.QueryRow(ctx, statsAvgDurationOfJobs, queue)
	var durration_in_microseconds float64
	err := row.Scan(&durration_in_microseconds)
	return durration_in_microseconds, err
}

const statsFailedJobs = `-- name: StatsFailedJobs :one
SELECT COUNT(*) FROM public.gue_jobs WHERE queue = $1 AND error_count <> 0
`

func (q *Queries) StatsFailedJobs(ctx context.Context, queue string) (int64, error) {
	row := q.db.QueryRow(ctx, statsFailedJobs, queue)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const statsPendingJobs = `-- name: StatsPendingJobs :one
SELECT COUNT(*) FROM public.gue_jobs WHERE queue = $1
`

func (q *Queries) StatsPendingJobs(ctx context.Context, queue string) (int64, error) {
	row := q.db.QueryRow(ctx, statsPendingJobs, queue)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const statsPendingJobsPerType = `-- name: StatsPendingJobsPerType :many
SELECT job_type, COUNT(*) as count FROM public.gue_jobs WHERE queue = $1 GROUP BY job_type
`

type StatsPendingJobsPerTypeRow struct {
	JobType string
	Count   int64
}

func (q *Queries) StatsPendingJobsPerType(ctx context.Context, queue string) ([]StatsPendingJobsPerTypeRow, error) {
	rows, err := q.db.Query(ctx, statsPendingJobsPerType, queue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []StatsPendingJobsPerTypeRow
	for rows.Next() {
		var i StatsPendingJobsPerTypeRow
		if err := rows.Scan(&i.JobType, &i.Count); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const statsProcessedJobs = `-- name: StatsProcessedJobs :one
SELECT COUNT(*) FROM public.gue_jobs_history WHERE queue = $1
`

func (q *Queries) StatsProcessedJobs(ctx context.Context, queue string) (int64, error) {
	row := q.db.QueryRow(ctx, statsProcessedJobs, queue)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const statsQueueWorkerPoolSize = `-- name: StatsQueueWorkerPoolSize :one
SELECT COALESCE(SUM(workers),0)::INTEGER FROM public.gue_jobs_worker_pool WHERE queue = $1 AND updated_at > NOW() - INTERVAL '1 minutes'
`

func (q *Queries) StatsQueueWorkerPoolSize(ctx context.Context, queue string) (int32, error) {
	row := q.db.QueryRow(ctx, statsQueueWorkerPoolSize, queue)
	var column_1 int32
	err := row.Scan(&column_1)
	return column_1, err
}

const updateRunAt = `-- name: UpdateRunAt :exec
UPDATE public.gue_jobs SET run_at = $1 WHERE job_id = $2
`

type UpdateRunAtParams struct {
	RunAt pgtype.Timestamptz
	JobID string
}

func (q *Queries) UpdateRunAt(ctx context.Context, arg UpdateRunAtParams) error {
	_, err := q.db.Exec(ctx, updateRunAt, arg.RunAt, arg.JobID)
	return err
}

const upsertWorkerToPool = `-- name: UpsertWorkerToPool :exec
INSERT INTO public.gue_jobs_worker_pool (id, queue, workers, created_at, updated_at)
    VALUES($1, $2, $3, STATEMENT_TIMESTAMP(), $4)
ON CONFLICT (id) DO
    UPDATE SET updated_at = STATEMENT_TIMESTAMP(), workers = $3
`

type UpsertWorkerToPoolParams struct {
	ID        string
	Queue     string
	Workers   int16
	UpdatedAt pgtype.Timestamptz
}

func (q *Queries) UpsertWorkerToPool(ctx context.Context, arg UpsertWorkerToPoolParams) error {
	_, err := q.db.Exec(ctx, upsertWorkerToPool,
		arg.ID,
		arg.Queue,
		arg.Workers,
		arg.UpdatedAt,
	)
	return err
}

-- name: PendingJobs :many
SELECT bins.*, COUNT(t)
FROM (SELECT date_bin($1, finished_at, TIMESTAMP WITH TIME ZONE'2001-01-01')::TIMESTAMPTZ as t
      FROM arrower.gue_jobs_history
      WHERE finished_at > $2) bins
GROUP BY bins.t
ORDER BY bins.t DESC
LIMIT $3;

-- name: JobTypes :many
SELECT DISTINCT(job_type)
FROM arrower.gue_jobs_history
WHERE queue = $1;

-- name: ScheduleJobs :copyfrom
INSERT INTO arrower.gue_jobs (job_id, created_at, updated_at, queue, job_type, priority, run_at, args)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: JobTableSize :one
SELECT pg_size_pretty(pg_total_relation_size('arrower.gue_jobs'))         as jobs,
       pg_size_pretty(pg_total_relation_size('arrower.gue_jobs_history')) as history;

-- name: JobHistorySize :one
SELECT COALESCE(pg_size_pretty(SUM(pg_column_size(arrower.gue_jobs_history.*))), '')
FROM arrower.gue_jobs_history
WHERE created_at <= $1;

-- name: PruneHistory :exec
DELETE
FROM arrower.gue_jobs_history
WHERE created_at <= $1;

-- name: JobHistoryPayloadSize :one
SELECT COALESCE(pg_size_pretty(SUM(pg_column_size(arrower.gue_jobs_history.args))), '')
FROM arrower.gue_jobs_history
WHERE queue = $1
  AND created_at <= $2
  AND args <> '';

-- name: PruneHistoryPayload :exec
UPDATE arrower.gue_jobs_history
SET args      = ''::BYTEA,
    pruned_at = NOW()
WHERE queue = $1
  AND created_at <= $2;


-- name: LastHistoryPayloads :many
SELECT args
FROM arrower.gue_jobs_history
WHERE queue = $1
  AND job_type = $2
ORDER BY created_at DESC
LIMIT 5;


-- name: GetQueues :many
SELECT queue
FROM arrower.gue_jobs
UNION
SELECT queue
FROM arrower.gue_jobs_history;


-- name: GetPendingJobs :many
SELECT *
FROM arrower.gue_jobs
WHERE queue = $1
ORDER BY run_at,priority ASC
LIMIT 100;

-- name: GetFinishedJobs :many
SELECT f.*
FROM (SELECT DISTINCT ON (job_id) *
      FROM arrower.gue_jobs_history
      WHERE finished_at IS NOT NULL
      LIMIT 100) as f
ORDER BY f.finished_at DESC;

-- name: GetFinishedJobsByQueue :many
SELECT f.*
FROM (SELECT DISTINCT ON (job_id) *
      FROM arrower.gue_jobs_history
      WHERE finished_at IS NOT NULL
        AND queue = $1
      LIMIT 100) as f
ORDER BY f.finished_at DESC;

-- name: GetFinishedJobsByQueueAndType :many
SELECT f.*
FROM (SELECT DISTINCT ON (job_id) *
      FROM arrower.gue_jobs_history
      WHERE finished_at IS NOT NULL
        AND queue = $1
        AND job_type = $2
      LIMIT 100) as f
ORDER BY f.finished_at DESC;

-- name: DeleteJob :exec
DELETE
FROM arrower.gue_jobs
WHERE job_id = $1;

-- name: UpdateRunAt :exec
UPDATE arrower.gue_jobs
SET run_at = $1
WHERE job_id = $2;


-- name: StatsPendingJobs :one
SELECT COUNT(*)
FROM arrower.gue_jobs
WHERE queue = $1;

-- name: StatsFailedJobs :one
SELECT COUNT(*)
FROM arrower.gue_jobs
WHERE queue = $1
  AND error_count <> 0;

-- name: StatsAvgDurationOfJobs :one
SELECT COALESCE(AVG(EXTRACT(MICROSECONDS FROM (finished_at - created_at))), 0)::FLOAT AS durration_in_microseconds
FROM arrower.gue_jobs_history
WHERE queue = $1;

-- name: StatsPendingJobsPerType :many
SELECT job_type, COUNT(*) as count
FROM arrower.gue_jobs
WHERE queue = $1
GROUP BY job_type;

-- name: StatsProcessedJobs :one
SELECT COUNT(DISTINCT job_id)
FROM arrower.gue_jobs_history
WHERE queue = $1
  AND success = true;

-- name: StatsQueueWorkerPoolSize :one
SELECT COALESCE(SUM(workers), 0)::INTEGER
FROM arrower.gue_jobs_worker_pool
WHERE queue = $1
  AND updated_at > NOW() - INTERVAL '1 minutes';


-- name: GetWorkerPools :many
SELECT *
FROM arrower.gue_jobs_worker_pool
WHERE updated_at > NOW() - INTERVAL '2 minutes'
ORDER BY queue, id;

-- name: UpsertWorkerToPool :exec
INSERT INTO arrower.gue_jobs_worker_pool (id, queue, workers, created_at, updated_at)
VALUES ($1, $2, $3, STATEMENT_TIMESTAMP(), $4)
ON CONFLICT (id, queue) DO UPDATE SET updated_at = STATEMENT_TIMESTAMP(),
                                      workers    = $3;

-- name: TotalFinishedJobs :one
SELECT COUNT(DISTINCT (job_id))
FROM arrower.gue_jobs_history
WHERE finished_at IS NOT NULL;

-- name: TotalFinishedJobsByQueue :one
SELECT COUNT(DISTINCT (job_id))
FROM arrower.gue_jobs_history
WHERE queue = $1
  AND finished_at IS NOT NULL;

-- name: TotalFinishedJobsByQueueAndType :one
SELECT COUNT(DISTINCT (job_id))
FROM arrower.gue_jobs_history
WHERE queue = $1
  AND job_type = $2
  AND finished_at IS NOT NULL;

-- name: GetJobHistory :many
SELECT *
FROM arrower.gue_jobs_history
WHERE job_id = $1
ORDER BY created_at DESC;
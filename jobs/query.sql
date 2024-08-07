-- name: GetWorkerPools :many
SELECT *
FROM arrower.gue_jobs_worker_pool
WHERE updated_at > NOW() - INTERVAL '2 minutes'
ORDER BY queue, id;

-- name: UpsertWorkerToPool :exec
INSERT INTO arrower.gue_jobs_worker_pool (id, queue, workers, git_hash, job_types, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW(), $6)
ON CONFLICT (id, queue) DO UPDATE SET updated_at = NOW(),
                                      workers    = $3,
                                      git_hash   = $4,
                                      job_types  = $5;

-- name: UpsertSchedule :exec
INSERT INTO arrower.gue_jobs_schedule (queue, spec, job_type, args, created_at, updated_at)
VALUES($1, $2, $3, $4, NOW(), $5)
ON CONFLICT (queue, spec, job_type, args) DO UPDATE SET updated_at = NOW();

-- name: InsertHistory :exec
INSERT INTO arrower.gue_jobs_history (job_id, priority, run_at, job_type, args, run_count, run_error, queue, created_at,
                                      updated_at, success, finished_at)
VALUES ($1, $2, $3, $4, $5, $6, sqlc.arg(run_error)::text, $7, STATEMENT_TIMESTAMP(), STATEMENT_TIMESTAMP(), FALSE,
        NULL);

-- name: UpdateHistory :exec
UPDATE arrower.gue_jobs_history
SET run_error   = sqlc.arg(run_error)::text,
    finished_at = STATEMENT_TIMESTAMP(), -- now() or CURRENT_TIMESTAMP record the start of the transaction, this is more precise in case of a long running job.
    run_count   = sqlc.arg(run_count),
    success     = $1
WHERE job_id = $2
  AND run_count = sqlc.arg(run_count)
  AND finished_at IS NULL;

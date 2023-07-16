-- name: GetQueues :many
SELECT queue FROM public.gue_jobs
UNION
SELECT queue FROM public.gue_jobs_history;


-- name: GetPendingJobs :many
SELECT * FROM public.gue_jobs WHERE queue = $1 ORDER BY priority, run_at ASC LIMIT 100;

-- name: DeleteJob :exec
DELETE FROM public.gue_jobs WHERE job_id = $1;

-- name: UpdateRunAt :exec
UPDATE public.gue_jobs SET run_at = $1 WHERE job_id = $2;


-- name: StatsPendingJobs :one
SELECT COUNT(*) FROM public.gue_jobs WHERE queue = $1;

-- name: StatsFailedJobs :one
SELECT COUNT(*) FROM public.gue_jobs WHERE queue = $1 AND error_count <> 0;

-- name: StatsAvgDurationOfJobs :one
SELECT AVG(EXTRACT(MICROSECONDS FROM (finished_at - created_at))) AS durration_in_microseconds FROM public.gue_jobs_history WHERE queue = $1;

-- name: StatsPendingJobsPerType :many
SELECT job_type, COUNT(*) as count FROM public.gue_jobs WHERE queue = $1 GROUP BY job_type;

-- name: StatsProcessedJobs :one
SELECT COUNT(*) FROM public.gue_jobs_history WHERE queue = $1;

-- name: StatsQueueWorkerPoolSize :one
SELECT COALESCE(SUM(workers),0)::INTEGER FROM public.gue_jobs_worker_pool WHERE queue = $1 AND updated_at > NOW() - INTERVAL '1 minutes';


-- name: GetWorkerPools :many
SELECT * FROM public.gue_jobs_worker_pool WHERE updated_at > NOW() - INTERVAL '2 minutes' ORDER BY queue, id;

-- name: UpsertWorkerToPool :exec
INSERT INTO public.gue_jobs_worker_pool (id, queue, workers, created_at, updated_at)
    VALUES($1, $2, $3, STATEMENT_TIMESTAMP(), $4)
ON CONFLICT (id) DO
    UPDATE SET updated_at = STATEMENT_TIMESTAMP(), workers = $3;
-- name: GetQueues :many
SELECT queue FROM public.gue_jobs
UNION
SELECT queue FROM public.gue_jobs_history;


-- name: GetPendingJobs :many
SELECT * FROM public.gue_jobs WHERE queue = $1 ORDER BY priority, run_at ASC LIMIT 100;


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
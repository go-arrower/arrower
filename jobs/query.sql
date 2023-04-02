-- name: GetQueues :many
SELECT queue FROM public.gue_jobs
UNION
SELECT queue FROM public.gue_jobs_history;

-- name: GetPendingJobs :many
SELECT * FROM public.gue_jobs WHERE queue = $1 ORDER BY priority, run_at ASC LIMIT 100;
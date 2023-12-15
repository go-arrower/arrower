-- name: GetWorkerPools :many
SELECT * FROM public.gue_jobs_worker_pool WHERE updated_at > NOW() - INTERVAL '2 minutes' ORDER BY queue, id;

-- name: UpsertWorkerToPool :exec
INSERT INTO public.gue_jobs_worker_pool (id, queue, workers, created_at, updated_at)
    VALUES($1, $2, $3, STATEMENT_TIMESTAMP(), $4)
ON CONFLICT (id, queue) DO
    UPDATE SET updated_at = STATEMENT_TIMESTAMP(), workers = $3;
-- name: LogRecords :copyfrom
INSERT INTO public.log (time, user_id, log)
VALUES ($1, $2, $3);

-- name: GetRecentLogs :many
SELECT *
FROM public.log
WHERE time > $1
  AND log ->> 'msg' ILIKE @msg::TEXT -- cast to make Go code easier to use
  AND (CASE WHEN cardinality(@level::TEXT[]) <> 0 THEN log->>'level' = ANY(@level::TEXT[]) ELSE TRUE END)
  --AND (CASE WHEN @k0::TEXT <> '' THEN log->>(@k0) ILIKE @f0::TEXT ELSE TRUE END)
  AND (CASE WHEN @f0::TEXT <> '' THEN jsonb_path_exists(log, @f0::JSONPATH) ELSE TRUE END) -- only use this filter, if not empty
  AND (CASE WHEN @f1::TEXT <> '' THEN jsonb_path_exists(log, @f1::JSONPATH) ELSE TRUE END) -- rename param with '@' as sqlc shortcut
  AND (CASE WHEN @f2::TEXT <> '' THEN jsonb_path_exists(log, @f2::JSONPATH) ELSE TRUE END) -- cast types, so the parameters in Go are easy to use
ORDER BY time ASC
LIMIT $2;

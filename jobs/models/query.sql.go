// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.17.2
// source: query.sql

package models

import (
	"context"
)

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
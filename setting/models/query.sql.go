// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.19.1
// source: query.sql

package models

import (
	"context"
)

const deleteSetting = `-- name: DeleteSetting :exec
DELETE
FROM arrower.setting
WHERE key = $1
`

func (q *Queries) DeleteSetting(ctx context.Context, key string) error {
	_, err := q.db.Exec(ctx, deleteSetting, key)
	return err
}

const getSetting = `-- name: GetSetting :one
SELECT value
FROM arrower.setting
WHERE key = $1
LIMIT 1
`

func (q *Queries) GetSetting(ctx context.Context, key string) (string, error) {
	row := q.db.QueryRow(ctx, getSetting, key)
	var value string
	err := row.Scan(&value)
	return value, err
}

const getSettings = `-- name: GetSettings :many
SELECT key, value
FROM arrower.setting
WHERE key = ANY ($1::TEXT[])
`

type GetSettingsRow struct {
	Key   string
	Value string
}

func (q *Queries) GetSettings(ctx context.Context, compositeKeys []string) ([]GetSettingsRow, error) {
	rows, err := q.db.Query(ctx, getSettings, compositeKeys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetSettingsRow
	for rows.Next() {
		var i GetSettingsRow
		if err := rows.Scan(&i.Key, &i.Value); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const upsertSetting = `-- name: UpsertSetting :exec
INSERT INTO arrower.setting (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET updated_at = NOW(),
                                value      = $2
`

type UpsertSettingParams struct {
	Key   string
	Value string
}

func (q *Queries) UpsertSetting(ctx context.Context, arg UpsertSettingParams) error {
	_, err := q.db.Exec(ctx, upsertSetting, arg.Key, arg.Value)
	return err
}

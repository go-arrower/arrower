// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.19.1

package models

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ArrowerLog struct {
	Time   pgtype.Timestamptz
	UserID uuid.NullUUID
	Log    []byte
}

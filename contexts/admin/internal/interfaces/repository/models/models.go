// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.19.1

package models

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type ArrowerGueJob struct {
	JobID      string
	Priority   int16
	RunAt      pgtype.Timestamptz
	JobType    string
	Args       []byte
	ErrorCount int32
	LastError  string
	Queue      string
	CreatedAt  pgtype.Timestamptz
	UpdatedAt  pgtype.Timestamptz
}

type ArrowerGueJobsHistory struct {
	JobID      string
	Priority   int16
	RunAt      pgtype.Timestamptz
	JobType    string
	Args       []byte
	Queue      string
	RunCount   int32
	RunError   string
	CreatedAt  pgtype.Timestamptz
	UpdatedAt  pgtype.Timestamptz
	Success    bool
	FinishedAt pgtype.Timestamptz
	PrunedAt   pgtype.Timestamptz
}

type ArrowerGueJobsSchedule struct {
	Queue     string
	Spec      string
	JobType   string
	Args      []byte
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

type ArrowerGueJobsWorkerPool struct {
	ID        string
	Queue     string
	Workers   int16
	GitHash   string
	JobTypes  []string
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

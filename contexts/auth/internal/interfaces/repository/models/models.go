// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.19.1

package models

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type AuthSession struct {
	Key          []byte
	Data         []byte
	ExpiresAtUtc pgtype.Timestamptz
	UserID       uuid.NullUUID
	UserAgent    string
	CreatedAt    pgtype.Timestamptz
	UpdatedAt    pgtype.Timestamptz
}

type AuthUser struct {
	ID              uuid.UUID
	CreatedAt       pgtype.Timestamptz
	UpdatedAt       pgtype.Timestamptz
	Login           string
	PasswordHash    string
	NameFirstname   string
	NameLastname    string
	NameDisplayname string
	Birthday        pgtype.Date
	Locale          string
	TimeZone        string
	PictureUrl      string
	Profile         pgtype.Hstore
	VerifiedAtUtc   pgtype.Timestamptz
	BlockedAtUtc    pgtype.Timestamptz
	SuperuserAtUtc  pgtype.Timestamptz
}

type AuthUserVerification struct {
	Token         uuid.UUID
	UserID        uuid.UUID
	ValidUntilUtc pgtype.Timestamptz
	CreatedAt     pgtype.Timestamptz
	UpdatedAt     pgtype.Timestamptz
}
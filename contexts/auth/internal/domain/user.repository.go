package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrPersistenceFailed = errors.New("persistence operation failed")
)

type Repository interface {
	All(ctx context.Context, filter Filter) ([]User, error)

	FindByID(ctx context.Context, id ID) (User, error)
	FindByLogin(ctx context.Context, login Login) (User, error)
	ExistsByLogin(ctx context.Context, login Login) (bool, error)

	Count(ctx context.Context) (int, error)

	Save(ctx context.Context, user User) error

	Delete(ctx context.Context, user User) error
	DeleteByID(ctx context.Context, id ID) error
	DeleteByIDs(ctx context.Context, ids []ID) error
	DeleteAll(ctx context.Context) error

	CreateVerificationToken(ctx context.Context, token VerificationToken) error
	VerificationTokenByToken(ctx context.Context, token uuid.UUID) (VerificationToken, error)
}

type Filter struct {
	Offset Login
	Limit  uint
}

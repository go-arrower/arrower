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
	AllByIDs(ctx context.Context, ids []ID) ([]User, error) // todo remove, as it is not called

	FindByID(ctx context.Context, id ID) (User, error)
	FindByLogin(ctx context.Context, login Login) (User, error)
	ExistsByID(ctx context.Context, id ID) (bool, error) // todo rm
	ExistsByLogin(ctx context.Context, login Login) (bool, error)

	Count(ctx context.Context) (int, error)

	Save(ctx context.Context, user User) error
	SaveAll(ctx context.Context, users []User) error // todo rm

	Delete(ctx context.Context, user User) error
	DeleteByID(ctx context.Context, id ID) error
	DeleteByIDs(ctx context.Context, ids []ID) error
	DeleteAll(ctx context.Context) error

	// todo investigate if this is good or token should have its own repo or whatever the heck an aggregate is
	CreateVerificationToken(ctx context.Context, token VerificationToken) error
	VerificationTokenByToken(ctx context.Context, token uuid.UUID) (VerificationToken, error)
}

type Filter struct {
	Offset Login
	Limit  uint
}

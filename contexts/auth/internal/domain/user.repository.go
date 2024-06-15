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
	All(context.Context, Filter) ([]User, error)
	AllByIDs(context.Context, []ID) ([]User, error) // todo remove, as it is not called

	FindByID(context.Context, ID) (User, error)
	FindByLogin(context.Context, Login) (User, error)
	ExistsByID(context.Context, ID) (bool, error) // todo rm
	ExistsByLogin(context.Context, Login) (bool, error)

	Count(context.Context) (int, error)

	Save(context.Context, User) error
	SaveAll(context.Context, []User) error // todo rm

	Delete(context.Context, User) error
	DeleteByID(context.Context, ID) error
	DeleteByIDs(context.Context, []ID) error
	DeleteAll(context.Context) error

	// todo investigate if this is good or token should have its own repo or whatever the heck an aggregate is
	CreateVerificationToken(context.Context, VerificationToken) error
	VerificationTokenByToken(context.Context, uuid.UUID) (VerificationToken, error)
}

type Filter struct {
	Limit  uint
	Offset Login
}

package application

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
)

var ErrNewUserFailed = errors.New("new user failed")

func NewNewUserCommandHandler(
	repo domain.Repository,
	registrator *domain.RegistrationService,
) app.Command[NewUserCommand] {
	return app.NewValidatedCommand[NewUserCommand](nil, &newUserCommandHandler{
		repo:        repo,
		registrator: registrator,
	})
}

type newUserCommandHandler struct {
	repo        domain.Repository
	registrator *domain.RegistrationService
}

type (
	NewUserCommand struct {
		Email       string `form:"email"       validate:"max=1024,required,email"`
		FirstName   string `form:"firstName"   validate:"max=1024"`
		LastName    string `form:"lastName"    validate:"max=1024"`
		DisplayName string `form:"displayName" validate:"max=1024"`
		Superuser   bool   `form:"superuser"   validate:"boolean"`
	}
)

const passwordLength = 12

func (h *newUserCommandHandler) H(ctx context.Context, cmd NewUserCommand) error {
	usr, err := h.registrator.RegisterNewUser(ctx, cmd.Email, generatePassword(passwordLength))
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	usr.Name = domain.NewName(cmd.FirstName, cmd.LastName, cmd.DisplayName)

	if cmd.Superuser {
		usr.Superuser = domain.BoolFlag{}.SetTrue()
	}

	err = h.repo.Save(ctx, usr)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"

func generatePassword(length int) string {
	password := make([]byte, length)
	for i := range password {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		password[i] = charset[idx.Int64()]
	}

	return string(password)
}

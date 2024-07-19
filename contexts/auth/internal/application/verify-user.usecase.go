package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
)

var ErrVerifyUserFailed = errors.New("verify user failed")

func NewVerifyUserCommandHandler(repo domain.Repository) app.Command[VerifyUserCommand] {
	return app.NewValidatedCommand[VerifyUserCommand](nil, &verifyUserCommandHandler{repo: repo})
}

type verifyUserCommandHandler struct {
	repo domain.Repository
}

type (
	VerifyUserCommand struct {
		UserID domain.ID `validate:"required"`
		Token  uuid.UUID `validate:"required"`
	}
)

func (h *verifyUserCommandHandler) H(ctx context.Context, cmd VerifyUserCommand) error {
	usr, err := h.repo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return fmt.Errorf("could not get user: %w", err)
	}

	verify := domain.NewVerificationService(h.repo)

	err = verify.Verify(ctx, &usr, cmd.Token)
	if err != nil {
		return fmt.Errorf("could not verify user: %w", err)
	}

	return nil
}

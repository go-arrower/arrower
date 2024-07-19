package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"

	"github.com/go-arrower/arrower/app"
)

var ErrUnblockUserFailed = errors.New("unblock user failed")

func NewUnblockUserRequestHandler(repo domain.Repository) app.Request[UnblockUserRequest, UnblockUserResponse] {
	return app.NewValidatedRequest[UnblockUserRequest, UnblockUserResponse](nil, &unblockUserRequestHandler{repo: repo})
}

type unblockUserRequestHandler struct {
	repo domain.Repository
}

type (
	UnblockUserRequest struct {
		UserID domain.ID `validate:"required"`
	}
	UnblockUserResponse struct {
		UserID  domain.ID
		Blocked domain.BoolFlag
	}
)

// todo merge usecase with the block use case, IF possible
func (h *unblockUserRequestHandler) H(ctx context.Context, req UnblockUserRequest) (UnblockUserResponse, error) {
	usr, err := h.repo.FindByID(ctx, req.UserID)
	if err != nil {
		return UnblockUserResponse{}, fmt.Errorf("could not get user: %w", err)
	}

	usr.Unblock()

	err = h.repo.Save(ctx, usr)
	if err != nil {
		return UnblockUserResponse{}, fmt.Errorf("could not get user: %w", err)
	}

	return UnblockUserResponse{
		UserID:  usr.ID,
		Blocked: usr.Blocked,
	}, nil
}

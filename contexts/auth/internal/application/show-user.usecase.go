package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
)

var ErrShowUserFailed = errors.New("show user failed")

func NewShowUserQueryHandler(
	repo domain.Repository,
) app.Query[ShowUserQuery, ShowUserResponse] {
	return &showUserQueryHandler{
		repo: repo,
	}
}

type showUserQueryHandler struct {
	repo domain.Repository
}

type (
	ShowUserQuery struct {
		UserID domain.ID
	}
	ShowUserResponse struct {
		User domain.User
	}
)

func (h *showUserQueryHandler) H(ctx context.Context, query ShowUserQuery) (ShowUserResponse, error) {
	if query.UserID == "" {
		return ShowUserResponse{}, fmt.Errorf("%w: invalid inout", ErrShowUserFailed)
	}

	usr, err := h.repo.FindByID(ctx, query.UserID)
	if err != nil {
		return ShowUserResponse{}, fmt.Errorf("could not get user: %w", err)
	}

	return ShowUserResponse{User: usr}, nil
}

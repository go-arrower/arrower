package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
)

var (
	ErrBlockUserFailed   = errors.New("block user failed")
	ErrUnblockUserFailed = errors.New("unblock user failed")
)

func NewBlockUserRequestHandler(repo domain.Repository) app.Request[BlockUserRequest, BlockUserResponse] {
	return app.NewValidatedRequest[BlockUserRequest, BlockUserResponse](nil, &blockUserRequestHandler{repo: repo})
}

type blockUserRequestHandler struct {
	repo domain.Repository
}

type (
	BlockUserRequest struct {
		UserID     domain.ID `validate:"required"`
		SetBlocked *bool     `validate:"required"`
	}
	BlockUserResponse struct {
		Blocked domain.BoolFlag
		UserID  domain.ID
	}
)

func (h *blockUserRequestHandler) H(ctx context.Context, req BlockUserRequest) (BlockUserResponse, error) {
	usr, err := h.repo.FindByID(ctx, req.UserID)
	if err != nil {
		return BlockUserResponse{}, fmt.Errorf("could not get user: %w", err)
	}

	if *req.SetBlocked {
		usr.Block()
	} else {
		usr.Unblock()
	}

	err = h.repo.Save(ctx, usr)
	if err != nil {
		return BlockUserResponse{}, fmt.Errorf("could not get user: %w", err)
	}

	return BlockUserResponse{
		UserID:  usr.ID,
		Blocked: usr.Blocked,
	}, nil
}

package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
)

var (
	ErrLoginFailed  = errors.New("login failed")
	ErrInvalidInput = errors.New("invalid input")
)

type (
	SendConfirmationNewDeviceLoggedIn struct {
		UserID     domain.ID
		OccurredAt time.Time
		IP         domain.ResolvedIP
		Device     domain.Device
		// Ip Location
	}
)

type (
	BlockUserRequest struct {
		UserID domain.ID `validate:"required"`
	}
	BlockUserResponse struct {
		UserID  domain.ID
		Blocked domain.BoolFlag
	}
)

func BlockUser(repo domain.Repository) func(context.Context, BlockUserRequest) (BlockUserResponse, error) {
	return func(ctx context.Context, in BlockUserRequest) (BlockUserResponse, error) {
		usr, err := repo.FindByID(ctx, in.UserID)
		if err != nil {
			return BlockUserResponse{}, fmt.Errorf("could not get user: %w", err)
		}

		usr.Block()

		err = repo.Save(ctx, usr)
		if err != nil {
			return BlockUserResponse{}, fmt.Errorf("could not get user: %w", err)
		}

		return BlockUserResponse{
			UserID:  usr.ID,
			Blocked: usr.Blocked,
		}, nil
	}
}

func UnblockUser(repo domain.Repository) func(context.Context, BlockUserRequest) (BlockUserResponse, error) {
	return func(ctx context.Context, in BlockUserRequest) (BlockUserResponse, error) {
		usr, err := repo.FindByID(ctx, in.UserID)
		if err != nil {
			return BlockUserResponse{}, fmt.Errorf("could not get user: %w", err)
		}

		usr.Unblock()

		err = repo.Save(ctx, usr)
		if err != nil {
			return BlockUserResponse{}, fmt.Errorf("could not get user: %w", err)
		}

		return BlockUserResponse{
			UserID:  usr.ID,
			Blocked: usr.Blocked,
		}, nil
	}
}

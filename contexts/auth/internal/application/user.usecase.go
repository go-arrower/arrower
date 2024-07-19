package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"

	"github.com/google/uuid"
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
	VerifyUserRequest struct {
		UserID domain.ID `validate:"required"`
		Token  uuid.UUID `validate:"required"`
	}
)

func VerifyUser(repo domain.Repository) func(context.Context, VerifyUserRequest) error {
	return func(ctx context.Context, in VerifyUserRequest) error {
		usr, err := repo.FindByID(ctx, in.UserID)
		if err != nil {
			return fmt.Errorf("could not get user: %w", err)
		}

		verify := domain.NewVerificationService(repo)

		err = verify.Verify(ctx, &usr, in.Token)
		if err != nil {
			return fmt.Errorf("could not verify user: %w", err)
		}

		return nil
	}
}

type (
	ShowUserRequest struct {
		UserID domain.ID
	}
	ShowUserResponse struct {
		User domain.User
	}
)

func ShowUser(repo domain.Repository) func(context.Context, ShowUserRequest) (ShowUserResponse, error) {
	return func(ctx context.Context, in ShowUserRequest) (ShowUserResponse, error) {
		if in.UserID == "" {
			return ShowUserResponse{}, ErrInvalidInput
		}

		usr, err := repo.FindByID(ctx, in.UserID)
		if err != nil {
			return ShowUserResponse{}, fmt.Errorf("could not get user: %w", err)
		}

		return ShowUserResponse{User: usr}, nil
	}
}

type (
	NewUserRequest struct {
		Email       string `form:"email"       validate:"max=1024,required,email"`
		FirstName   string `form:"firstName"   validate:"max=1024"`
		LastName    string `form:"lastName"    validate:"max=1024"`
		DisplayName string `form:"displayName" validate:"max=1024"`
		Superuser   bool   `form:"superuser"   validate:"boolean"`
	}
)

func NewUser(repo domain.Repository, registrator *domain.RegistrationService) func(context.Context, NewUserRequest) error {
	return func(ctx context.Context, in NewUserRequest) error {
		usr, err := registrator.RegisterNewUser(ctx, in.Email, "RanDomS1cuP!") // todo set random pw
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		usr.Name = domain.NewName(in.FirstName, in.LastName, in.DisplayName)

		if in.Superuser {
			usr.SuperUser = domain.BoolFlag{}.SetTrue()
		}

		err = repo.Save(ctx, usr)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		return nil
	}
}

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

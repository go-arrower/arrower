package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
	"github.com/go-arrower/arrower/contexts/auth/internal/infrastructure"
	"github.com/go-arrower/arrower/jobs"
)

var ErrRegisterUserFailed = errors.New("register user failed")

func NewRegisterUserRequestHandler(
	logger alog.Logger,
	repo domain.Repository,
	registrator *domain.RegistrationService,
	queue jobs.Enqueuer,
) app.Request[RegisterUserRequest, RegisterUserResponse] {
	return app.NewValidatedRequest[RegisterUserRequest, RegisterUserResponse](nil, &registerUserRequestHandler{
		logger:      logger,
		repo:        repo,
		registrator: registrator,
		queue:       queue,
		ip:          infrastructure.NewIP2LocationService(""),
	})
}

type registerUserRequestHandler struct {
	logger      alog.Logger
	repo        domain.Repository
	registrator *domain.RegistrationService
	queue       jobs.Enqueuer
	ip          domain.IPResolver
}

type (
	RegisterUserRequest struct {
		RegisterEmail          string `form:"login"                 validate:"max=1024,required,email"`
		Password               string `form:"password"              validate:"max=1024,min=8"`
		PasswordConfirmation   string `form:"password_confirmation" validate:"max=1024,eqfield=Password"`
		AcceptedTermsOfService bool   `form:"tos"                   validate:"required"`

		UserAgent  string `validate:"max=2048"`
		IP         string `validate:"ip"`
		SessionKey string
	}
	RegisterUserResponse struct {
		User domain.Descriptor
	}
)

func (h *registerUserRequestHandler) H(ctx context.Context, req RegisterUserRequest) (RegisterUserResponse, error) {
	usr, err := h.registrator.RegisterNewUser(ctx, req.RegisterEmail, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			h.logger.Log(ctx, slog.LevelInfo, "register new user failed",
				slog.String("email", req.RegisterEmail),
				slog.String("ip", req.IP),
				slog.String("err", err.Error()),
			)
		}

		return RegisterUserResponse{}, fmt.Errorf("%w", err)
	}

	// The session is not valid until the end of the controller.
	// Thus, the session is created here and very short-lived, as the controller will update it with the right values.
	usr.Sessions = append(usr.Sessions, domain.Session{
		ID:        req.SessionKey,
		Device:    domain.NewDevice(req.UserAgent),
		CreatedAt: time.Now().UTC(),
		// ExpiresAt: // will be set & updated via the session store
	})

	err = h.repo.Save(ctx, usr)
	if err != nil {
		return RegisterUserResponse{}, fmt.Errorf("could not save new user: %w", err)
	}

	resolved, err := h.ip.ResolveIP(req.IP)
	if err != nil {
		return RegisterUserResponse{}, fmt.Errorf("could not resolve ip address: %w", err)
	}

	// !!! CONSIDER !!! if the email output port is async (outbox pattern) call it directly instead of a job
	err = h.queue.Enqueue(ctx, NewUserVerificationEmail{
		UserID:     usr.ID,
		OccurredAt: time.Now().UTC(),
		IP:         resolved,
		Device:     domain.NewDevice(req.UserAgent),
	})
	if err != nil {
		return RegisterUserResponse{}, fmt.Errorf("could not queue job to send verification email: %w", err)
	}

	// todo return a short "UserDescriptor" or something instead of a partial user.
	return RegisterUserResponse{User: usr.Descriptor()}, nil
}

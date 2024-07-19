package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-arrower/arrower/contexts/auth/internal/infrastructure"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/jobs"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"

	"github.com/go-arrower/arrower/app"
)

var ErrLoginUserFailed = errors.New("login user failed")

func NewLoginUserRequestHandler(
	logger alog.Logger,
	repo domain.Repository,
	queue jobs.Enqueuer,
	authenticator *domain.AuthenticationService,
) app.Request[LoginUserRequest, LoginUserResponse] {
	return &loginUserRequestHandler{
		logger:        logger,
		repo:          repo,
		queue:         queue,
		authenticator: authenticator,
		ip:            infrastructure.NewIP2LocationService(""),
	}
}

type loginUserRequestHandler struct {
	logger        alog.Logger
	repo          domain.Repository
	queue         jobs.Enqueuer
	authenticator *domain.AuthenticationService
	ip            domain.IPResolver
}

type (
	LoginUserRequest struct { //nolint:govet // fieldalignment less important than grouping of params.
		LoginEmail string `form:"login" validate:"max=1024,required,email"`
		Password   string `form:"password" validate:"max=1024,min=8"`

		IsNewDevice bool
		UserAgent   string
		IP          string `validate:"ip"`
		SessionKey  string
	}
	LoginUserResponse struct {
		User domain.User
	}
)

func (h *loginUserRequestHandler) H(ctx context.Context, req LoginUserRequest) (LoginUserResponse, error) {
	usr, err := h.repo.FindByLogin(ctx, domain.Login(req.LoginEmail))
	if err != nil {
		h.logger.Log(ctx, slog.LevelInfo, "login failed",
			slog.String("email", req.LoginEmail),
			slog.String("ip", req.IP),
			slog.String("err", err.Error()),
		)

		return LoginUserResponse{}, ErrLoginFailed
	}

	if !h.authenticator.Authenticate(ctx, usr, req.Password) {
		h.logger.Log(ctx, slog.LevelInfo, "login failed",
			slog.String("email", req.LoginEmail),
			slog.String("ip", req.IP),
		)

		return LoginUserResponse{}, ErrLoginFailed
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
		return LoginUserResponse{}, fmt.Errorf("could not update user session: %w", err)
	}
	// FIXME: add a method to user or a domain service, that ensures session is not added, if one with same ID already exists.

	if req.IsNewDevice {
		resolved, err := h.ip.ResolveIP(req.IP)
		if err != nil {
			return LoginUserResponse{}, fmt.Errorf("could not resolve ip address: %w", err)
		}

		err = h.queue.Enqueue(ctx, SendConfirmationNewDeviceLoggedIn{
			UserID:     usr.ID,
			OccurredAt: time.Now().UTC(),
			IP:         resolved,
			Device:     domain.NewDevice(req.UserAgent),
		})
		if err != nil {
			return LoginUserResponse{}, fmt.Errorf("could not queue confirmation about new device: %w", err)
		}
	}

	return LoginUserResponse{User: usr}, nil
}

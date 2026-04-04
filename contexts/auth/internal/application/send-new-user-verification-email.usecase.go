package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain/logging"
)

var ErrSendNewUserVerificationEmailFailed = errors.New("send new user verification email failed")

func NewSendNewUserVerificationEmailJobHandler(
	logger alog.Logger,
	repo domain.Repository,
) app.Job[NewUserVerificationEmail] {
	return &sendNewUserVerificationEmailJobHandler{
		logger: logger,
		repo:   repo,
	}
}

type sendNewUserVerificationEmailJobHandler struct {
	logger alog.Logger
	repo   domain.Repository
}

type (
	NewUserVerificationEmail struct {
		UserID     domain.ID
		OccurredAt time.Time
		IP         domain.ResolvedIP
		Device     domain.Device
	}
)

func (h *sendNewUserVerificationEmailJobHandler) H(ctx context.Context, job NewUserVerificationEmail) error {
	usr, err := h.repo.FindByID(ctx, job.UserID)
	if err != nil {
		return fmt.Errorf("could not get user: %w", err)
	}

	verify := domain.NewVerificationService(h.repo)

	token, err := verify.NewVerificationToken(ctx, usr)
	if err != nil {
		return fmt.Errorf("could not generate verification token: %w", err)
	}

	// later: instead of logging this => send it to an email output port
	// later: assert the email has been sent via the email interface
	h.logger.InfoContext(ctx, "send verification email to user",
		logging.Token(token.Token().String()),
		logging.Device(job.Device.String()),
		logging.IP(job.IP.IP.String()),
		logging.Time(job.OccurredAt),
		logging.Email(string(usr.Login)),
	)

	return nil
}

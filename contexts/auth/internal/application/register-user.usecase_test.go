package application_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
	"github.com/go-arrower/arrower/jobs"
)

func TestRegisterUserRequestHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("login already in use", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository()
		_ = repo.Save(ctx, userVerified)

		registrator := registrator(repo)

		buf := bytes.Buffer{}
		logger := alog.NewTest(&buf)
		alog.Unwrap(logger).SetLevel(alog.LevelInfo)

		handler := application.NewRegisterUserRequestHandler(logger, repo, registrator, nil)

		_, err := handler.H(ctx, application.RegisterUserRequest{RegisterEmail: user0Login})
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)

		// assert failed attempt is logged, e.g. for monitoring or fail2ban etc.
		assert.Contains(t, buf.String(), "register new user failed")
	})

	t.Run("register new user", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository()
		queue := jobs.NewTestingJobs()
		registrator := registrator(repo)

		handler := application.NewRegisterUserRequestHandler(alog.NewNoopLogger(), repo, registrator, queue)

		usr, err := handler.H(ctx, application.RegisterUserRequest{
			RegisterEmail: newUserLogin,
			Password:      strongPassword,
			UserAgent:     userAgent,
			IP:            ip,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, usr.User.ID)

		dbUser, err := repo.FindByLogin(ctx, newUserLogin)
		assert.NoError(t, err)
		assert.Empty(t, dbUser.Verified)
		assert.Empty(t, dbUser.Blocked)

		// assert session got updated with device info
		dbUser, _ = repo.FindByID(ctx, usr.User.ID)
		assert.Len(t, dbUser.Sessions, 1)
		assert.Equal(t, domain.NewDevice(userAgent), dbUser.Sessions[0].Device)

		queue.Assert(t).Queued(application.NewUserVerificationEmail{}, 1)
		job := queue.GetFirstOf(application.NewUserVerificationEmail{}).(application.NewUserVerificationEmail)
		assert.NotEmpty(t, job.UserID)
		assert.NotEmpty(t, job.OccurredAt)
		assert.Equal(t, resolvedIP, job.IP)
		assert.Equal(t, domain.NewDevice(userAgent), job.Device)
	})
}

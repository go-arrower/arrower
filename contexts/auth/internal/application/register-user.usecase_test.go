package application_test

import (
	"log/slog"
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

	t.Run("validate inputs", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			in application.RegisterUserRequest
		}{
			"empty":               {application.RegisterUserRequest{}},
			"missing email":       {registerUserRequest(empty("RegisterEmail"))},
			"missing password":    {registerUserRequest(empty("Password"))},
			"passwords not equal": {registerUserRequest(with("PasswordConfirmation", "123456789"))},
			"missing tos":         {registerUserRequest(empty("AcceptedTermsOfService"))},
			"missing ip":          {registerUserRequest(empty("IP"))},
			"invalid ip":          {registerUserRequest(with("IP", "invalid-ip-format"))},
		}

		handler := application.NewRegisterUserRequestHandler(nil, nil, nil, nil)

		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				_, err := handler.H(ctx, tc.in)
				assert.Error(t, err)
				t.Log(name, err)
			})
		}
	})

	t.Run("login already in use", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()
		_ = repo.Save(ctx, userVerified)

		registrator := registrator(repo)

		logger := alog.Test(t)
		alog.Unwrap(logger).SetLevel(alog.LevelInfo)

		handler := application.NewRegisterUserRequestHandler(logger, repo, registrator, nil)

		_, err := handler.H(ctx, registerUserRequest(with("RegisterEmail", user0Login)))
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)

		// assert failed attempt is logged, e.g. for monitoring or fail2ban etc.
		logger.Contains("register new user failed")
	})

	t.Run("register new user", func(t *testing.T) {
		t.Parallel()
		t.Skip("FIXME: skip for now, ip2location is not available in this repo")

		repo := repository.NewUserMemoryRepository()
		queue := jobs.Test(t)
		registrator := registrator(repo)

		handler := application.NewRegisterUserRequestHandler(slog.New(slog.DiscardHandler), repo, registrator, queue)

		usr, err := handler.H(ctx, registerUserRequest(
			with("RegisterEmail", newUserLogin),
			with("UserAgent", userAgent),
			with("IP", ip)))
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

		queue.Contains(application.NewUserVerificationEmail{})
		job := queue.GetFirstOf(application.NewUserVerificationEmail{}).(application.NewUserVerificationEmail)
		assert.NotEmpty(t, job.UserID)
		assert.NotEmpty(t, job.OccurredAt)
		assert.Equal(t, resolvedIP, job.IP)
		assert.Equal(t, domain.NewDevice(userAgent), job.Device)
	})
}

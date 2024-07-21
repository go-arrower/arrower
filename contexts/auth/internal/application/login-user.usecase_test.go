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

func TestLoginUserRequestHandler_H(t *testing.T) {
	t.Parallel()

	t.Run("authentication fails", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()
		_ = repo.Save(ctx, userVerified)

		buf := bytes.Buffer{}
		logger := alog.NewTest(&buf)
		alog.Unwrap(logger).SetLevel(alog.LevelInfo)

		handler := application.NewLoginUserRequestHandler(logger, repo, nil, authentificator())

		_, err := handler.H(ctx, application.LoginUserRequest{
			LoginEmail: user0Login,
			Password:   "wrong-password",
			IP:         ip,
		})
		assert.Error(t, err)
		assert.ErrorIs(t, err, application.ErrLoginUserFailed)

		// assert failed attempt is logged, e.g. for monitoring or fail2ban etc.
		assert.Contains(t, buf.String(), "login failed")
	})

	t.Run("login succeeds", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()
		_ = repo.Save(ctx, userVerified)
		queue := jobs.NewTestingJobs()

		handler := application.NewLoginUserRequestHandler(alog.NewTest(nil), repo, queue, authentificator()) // todo logger

		res, err := handler.H(ctx, application.LoginUserRequest{
			LoginEmail: validUserLogin,
			Password:   strongPassword,
			UserAgent:  userAgent,
			SessionKey: "new-session-key",
			IP:         ip,
		})

		// assert return values
		assert.NoError(t, err)
		assert.Equal(t, domain.Login(validUserLogin), res.User.Login)
		assert.NotEmpty(t, validUserLogin, res.User.ID)

		// assert session got updated with device info // todo IF repo gets methods to retrieve sessions directly: use them instead
		usr, _ := repo.FindByID(ctx, userVerified.ID)
		assert.Len(t, usr.Sessions, 2)
		assert.Equal(t, domain.NewDevice(userAgent), usr.Sessions[1].Device)

		queue.Assert(t).Queued(application.SendConfirmationNewDeviceLoggedIn{}, 0)
	})

	t.Run("unknown device - send email about login to user", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewUserMemoryRepository()
		_ = repo.Save(ctx, userVerified)
		queue := jobs.NewTestingJobs()

		handler := application.NewLoginUserRequestHandler(alog.NewTest(nil), repo, queue, authentificator()) // todo logger

		_, err := handler.H(ctx, application.LoginUserRequest{
			LoginEmail:  validUserLogin,
			Password:    strongPassword,
			UserAgent:   userAgent,
			IP:          ip,
			SessionKey:  "new-session-key",
			IsNewDevice: true,
		})

		// assert return values
		assert.NoError(t, err)
		queue.Assert(t).Queued(application.SendConfirmationNewDeviceLoggedIn{}, 1)
		job := queue.GetFirstOf(application.SendConfirmationNewDeviceLoggedIn{}).(application.SendConfirmationNewDeviceLoggedIn)
		assert.NotEmpty(t, job.UserID)
		assert.NotEmpty(t, job.OccurredAt)
		assert.Equal(t, resolvedIP, job.IP)
		assert.Equal(t, domain.NewDevice(userAgent), job.Device)
	})
}

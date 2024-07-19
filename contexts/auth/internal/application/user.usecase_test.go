package application_test

import (
	"bytes"
	"testing"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/jobs"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/application"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository"
)

func TestRegisterUser(t *testing.T) {
	t.Parallel()

	t.Run("login already in use", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository()
		_ = repo.Save(ctx, userVerified)

		registrator := registrator(repo)

		buf := bytes.Buffer{}
		logger := alog.NewTest(&buf)
		alog.Unwrap(logger).SetLevel(alog.LevelInfo)

		cmd := application.RegisterUser(logger, repo, registrator, nil)

		_, err := cmd(ctx, application.RegisterUserRequest{RegisterEmail: user0Login})
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

		cmd := application.RegisterUser(alog.NewNoopLogger(), repo, registrator, queue)

		usr, err := cmd(ctx, application.RegisterUserRequest{
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

func TestVerifyUser(t *testing.T) {
	t.Parallel()

	t.Run("verify user", func(t *testing.T) {
		t.Parallel()

		// setup
		repo := repository.NewMemoryRepository()
		repo.Save(ctx, userNotVerified)

		usr, _ := repo.FindByID(ctx, userNotVerifiedUserID)
		verify := domain.NewVerificationService(repo)
		token, _ := verify.NewVerificationToken(ctx, usr)

		// action
		err := application.VerifyUser(repo)(ctx, application.VerifyUserRequest{
			Token:  token.Token(),
			UserID: userNotVerifiedUserID,
		})
		assert.NoError(t, err)

		usr, _ = repo.FindByID(ctx, userNotVerifiedUserID)
		assert.True(t, usr.IsVerified())
	})
}

func TestShowUser(t *testing.T) {
	t.Parallel()

	t.Run("invalid userID", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository()

		cmd := application.ShowUser(repo)
		res, err := cmd(ctx, application.ShowUserRequest{})
		assert.Error(t, err)
		assert.Empty(t, res)
	})

	t.Run("show user", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository()
		repo.Save(ctx, userVerified)

		cmd := application.ShowUser(repo)
		res, err := cmd(ctx, application.ShowUserRequest{
			UserID: userIDZero,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, res)

		assert.Equal(t, userIDZero, res.User.ID)
		assert.Len(t, res.User.Sessions, 1)
	})
}

func TestBlockUser(t *testing.T) {
	t.Parallel()

	t.Run("block user", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository()
		repo.Save(ctx, userVerified)

		cmd := application.BlockUser(repo)
		_, err := cmd(ctx, application.BlockUserRequest{UserID: userIDZero})
		assert.NoError(t, err)

		// verify
		usr, err := repo.FindByID(ctx, userIDZero)
		assert.NoError(t, err)
		assert.True(t, usr.IsBlocked())
	})
}

func TestUnblockUser(t *testing.T) {
	t.Parallel()

	t.Run("unblock user", func(t *testing.T) {
		t.Parallel()

		repo := repository.NewMemoryRepository()
		repo.Save(ctx, userBlocked)

		cmd := application.UnblockUser(repo)
		_, err := cmd(ctx, application.BlockUserRequest{UserID: userBlockedUserID})
		assert.NoError(t, err)

		// verify
		usr, err := repo.FindByID(ctx, userBlockedUserID)
		assert.NoError(t, err)
		assert.True(t, !usr.IsBlocked())
	})
}

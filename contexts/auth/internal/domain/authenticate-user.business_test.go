package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
)

func TestAuthenticationService_Authenticate(t *testing.T) {
	t.Parallel()

	t.Run("login setting disabled", func(t *testing.T) {
		t.Parallel()

		authenticator := domain.NewAuthenticationService(settingsService(false))

		auth := authenticator.Authenticate(ctx, newVerifiedUser(), rawPassword)
		assert.False(t, auth)
	})

	t.Run("user not verified", func(t *testing.T) {
		t.Parallel()

		authenticator := domain.NewAuthenticationService(settingsService(true))

		auth := authenticator.Authenticate(ctx, newUser(), "")
		assert.False(t, auth)
	})

	t.Run("user blocked", func(t *testing.T) {
		t.Parallel()

		usr := newVerifiedUser()
		usr.Blocked = domain.BoolFlag{}.SetTrue()
		authenticator := domain.NewAuthenticationService(settingsService(true))

		auth := authenticator.Authenticate(ctx, usr, "")
		assert.False(t, auth)
	})

	t.Run("password doesn't match", func(t *testing.T) {
		t.Parallel()

		authenticator := domain.NewAuthenticationService(settingsService(true))

		auth := authenticator.Authenticate(ctx, newVerifiedUser(), "wrong-password")
		assert.False(t, auth)
	})

	t.Run("password matches", func(t *testing.T) {
		t.Parallel()

		authenticator := domain.NewAuthenticationService(settingsService(true))

		auth := authenticator.Authenticate(ctx, newVerifiedUser(), rawPassword)
		assert.True(t, auth)
	})
}

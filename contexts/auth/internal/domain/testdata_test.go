package domain_test

import (
	"context"

	"github.com/go-arrower/arrower/setting"

	"github.com/go-arrower/arrower/contexts/auth"
	"github.com/go-arrower/arrower/contexts/auth/internal/domain"
)

var (
	ctx                   = context.Background()
	rawPassword           = "0Secret!"
	strongPasswordHash, _ = domain.NewPasswordHash(rawPassword)
)

// newUser returns a new User, so you don't have to worry about changing fields when verifying.
func newUser() domain.User {
	return domain.User{
		ID:       domain.NewID(),
		Verified: domain.BoolFlag{}.SetFalse(),
	}
}

// used by RegistrationService
const (
	userID    = domain.ID("00000000-0000-0000-0000-000000000000")
	userLogin = "0@test.com"
)

var (
	userVerified = domain.User{
		ID:           userID,
		Login:        userLogin,
		PasswordHash: strongPasswordHash,
		Verified:     domain.BoolFlag{}.SetTrue(),
	}
)

// used by AuthenticationService
func settingsService(active bool) setting.Settings {
	settings := setting.NewInMemorySettings()
	settings.Save(ctx, auth.SettingAllowLogin, setting.NewValue(active))

	return settings
}

func newVerifiedUser() domain.User {
	usr := newUser()
	usr.Verified = domain.BoolFlag{}.SetTrue()
	usr.PasswordHash = strongPasswordHash

	return usr
}

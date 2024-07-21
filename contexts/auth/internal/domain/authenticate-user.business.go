package domain

import (
	"context"

	"github.com/go-arrower/arrower/contexts/auth"
	"github.com/go-arrower/arrower/setting"
)

func NewAuthenticationService(settings setting.Settings) *AuthenticationService {
	return &AuthenticationService{settingsService: settings}
}

type AuthenticationService struct {
	settingsService setting.Settings
}

func (s *AuthenticationService) Authenticate(ctx context.Context, user User, password string) bool {
	isLoginActive, err := s.settingsService.Setting(ctx, auth.SettingAllowLogin)
	if err != nil || !isLoginActive.MustBool() {
		return false
	}

	if !user.IsVerified() {
		return false
	}

	if user.IsBlocked() {
		return false
	}

	if !user.PasswordHash.Matches(password) {
		return false
	}

	return true
}

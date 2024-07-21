package views

import (
	"context"

	"github.com/go-arrower/arrower/contexts/auth"
	"github.com/go-arrower/arrower/setting"
)

const appTitle = "skeleton"

func NewDefaultBaseDataFunc(settings setting.Settings) func(_ context.Context) (map[string]any, error) {
	return func(ctx context.Context) (map[string]any, error) {
		isRegisterActive, _ := settings.Setting(ctx, auth.SettingAllowRegistration)
		isLoginActive, _ := settings.Setting(ctx, auth.SettingAllowLogin)

		showLoginBtn := isLoginActive.MustBool() && !auth.IsLoggedIn(ctx)

		data := map[string]any{
			"Title":                    appTitle,
			"Flashes":                  nil,
			"ShowRegistrationBtn":      isRegisterActive.MustBool() && !auth.IsLoggedIn(ctx),
			"ShowLoginBtn":             showLoginBtn,
			"ShowLogoutBtn":            auth.IsLoggedIn(ctx),
			"ShowAdminBtn":             auth.IsSuperuser(ctx),
			"ShowLoggedInAsUserBanner": auth.IsLoggedInAsOtherUser(ctx),
		}

		return data, nil
	}
}

//go:build e2e

package web_test

import (
	"net/http"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth/tests"
	"github.com/go-arrower/arrower/e2e"
)

func TestAssert_Login(t *testing.T) {
	t.Parallel()

	// access page that needs login and be rejected
	page := tests.Test(t).Goto("/auth/profile", e2e.WithRedirectPolicy(req.NoRedirectPolicy()))
	page.IsRedirected("should redirect")
	page.RedirectsTo("/auth/login?returnUrl=%2Fauth%2Fprofile", "should redirect to login page")
}

// todo logout

func TestScenario_LoginAndLogout(t *testing.T) {
	t.Parallel()

	suite := tests.Test(t)
	suite.RegisterUser("user@example.com", "01234aA*") // FIXME random values as vars

	page := suite.Goto("/auth/profile")
	page.IsPath("/auth/login", "should redirect to login page")
	page.HasQueryParam("returnUrl", "/auth/profile", "should forward to profile page after login")
	page.Contains("Login", "should redirect to login page")
	// todo has link to register page

	page.Scripts().Total(1)
	page.Scripts().HasOnlyLocal()
	page.Scripts().HasLibrary("htmx") // todo should be htmx.org or min or something => load via npm and makefile
	page.HasCookieTotal(0)

	page.Form("/auth/login")
	form := page.Form("/auth/login")
	form.Method(http.MethodPost)
	form.Total(4)
	form.HasField("login")
	form.HasField("password")
	form.HasField("remember_me")
	form.HasSubmit()
	form.HasNoErrors()

	// submit with invalid data
	page = form.Submit(map[string]any{
		"login":       "invalid email",
		"password":    "short", // less than 8 chars to trigger min validation
		"remember_me": "true",
	})

	form = page.Form("/auth/login")
	form.HasErrors()
	form.HasFieldError("login")
	form.HasFieldError("password")
	form.HasFieldValue("login", "invalid email")
	form.HasFieldValue("password", "")

	_, err := suite.PGx.Exec(t.Context(), `UPDATE auth.user SET verified_at_utc = now();`)
	assert.NoError(t, err)

	page = form.Submit(map[string]any{
		"login":       "user@example.com",
		"password":    "01234aA*",
		"remember_me": "true",
	})
	// page.IsPath("/auth/profile", "should redirect to profile page")
	// page.Contains("Profile", "should be redirected to profile page")
	page.HasCookieTotal(2)
	page.HasCookie("arrower.auth")
	page.HasCookie("arrower.auth.known_device")

	// logout and assert
	page = page.Goto("/auth/logout")
	page.HasCookieTotal(1)
	page.HasCookie("arrower.auth.known_device")

	page = page.Goto("/auth/profile")
	page.IsPath("/auth/login", "should redirect to login page")
	page.Contains("Login", "should redirect to login page")
	page.HasCookieTotal(1)

	// TODO: if login is disabled in the settings, there is not feedback for the user; it looks like login would be working
}

// todo Logged in → but no permission (e.g. admin area) → 403 Forbidden.

func TestScenario_LoginDisabled(t *testing.T) {}

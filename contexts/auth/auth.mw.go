package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	ctx2 "github.com/go-arrower/arrower/ctx"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

var ErrInvalidSessionValue = errors.New("invalid session value")

const (
	CtxAuthLoggedIn                  ctx2.CTXKey = "auth.pass"
	CtxAuthIsSuperuser               ctx2.CTXKey = "auth.superuser"
	CtxAuthIsSuperuserLoggedInAsUser ctx2.CTXKey = "auth.superuser_logged_in_as_user"
	// CtxAuthUserID                 arrower.CTXKey = "auth.user_id", see arrower/comtext.go.
)

const (
	// FIXME: is redundant and can disappear from the session, use the existance of user_id to set the flag in the ctx middleware.
	SessKeyLoggedIn               = "auth.user_is_logged_in" // FIXME don't export from the context => move internally
	SessKeyUserID                 = "auth.user_id"
	SessKeyIsSuperuser            = "auth.user_is_superuser"
	SessIsSuperuserLoggedInAsUser = "auth.superuser.is_logged_in_as_user"
	SessSuperuserOriginalUserID   = "auth.superuser.original_user_id"
)

// EnsureUserIsLoggedInMiddleware makes sure the routes can only be accessed by a logged-in user.
// It does set the User in the same way EnrichCtxWithUserInfoMiddleware does.
// OR LoginRequired.
func EnsureUserIsLoggedInMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	type passed struct {
		loggedIn bool
		userID   bool
	}

	return func(c echo.Context) error {
		sess, err := session.Get(SessionName, c)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		passed := passed{}

		if sess.Values[SessKeyLoggedIn] != nil {
			lin, ok := sess.Values[SessKeyLoggedIn].(bool)
			if !ok {
				return fmt.Errorf("could not access user_logged_in: %w", ErrInvalidSessionValue)
			}

			c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), CtxAuthLoggedIn, lin)))

			passed.loggedIn = lin
		}

		if sess.Values[SessKeyUserID] != nil {
			uID, ok := sess.Values[SessKeyUserID].(string)
			if !ok {
				return fmt.Errorf("could not access user_id: %w", ErrInvalidSessionValue)
			}

			c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), ctx2.CtxAuthUserID, uID)))

			passed.userID = true
		}

		if passed.loggedIn && passed.userID {
			return next(c)
		}

		return c.Redirect(http.StatusSeeOther, "/")
	}
}

func EnsureUserIsSuperuserMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	type passed struct {
		loggedIn    bool
		userID      bool
		isSuperuser bool
	}

	return func(c echo.Context) error {
		sess, err := session.Get(SessionName, c)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		passed := passed{}

		if sess.Values[SessKeyLoggedIn] != nil {
			lin, ok := sess.Values[SessKeyLoggedIn].(bool)
			if !ok {
				return fmt.Errorf("could not access user_logged_in: %w", ErrInvalidSessionValue)
			}

			c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), CtxAuthLoggedIn, lin)))

			passed.loggedIn = lin
		}

		if sess.Values[SessKeyUserID] != nil {
			uID, ok := sess.Values[SessKeyUserID].(string)
			if !ok {
				return fmt.Errorf("could not access user_id: %w", ErrInvalidSessionValue)
			}

			c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), ctx2.CtxAuthUserID, uID)))

			passed.userID = true
		}

		if sess.Values[SessKeyIsSuperuser] != nil {
			su, ok := sess.Values[SessKeyIsSuperuser].(bool)
			if !ok {
				return fmt.Errorf("could not access user_is_superuser: %w", ErrInvalidSessionValue)
			}

			c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), CtxAuthIsSuperuser, su)))

			passed.isSuperuser = su
		}

		if passed.loggedIn && passed.userID && passed.isSuperuser {
			return next(c)
		}

		return c.Redirect(http.StatusSeeOther, "/")
	}
}

// EnrichCtxWithUserInfoMiddleware checks if a User is logged in and puts those values into the http request's context,
// so they are available in other parts of the app. For convenience use the helpers like: IsLoggedIn.
// If you want to ensure only logged-in users can access a URL use EnsureUserIsLoggedInMiddleware instead.
func EnrichCtxWithUserInfoMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get(SessionName, c)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		if sess.Values[SessKeyLoggedIn] != nil {
			lin, ok := sess.Values[SessKeyLoggedIn].(bool)
			if !ok {
				return fmt.Errorf("could not access user_logged_in: %w", ErrInvalidSessionValue)
			}

			c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), CtxAuthLoggedIn, lin)))
		}

		if sess.Values[SessKeyUserID] != nil {
			uID, ok := sess.Values[SessKeyUserID].(string)
			if !ok {
				return fmt.Errorf("could not access user_id: %w", ErrInvalidSessionValue)
			}

			c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), ctx2.CtxAuthUserID, uID)))
		}

		if sess.Values[SessKeyIsSuperuser] != nil {
			su, ok := sess.Values[SessKeyIsSuperuser].(bool)
			if ok {
				c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), CtxAuthIsSuperuser, su)))
			}
		}

		if sess.Values[SessIsSuperuserLoggedInAsUser] != nil {
			if _, ok := sess.Values[SessIsSuperuserLoggedInAsUser].(bool); ok {
				c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), CtxAuthIsSuperuserLoggedInAsUser, true)))
			}
		}

		return next(c)
	}
}

func IsLoggedIn(ctx context.Context) bool {
	if v, ok := ctx.Value(CtxAuthLoggedIn).(bool); ok {
		return v
	}

	return false
}

func CurrentUserID(ctx context.Context) string {
	if v, ok := ctx.Value(ctx2.CtxAuthUserID).(string); ok {
		return v
	}

	return ""
}

func IsSuperUser(ctx context.Context) bool {
	if v, ok := ctx.Value(CtxAuthIsSuperuser).(bool); ok {
		return v
	}

	return false
}

func IsLoggedInAsOtherUser(ctx context.Context) bool {
	if v, ok := ctx.Value(CtxAuthIsSuperuserLoggedInAsUser).(bool); ok {
		return v
	}

	return false
}

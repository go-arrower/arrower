package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth"
)

func TestEnsureUserIsLoggedInMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("no session => redirect", func(t *testing.T) {
		t.Parallel()

		echoRouter := newEnsureRouterToAssertOnHandler(func(c echo.Context) error {
			ctx := c.Request().Context()
			assert.False(t, auth.IsLoggedIn(ctx))
			assert.Empty(t, auth.CurrentUserID(ctx))

			return c.NoContent(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/auth/", nil)
		rec := httptest.NewRecorder()

		echoRouter.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
	})

	t.Run("user is logged in", func(t *testing.T) {
		t.Parallel()

		echoRouter := newEnsureRouterToAssertOnHandler(func(c echo.Context) error {
			ctx := c.Request().Context()
			assert.True(t, auth.IsLoggedIn(ctx))
			assert.Equal(t, auth.UserID("1337"), auth.CurrentUserID(ctx))

			return c.NoContent(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/auth/", nil)
		req.AddCookie(getSessionCookie(echoRouter))

		rec := httptest.NewRecorder()

		echoRouter.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestIsSuperuser(t *testing.T) {
	t.Parallel()

	t.Run("no session => redirect", func(t *testing.T) {
		t.Parallel()

		echoRouter := newEnsureIsSuperuserRouterToAssertOnHandler(func(c echo.Context) error {
			ctx := c.Request().Context()
			assert.False(t, auth.IsLoggedIn(ctx))
			assert.False(t, auth.IsSuperuser(ctx))
			assert.Empty(t, auth.CurrentUserID(ctx))

			return c.NoContent(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
		rec := httptest.NewRecorder()

		echoRouter.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
	})

	t.Run("user is superuser", func(t *testing.T) {
		t.Parallel()

		echoRouter := newEnsureIsSuperuserRouterToAssertOnHandler(func(c echo.Context) error {
			ctx := c.Request().Context()
			assert.True(t, auth.IsLoggedIn(ctx))
			assert.True(t, auth.IsSuperuser(ctx))
			assert.Equal(t, auth.UserID("1337"), auth.CurrentUserID(ctx))

			return c.NoContent(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
		req.AddCookie(getSessionCookie(echoRouter))

		rec := httptest.NewRecorder()

		echoRouter.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestEnrichCtxWithUserInfoMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("no session => no id", func(t *testing.T) {
		t.Parallel()

		echoRouter := newEnrichRouterToAssertOnHandler(func(c echo.Context) error {
			ctx := c.Request().Context()
			assert.False(t, auth.IsLoggedIn(ctx))
			assert.Empty(t, auth.CurrentUserID(ctx))

			return c.NoContent(http.StatusOK)
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		echoRouter.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("logged in user", func(t *testing.T) {
		t.Parallel()

		echoRouter := newEnrichRouterToAssertOnHandler(func(c echo.Context) error {
			ctx := c.Request().Context()
			assert.True(t, auth.IsLoggedIn(ctx))
			assert.Equal(t, auth.UserID("1337"), auth.CurrentUserID(ctx))
			assert.True(t, auth.IsSuperuser(ctx))

			return c.NoContent(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(getSessionCookie(echoRouter))

		rec := httptest.NewRecorder()

		echoRouter.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func newEnsureRouterToAssertOnHandler(handler func(c echo.Context) error) *echo.Echo {
	echoRouter := echo.New()

	echoRouter.Use(session.Middleware(sessions.NewFilesystemStore("", []byte("secret"))))

	// endpoint to set an example cookie, that the middleware under test can work with.
	echoRouter.GET("/createSession", func(c echo.Context) error {
		sess, _ := session.Get(auth.SessionName, c)

		sess.Values[auth.SessKeyLoggedIn] = true
		sess.Values[auth.SessKeyUserID] = auth.UserID("1337")

		_ = sess.Save(c.Request(), c.Response())

		return c.NoContent(http.StatusOK)
	})

	authRoutes := echoRouter.Group("/auth")
	authRoutes.Use(auth.EnsureUserIsLoggedInMiddleware)
	authRoutes.GET("/", handler)

	return echoRouter
}

func newEnsureIsSuperuserRouterToAssertOnHandler(handler func(c echo.Context) error) *echo.Echo {
	echoRouter := echo.New()

	echoRouter.Use(session.Middleware(sessions.NewFilesystemStore("", []byte("secret"))))

	// endpoint to set an example cookie, that the middleware under test can work with.
	echoRouter.GET("/createSession", func(c echo.Context) error {
		sess, _ := session.Get(auth.SessionName, c)

		sess.Values[auth.SessKeyLoggedIn] = true
		sess.Values[auth.SessKeyUserID] = auth.UserID("1337")
		sess.Values[auth.SessKeyIsSuperuser] = true

		_ = sess.Save(c.Request(), c.Response())

		return c.NoContent(http.StatusOK)
	})

	authRoutes := echoRouter.Group("/admin")
	authRoutes.Use(auth.EnsureUserIsSuperuserMiddleware)
	authRoutes.GET("/", handler)

	return echoRouter
}

// newEnrichRouterToAssertOnHandler is a helper for unit tests, by returning a valid web router.
func newEnrichRouterToAssertOnHandler(handler func(c echo.Context) error) *echo.Echo {
	echoRouter := echo.New()

	echoRouter.Use(session.Middleware(sessions.NewFilesystemStore("", []byte("secret"))))
	echoRouter.Use(auth.EnrichCtxWithUserInfoMiddleware)
	echoRouter.GET("/", handler)

	// endpoint to set an example cookie, that the middleware under test can work with.
	echoRouter.GET("/createSession", func(c echo.Context) error {
		sess, _ := session.Get(auth.SessionName, c)

		sess.Values[auth.SessKeyLoggedIn] = true
		sess.Values[auth.SessKeyUserID] = auth.UserID("1337")
		sess.Values[auth.SessKeyIsSuperuser] = true

		_ = sess.Save(c.Request(), c.Response())

		return c.NoContent(http.StatusOK)
	})

	return echoRouter
}

// getSessionCookie calls the /createSession route setup in newEnrichRouterToAssertOnHandler.
func getSessionCookie(e *echo.Echo) *http.Cookie {
	req := httptest.NewRequest(http.MethodGet, "/createSession", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	response := rec.Result()
	defer response.Body.Close()

	return response.Cookies()[0]
}

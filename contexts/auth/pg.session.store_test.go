//go:build integration

package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/auth"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository/models"
	"github.com/go-arrower/arrower/tests"
)

var (
	ctx       = context.Background()
	pgHandler *tests.PostgresDocker
)

func TestMain(m *testing.M) {
	pgHandler = tests.GetPostgresDockerForIntegrationTestingInstance()

	//
	// Run tests
	code := m.Run()

	pgHandler.Cleanup()
	os.Exit(code)
}

func TestNewPGSessionStore(t *testing.T) {
	t.Parallel()

	t.Run("create fails", func(t *testing.T) {
		t.Parallel()

		ss, err := auth.NewPGSessionStore(nil, keyPairs)
		assert.Error(t, err)
		assert.Empty(t, ss)
	})

	t.Run("create succeeds", func(t *testing.T) {
		t.Parallel()

		pg := pgHandler.NewTestDatabase()

		ss, err := auth.NewPGSessionStore(pg, keyPairs)
		assert.NoError(t, err)
		assert.NotEmpty(t, ss)

		assert.NotEmpty(t, ss.Codecs)
		assert.NotEmpty(t, ss.Options)
	})
}

func TestPGSessionStore_New(t *testing.T) {
	t.Parallel()

	pg := pgHandler.NewTestDatabase()
	ss, _ := auth.NewPGSessionStore(pg, keyPairs)

	t.Run("save session with max age 0", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)

		sess, err := ss.New(req, auth.SessionName)
		assert.NoError(t, err)
		assert.NotEmpty(t, sess.ID)
	})

	t.Run("access a session that got already deleted (e.g. by a superuser)", func(t *testing.T) {
		t.Parallel()

		// setup
		queries := models.New(pg)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		sess0 := sessions.NewSession(ss, auth.SessionName)
		err := ss.Save(req, rec, sess0)
		assert.NoError(t, err)
		assert.NotEmpty(t, sess0.ID)

		result := rec.Result()
		defer result.Body.Close()

		req.AddCookie(result.Cookies()[0]) // set cookie of existing session for next http call

		err = queries.DeleteSessionByKey(ctx, []byte(sess0.ID))
		assert.NoError(t, err)

		// test
		sess1, err := ss.New(req, auth.SessionName)
		assert.NoError(t, err)
		assert.NotEmpty(t, sess1.ID)

		assert.Len(t, req.Cookies(), 2, "the original cookie and the one to overwrite the deletion. I am not sure what the browser or echo does with the two cookies")
		assert.Equal(t, 0, req.Cookies()[1].MaxAge, "cookie will be deleted by the browser, as session got deleted")
	})
}

func TestPGSessionStore_Save(t *testing.T) {
	t.Parallel()

	pg := pgHandler.NewTestDatabase()
	ss, _ := auth.NewPGSessionStore(pg, keyPairs)

	t.Run("save session with max age 0", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		session := sessions.NewSession(ss, auth.SessionName)
		session.Options = &sessions.Options{
			MaxAge: 0,
		}

		err := ss.Save(req, rec, session)
		assert.NoError(t, err)

		// assert db entry
		queries := models.New(pg)
		sessions, _ := queries.AllSessions(ctx)
		assert.Len(t, sessions, 1, "session with MaxAge 0, is deleted by browser on close")
	})
}

//nolint:tparallel,paralleltest // the tests depend on each other and the order is important.
func TestNewPGSessionStore_HTTPRequest(t *testing.T) {
	t.Parallel()

	pg := pgHandler.NewTestDatabase()
	echoRouter := newTestRouter(pg)

	var cookie *http.Cookie // the cookie to use over all requests

	t.Run("set initial cookie when surfing the site", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		echoRouter.ServeHTTP(rec, req)

		result := rec.Result()
		defer result.Body.Close()

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Body.String())

		cookie = result.Cookies()[0] // safe cookie for reuse later on

		// assert cookie
		assert.Len(t, result.Cookies(), 1)
		assert.Equal(t, "/", result.Cookies()[0].Path)
		assert.Equal(t, auth.SessionName, result.Cookies()[0].Name)
		assert.Equal(t, http.SameSiteStrictMode, result.Cookies()[0].SameSite)
		assert.Equal(t, 86400*30, result.Cookies()[0].MaxAge)

		// assert db entry
		queries := models.New(pg)
		sessions, _ := queries.AllSessions(ctx)

		assert.Len(t, sessions, 1)
		// cookie and session expire at the same time, allow 1 second of diff to make sure different granulates
		// in the representation like nanoseconds in pg are not an issue.
		assert.True(t, result.Cookies()[0].Expires.Sub(sessions[0].ExpiresAtUtc.Time) < 1)
		assert.Empty(t, sessions[0].UserID)
	})

	t.Run("session already exists => user logs in", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/login", nil)
		req.AddCookie(cookie) // use the cookie / session from the call before
		rec := httptest.NewRecorder()
		echoRouter.ServeHTTP(rec, req)

		result := rec.Result()
		defer result.Body.Close()
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Body.String())
		assert.Len(t, result.Cookies(), 1)

		// assert db entry
		queries := models.New(pg)
		sessions, _ := queries.AllSessions(ctx)

		assert.Len(t, sessions, 1)
		assert.NotEmpty(t, sessions[0].UserID)
		assert.Equal(t, userID, sessions[0].UserID.UUID)
	})

	t.Run("destroy session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/destroy", nil)
		req.AddCookie(cookie) // use the cookie / session from the call before
		rec := httptest.NewRecorder()
		echoRouter.ServeHTTP(rec, req)

		// assert cookie
		result := rec.Result()
		defer result.Body.Close()
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Body.String())
		assert.Len(t, result.Cookies(), 1)
		assert.Equal(t, "/", result.Cookies()[0].Path)
		assert.Equal(t, auth.SessionName, result.Cookies()[0].Name)
		assert.Equal(t, -1, result.Cookies()[0].MaxAge)

		// assert db entry
		queries := models.New(pg)
		sessions, _ := queries.AllSessions(ctx)

		assert.Len(t, sessions, 0)
	})
}

// --- --- --- TEST DATA --- --- ---

var (
	keyPairs = []byte("secret")
	userID   = uuid.New()
)

func newTestRouter(pg *pgxpool.Pool) *echo.Echo {
	ss, _ := auth.NewPGSessionStore(pg, keyPairs)
	echoRouter := echo.New()
	echoRouter.Use(session.Middleware(ss))

	queries := *models.New(pg)

	echoRouter.GET("/", func(c echo.Context) error {
		sess, err := session.Get(auth.SessionName, c)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		sess.Values["some-session"] = "some-value"

		err = sess.Save(c.Request(), c.Response())
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		return c.NoContent(http.StatusOK)
	})

	echoRouter.GET("/login", func(c echo.Context) error {
		sess, err := session.Get(auth.SessionName, c)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		sess.Values[auth.SessKeyUserID] = userID.String()

		err = sess.Save(c.Request(), c.Response())
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		// login is required to set the userID and userAgent for a session. This is done manually here
		err = queries.UpsertNewSession(c.Request().Context(), models.UpsertNewSessionParams{
			Key:       []byte(sess.ID),
			UserID:    uuid.NullUUID{UUID: uuid.MustParse(userID.String()), Valid: true},
			UserAgent: "arrower/1",
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		return c.NoContent(http.StatusOK)
	})

	echoRouter.GET("/destroy", func(c echo.Context) error {
		sess, err := session.Get(auth.SessionName, c)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		delete(sess.Values, auth.SessKeyUserID)

		sess.Options = &sessions.Options{
			Path:   "/",
			MaxAge: -1, // delete cookie immediately
		}

		err = sess.Save(c.Request(), c.Response())
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		return c.NoContent(http.StatusOK)
	})

	// seed db with example user
	_, _ = pg.Exec(ctx, `INSERT INTO auth.user (id, login) VALUES ($1, $2);`, userID, "login")

	return echoRouter
}

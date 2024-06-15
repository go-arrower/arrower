package auth

import (
	"context"
	"encoding/base32"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository/models"
)

var ErrSessionStoreFailed = errors.New("creating session store failed")

func NewPGSessionStore(pgx *pgxpool.Pool, keyPairs ...[]byte) (*PGSessionStore, error) {
	if pgx == nil {
		return nil, fmt.Errorf("missing postgres dependeny: %w", ErrSessionStoreFailed)
	}

	if err := pgx.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("%v: could not reach postgres: %w", ErrSessionStoreFailed, err) // todo check, if new go version can have multiple error verbs
	}

	queries := models.New(pgx)

	const oneMonth = 86400 * 30

	return &PGSessionStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:     "/",
			Domain:   "",
			MaxAge:   oneMonth,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		},
		queries: queries,
	}, nil
}

type PGSessionStore struct {
	queries *models.Queries

	Options *sessions.Options // default configuration
	Codecs  []securecookie.Codec
}

var _ sessions.Store = (*PGSessionStore)(nil)

// Get returns a session for the given name after adding it to the registry.
func (ss *PGSessionStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(ss, name) //nolint:wrapcheck // export session.Store errors, as caller expects it
}

// New returns a session for the given name without adding it to the registry.
func (ss *PGSessionStore) New(r *http.Request, name string) (*sessions.Session, error) { //nolint:varnamelen
	session := sessions.NewSession(ss, name)
	opts := *ss.Options
	session.Options = &opts
	session.IsNew = true
	session.ID = newSessionID()

	var err error
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, ss.Codecs...)
		if err == nil {
			data, err2 := ss.queries.FindSessionDataByKey(r.Context(), []byte(session.ID))
			if errors.Is(err2, pgx.ErrNoRows) {
				// session got deleted => remove cookie
				c.MaxAge = 0
				r.AddCookie(c)
			}

			if err2 == nil {
				err = securecookie.DecodeMulti(session.Name(), string(data), &session.Values, ss.Codecs...)
				if err == nil {
					session.IsNew = false
				}
			}
		}
	}

	return session, err //nolint:wrapcheck // export session.Store errors, as caller expects it
}

// Save adds a single session to the response.
//
// If the Options.MaxAge of the session is <= 0 then the session will be
// deleted from the db table. With this process it enforces the proper
// session cookie handling so no need to trust in the cookie management in the
// web browser.
func (ss *PGSessionStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error { //nolint:varnamelen,lll
	// Delete if max-age is < 0, if max-age == 0 the cookie will delete ones the browser closes
	if session.Options.MaxAge < 0 {
		if err := ss.queries.DeleteSessionByKey(r.Context(), []byte(session.ID)); err != nil {
			return fmt.Errorf("%w", err)
		}

		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))

		return nil
	}

	if session.ID == "" {
		session.ID = newSessionID()
	}

	if err := ss.save(r.Context(), session); err != nil {
		return err
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, ss.Codecs...)
	if err != nil {
		return err //nolint:wrapcheck // export session.Store errors, as caller expects it
	}

	http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))

	return nil
}

func (ss *PGSessionStore) save(ctx context.Context, session *sessions.Session) error {
	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, ss.Codecs...)
	if err != nil {
		return err //nolint:wrapcheck // export session.Store errors, as caller expects it
	}

	err = ss.queries.UpsertSessionData(ctx, models.UpsertSessionDataParams{
		Key:  []byte(session.ID),
		Data: []byte(encoded),
		ExpiresAtUtc: pgtype.Timestamptz{
			Time:             time.Now().Add(time.Second * time.Duration(session.Options.MaxAge)),
			Valid:            true,
			InfinityModifier: pgtype.Finite,
		},
	})
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func newSessionID() string {
	const keyLength = 32

	return strings.TrimRight(
		base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(keyLength)),
		"=",
	)
}

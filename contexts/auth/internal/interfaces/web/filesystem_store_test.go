package web_test

import (
	"encoding/base32"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// This is a copy of the store from the gorillas package.
// The purpose is to work around the bad behaviour in persisting sessions with MaxAge 0,
// see: https://github.com/gorilla/sessions/issues/267
// Ones the issue is fixed, this file should be deleted and the tests refactored to use the gorillas implementation.
// The fix is done in line 115.
// Line 86ff. is also new behaviour, to make sessions work better with the use case of accessing them already in the application layer (login, register).
// FilesystemStore ------------------------------------------------------------

var fileMutex sync.RWMutex

// NewFilesystemStore returns a new FilesystemStore.
//
// The path argument is the directory where sessions will be saved. If empty
// it will use os.TempDir().
//
// See NewCookieStore() for a description of the other parameters.
func NewFilesystemStore(path string, keyPairs ...[]byte) *FilesystemStore {
	if path == "" {
		path = os.TempDir()
	}
	fs := &FilesystemStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
		path: path,
	}

	fs.MaxAge(fs.Options.MaxAge)
	return fs
}

// FilesystemStore stores sessions in the filesystem.
//
// It also serves as a reference for custom stores.
//
// This store is still experimental and not well tested. Feedback is welcome.
type FilesystemStore struct {
	Codecs  []securecookie.Codec
	Options *sessions.Options // default configuration
	path    string
}

// MaxLength restricts the maximum length of new sessions to l.
// If l is 0 there is no limit to the size of a session, use with caution.
// The default for a new FilesystemStore is 4096.
func (s *FilesystemStore) MaxLength(l int) {
	for _, c := range s.Codecs {
		if codec, ok := c.(*securecookie.SecureCookie); ok {
			codec.MaxLength(l)
		}
	}
}

// Get returns a session for the given name after adding it to the registry.
//
// See CookieStore.Get().
func (s *FilesystemStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// New returns a session for the given name without adding it to the registry.
//
// See CookieStore.New().
func (s *FilesystemStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(s, name)
	opts := *s.Options
	session.Options = &opts
	session.IsNew = true

	const keyLength = 32
	session.ID = strings.TrimRight(
		base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(keyLength)),
		"=",
	)

	var err error
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...)
		if err == nil {
			err = s.load(session)
			if err == nil {
				session.IsNew = false
			}
		}
	}
	return session, err
}

// Save adds a single session to the response.
//
// If the Options.MaxAge of the session is <= 0 then the session file will be
// deleted from the store path. With this process it enforces the properly
// session cookie handling so no need to trust in the cookie management in the
// web browser.
func (s *FilesystemStore) Save(r *http.Request, w http.ResponseWriter,
	session *sessions.Session) error {
	// Delete if max-age is < 0
	if session.Options.MaxAge < 0 { // THIS IS THE FIX: CHANGED IT FROM <= TO <
		if err := s.erase(session); err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	if session.ID == "" {
		// Because the ID is used in the filename, encode it to
		// use alphanumeric characters only.
		session.ID = strings.TrimRight(
			base32.StdEncoding.EncodeToString(
				securecookie.GenerateRandomKey(32)), "=")
	}
	if err := s.save(session); err != nil {
		return err
	}
	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID,
		s.Codecs...)
	if err != nil {
		return err
	}
	http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))
	return nil
}

// MaxAge sets the maximum age for the store and the underlying cookie
// implementation. Individual sessions can be deleted by setting Options.MaxAge
// = -1 for that session.
func (s *FilesystemStore) MaxAge(age int) {
	s.Options.MaxAge = age

	// Set the maxAge for each securecookie instance.
	for _, codec := range s.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

// save writes encoded session.Values to a file.
func (s *FilesystemStore) save(session *sessions.Session) error {
	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values,
		s.Codecs...)
	if err != nil {
		return err
	}
	filename := filepath.Join(s.path, "session_"+session.ID)
	fileMutex.Lock()
	defer fileMutex.Unlock()
	return ioutil.WriteFile(filename, []byte(encoded), 0600)
}

// load reads a file and decodes its content into session.Values.
func (s *FilesystemStore) load(session *sessions.Session) error {
	filename := filepath.Join(s.path, "session_"+session.ID)
	fileMutex.RLock()
	defer fileMutex.RUnlock()
	fdata, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	if err = securecookie.DecodeMulti(session.Name(), string(fdata),
		&session.Values, s.Codecs...); err != nil {
		return err
	}
	return nil
}

// delete session file
func (s *FilesystemStore) erase(session *sessions.Session) error {
	filename := filepath.Join(s.path, "session_"+session.ID)

	fileMutex.RLock()
	defer fileMutex.RUnlock()

	err := os.Remove(filename)
	return err
}

// Package secret contains secrets to use in the application.
//
// Its purpose is to easily deal with sensitive data
// you want to keep from accidentally being exposed.
package secret

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

func New(secret string) Secret {
	return Secret{secret: &secret}
}

// Secret prevents accidentally exposing
// any data you did not want to expose by masking it.
type Secret struct {
	// secret being a pointer does make it harder to access the value.
	// It is still possible by directly accessing the memory address.
	// See the example on how it would work.
	secret *string
}

// Secret returns the actual value of the Secret.
func (s Secret) Secret() string {
	if s == *new(Secret) {
		return ""
	}

	return *s.secret
}

func (s Secret) String() string {
	return "******"
}

func (s Secret) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String()) //nolint:wrapcheck // export the underlying error
}

func (s *Secret) UnmarshalJSON(data []byte) error {
	var des string
	if err := json.Unmarshal(data, &des); err != nil {
		return err //nolint:wrapcheck // export the underlying error
	}

	s.secret = &des

	return nil
}

func (s Secret) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *Secret) UnmarshalText(data []byte) error {
	text := string(data)
	s.secret = &text

	return nil
}

func (s *Secret) Scan(value interface{}) error {
	if value == nil {
		*s.secret = ""
		return nil
	}

	strValue, ok := value.(string)
	if !ok {
		return errors.New("failed to scan Secret: value is not a string")
	}

	s.secret = &strValue
	return nil
}

func (s Secret) Value() (driver.Value, error) {
	return s.secret, nil
}

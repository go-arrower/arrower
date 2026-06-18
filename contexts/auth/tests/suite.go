//go:build e2e

package tests

import (
	"testing"

	"github.com/go-arrower/arrower/e2e"
)

type Suite struct {
	*e2e.Suite
	t *testing.T
}

func Test(t *testing.T) *Suite {
	return &Suite{
		Suite: e2e.Test(t),
		t:     t,
	}
}

func (s *Suite) RegisterUser(name string, password string) {
	s.t.Helper()

	page := s.Goto("/auth/register").Form("/auth/register").Submit(map[string]any{
		"login":                 name,
		"password":              password,
		"password_confirmation": password,
		"tos":                   "true",
	})
	page.NotContains("Register", "should not contain register form")
}

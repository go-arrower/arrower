//go:build e2e

package e2e

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/go-arrower/arrower"
	"github.com/go-arrower/arrower/postgres"
)

const (
	defaultRetryCount = 5
)

// Suite is the entry point for e2e tests. Call e2e.Test(t) to create one.
//
// Navigation model:
//   - Suite.Goto() think a fresh browser tab (new client, no cookies).
//   - Page.Goto() navigates within the same tab (preserves cookies, same session).
type Suite struct {
	t   *testing.T
	c   *req.Client
	PGx *pgxpool.Pool
}

type SuiteOption func(*Suite)

func WithBaseURL(url string) SuiteOption {
	return func(s *Suite) {
		s.c = s.c.SetBaseURL(url)
	}
}

func Test(t *testing.T, ops ...SuiteOption) *Suite {
	t.Helper()
	assertTestCaseNamingConvention(t)

	suite := &Suite{
		t:   t,
		PGx: connectToDatabaseOrNil(t.Context(), getProjectRoot()+"/tests/e2e/test.config.yaml"),
		c: req.C().
			SetBaseURL("http://localhost:8080").
			SetCommonRetryCount(defaultRetryCount).
			SetCommonRetryHook(func(_ *req.Response, err error) {
				t.Log("request failed with: " + err.Error())
			}),
	}

	for _, op := range ops {
		op(suite)
	}

	return suite
}

type GotoOption func(*req.Client) *req.Client

func WithRedirectPolicy(policy req.RedirectPolicy) func(*req.Client) *req.Client {
	return func(c *req.Client) *req.Client {
		return c.SetRedirectPolicy(policy)
	}
}

func WithHeaders(headers map[string]string) func(*req.Client) *req.Client {
	return func(c *req.Client) *req.Client {
		return c.SetCommonHeaders(headers)
	}
}

func (s *Suite) Goto(url string, opts ...GotoOption) Page {
	client := s.c.Clone() // clone to prevent changes to further Goto calls.
	for _, opt := range opts {
		client = opt(client)
	}

	resp, err := client.R().Get(url)
	p := NewPage(s.t, client, resp, err)

	return p
}

func (s *Suite) Get(url string) Document {
	resp, err := s.c.Clone().R().Get(url)
	return NewJSON(s.t, s.c, resp, err)
}

func (s *Suite) Post(url string, body any) Document {
	resp, err := s.c.Clone().R().
		SetContentType("application/json").
		SetBody(body).
		Post(url)

	return NewJSON(s.t, s.c, resp, err)
}

func (s *Suite) Put(url string, body any) Document {
	resp, err := s.c.Clone().R().
		SetContentType("application/json").
		SetBody(body).
		Put(url)

	return NewJSON(s.t, s.c, resp, err)
}

func (s *Suite) Patch(url string, body any) Document {
	resp, err := s.c.Clone().R().
		SetContentType("application/json").
		SetBody(body).
		Patch(url)

	return NewJSON(s.t, s.c, resp, err)
}

func (s *Suite) Delete(url string, body ...any) Document {
	req := s.c.Clone().R()
	if len(body) > 0 {
		req = req.
			SetContentType("application/json").
			SetBody(body[0])
	}

	resp, err := req.Delete(url)

	return NewJSON(s.t, s.c, resp, err)
}

type downloadConfig struct {
	method   string
	formData map[string]any
}

type DownloadOption func(*downloadConfig)

// WithFormData makes the download request a POST request and submits the given data.
func WithFormData(data map[string]any) DownloadOption {
	return func(cfg *downloadConfig) {
		cfg.method = http.MethodPost
		cfg.formData = data
	}
}

// Download makes a GET request and returns a Download for asserting on binary responses.
// It clones the client (fresh session, no cookies).
func (s *Suite) Download(url string, opts ...DownloadOption) Download {
	cfg := downloadConfig{method: http.MethodGet, formData: map[string]any{}}
	for _, opt := range opts {
		opt(&cfg)
	}

	var (
		resp *req.Response
		err  error
	)

	req := s.c.Clone().R()

	switch cfg.method {
	case http.MethodPost:
		req.SetFormDataAnyType(cfg.formData)
		resp, err = req.Post(url)
	default:
		resp, err = req.Get(url)
	}

	return NewDownload(s.t, s.c, resp, err)
}

func connectToDatabaseOrNil(ctx context.Context, cfgFile string) *pgxpool.Pool {
	if _, err := os.Stat(cfgFile); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	configData, err := getTestConfig(cfgFile)
	if err != nil {
		panic(err)
	}

	pgx, err := postgres.Connect(ctx, postgres.Config{
		Host:       configData.Postgres.Host,
		Port:       configData.Postgres.Port,
		User:       configData.Postgres.User,
		Password:   configData.Postgres.Password.Secret(),
		Database:   configData.Postgres.Database,
		MaxConns:   configData.Postgres.MaxConns,
		Migrations: nil, // no migrations on purpose: connect to running e2e test, not modify app
		SSLMode:    "disable",
	}, noop.NewTracerProvider())
	if err != nil {
		panic(err)
	}

	return pgx.PGx
}

// Go tests are called from the path of the calling package. This changes the current directory for each test and will
// make it impossible to find relative test data.
// This helper will find the files independent of the caller's path.
//
//lint:ignore U1000 kept for future use
func getProjectRoot() string {
	cmdOut, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(cmdOut)) + "/"
}

// getTestConfig loads the configuration file.
func getTestConfig(cfgFile string) (arrower.Config, error) {
	vip := arrower.DefaultViper()
	if cfgFile != "" {
		vip.SetConfigFile(cfgFile)
	}

	if err := vip.ReadInConfig(); err != nil {
		return arrower.Config{}, fmt.Errorf("could not read config %q: %w", cfgFile, err)
	}

	cfg := arrower.Config{}
	if err := vip.Unmarshal(&cfg); err != nil {
		return arrower.Config{}, fmt.Errorf("could not unmarshal into configuration: %w", err)
	}

	return cfg, nil
}

func assertTestCaseNamingConvention(t *testing.T) {
	t.Helper()
	followsNamingConvention := strings.HasPrefix(t.Name(), "TestHelper_") ||
		strings.HasPrefix(t.Name(), "TestScenario_") ||
		strings.HasPrefix(t.Name(), "TestAssert_")
	assert.True(t, followsNamingConvention, "Name of the test case `"+t.Name()+"` does not follow the naming convention. Prefix should be TestHelper_, TestScenario_ or TestAssert_") //nolint:lll
}

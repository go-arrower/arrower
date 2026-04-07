package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"
)

func Test(
	t *testing.T,
	viewFS fs.FS,
	funcMap template.FuncMap,
) (*TestRenderer, error) {
	renderer, err := New(slog.New(slog.DiscardHandler), noop.NewTracerProvider(), viewFS, funcMap, false)
	if err != nil {
		return &TestRenderer{}, fmt.Errorf("could not create test renderer: %w", err)
	}

	return &TestRenderer{
		t:           t,
		mu:          sync.Mutex{},
		renderer:    renderer,
		shipResults: ping(), // todo add test case for this setting
	}, nil
}

type TestRenderer struct {
	t        *testing.T
	mu       sync.Mutex
	renderer *Renderer

	shipResults bool
}

func (r *TestRenderer) Render(
	w io.Writer,
	context string,
	name string,
	data interface{},
) (*TestRendererAssertions, error) {
	r.t.Helper()

	r.mu.Lock()

	rawTemplate := ""
	r.renderer.rawTemplate = func(t *template.Template) {
		rawTemplate = t.Tree.Root.String()
	}

	buf := &bytes.Buffer{}

	err := r.renderer.Render(r.t.Context(), buf, context, name, data)
	if err != nil {
		return &TestRendererAssertions{}, err
	}

	_, err = io.Copy(w, buf)
	if err != nil {
		return &TestRendererAssertions{}, err
	}

	r.mu.Unlock()

	testName := r.t.Name()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return &TestRendererAssertions{}, fmt.Errorf("could not marshal data: %w", err)
	}

	run := testRun{ // include: context
		TestName: testName,
		RunName:  time.Now().Format(time.TimeOnly),
		Template: template.HTML(rawTemplate), //nolint:gosec // html is trusted and renders in the test viewer
		Data:     string(jsonData),
		HTML:     template.HTML(buf.String()), //nolint:gosec // html is trusted and renders in the test viewer
	}

	if r.shipResults {
		postData, err := json.Marshal(run)
		if err != nil {
			return &TestRendererAssertions{}, fmt.Errorf("could not marshal post body: %w", err)
		}

		req, err := http.NewRequestWithContext(r.t.Context(), http.MethodPost, "http://localhost:3030/testcase", bytes.NewBuffer(postData))
		if err != nil {
			return &TestRendererAssertions{}, fmt.Errorf("could not build request to send testcase: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			return &TestRendererAssertions{}, fmt.Errorf("could not send testcase: %w", err)
		}

		resp.Body.Close()
	}

	return &TestRendererAssertions{
		run:         run,
		t:           r.t,
		shipResults: r.shipResults,
	}, nil
}

func TestEcho(
	t *testing.T,
	echo *echo.Echo,
	viewFS fs.FS,
	funcs template.FuncMap,
) *TestEchoRenderer {
	if viewFS == nil {
		viewFS = fstest.MapFS{}
	}

	mergedFM := template.FuncMap{
		"route": echo.Reverse,
	}

	for name, fn := range funcs {
		mergedFM[name] = fn
	}

	renderer, err := Test(t, viewFS, mergedFM)
	assert.NoError(t, err)

	return &TestEchoRenderer{TestRenderer: renderer}
}

// EchoRenderer is a wrapper that makes the Renderer available for the echo router: https://echo.labstack.com/
type TestEchoRenderer struct {
	*TestRenderer
}

var _ echo.Renderer = (*TestEchoRenderer)(nil)

func (r *TestEchoRenderer) Render(w io.Writer, templateName string, data interface{}, c echo.Context) error {
	_, _, context := r.isRegisteredContext(c) // todo test how it is split

	_, err := r.TestRenderer.Render(w, context, templateName, data)

	return err
}

// todo: see comments in EchoRenderer method of this.
func (r *TestEchoRenderer) isRegisteredContext(c echo.Context) (bool, bool, string) {
	paths := strings.Split(c.Path(), "/")

	isAdmin := false

	for _, path := range paths {
		if path == "" {
			continue
		}

		if path == adminPathPrefix {
			isAdmin = true

			continue
		}

		_, exists := r.renderer.views[path]
		if exists {
			if isAdmin {
				return true, true, "/" + adminPathPrefix + "/" + path
			}

			return true, false, path
		}
	}

	if isAdmin {
		return true, true, adminPathPrefix // todo return normal context name, as the flag isAdmin is returned already
	}

	return false, false, ""
}

// ping is a simplistic check if the developer has `arrower run` open locally.
// If URL is available, test cases are shipped for inspection in the arrower UI.
func ping() bool {
	url := "http://localhost:3030/testcase"

	resp, err := http.Head(url) //nolint:noctx // ignore as this is just a ping for local testing
	if err != nil {
		return false
	}

	resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

type testRun struct {
	TestName string `json:"testName"`

	RunName  string        `json:"runName"`
	Template template.HTML `json:"template"`
	Data     string        `json:"data"`
	HTML     template.HTML `json:"html"`
}

// TestRendererAssertions is a helper that exposes a lot of TestRenderer-specific assertions for the use in tests.
// The interface follows stretchr/testify as close as possible.
//
//   - Every assert func returns a bool indicating whether the assertion was successful or not,
//     this is useful for if you want to go on making further assertions under certain conditions.
type TestRendererAssertions struct {
	t   *testing.T
	run testRun

	shipResults bool
}

// NotEmpty asserts that the html is not empty.
func (a *TestRendererAssertions) NotEmpty(msgAndArgs ...any) bool {
	a.t.Helper()

	ass := assertion{
		TestName: a.run.TestName,
		RunName:  a.run.RunName,
		Name:     "NotEmpty",
		Args:     append([]any{}, msgAndArgs...),
		Pass:     false,
	}

	if len(a.run.HTML) == 0 {
		a.sendAssertion(ass) // todo can I use defer?

		return assert.Fail(a.t, "html is empty, should not be", msgAndArgs...)
	}

	ass.Pass = true

	if a.shipResults {
		a.sendAssertion(ass)
	}

	return true
}

func (a *TestRendererAssertions) Contains(contains any, msgAndArgs ...any) bool {
	a.t.Helper()

	ass := assertion{
		TestName: a.run.TestName,
		RunName:  a.run.RunName,
		Name:     "Contains",
		Args:     append(append([]any{}, contains), msgAndArgs...),
		Pass:     false,
	}

	pass := assert.Contains(a.t, a.run.HTML, contains, msgAndArgs)
	if !pass {
		a.sendAssertion(ass)
		return pass
	}

	ass.Pass = true
	a.sendAssertion(ass)

	return pass
}

type assertion struct {
	TestName string `json:"testName"`
	RunName  string `json:"runName"`

	Name string `json:"name"`
	Args []any  `json:"args"`
	Pass bool   `json:"pass"`
}

func (a *TestRendererAssertions) sendAssertion(assertion assertion) {
	if !a.shipResults {
		return
	}

	postData, err := json.Marshal(assertion)
	if err != nil {
		panic("could not marshal post body: " + err.Error())
	}

	req, err := http.NewRequestWithContext(a.t.Context(), http.MethodPost, "http://localhost:3030/testcase", bytes.NewBuffer(postData))
	if err != nil {
		panic("could not build request to send testcase: " + err.Error())
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		panic("could not send testcase: " + err.Error())
	}
	resp.Body.Close() //nolint:wsl_v5
}

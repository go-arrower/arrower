package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"
)

func Test(
	viewFS fs.FS,
	funcMap template.FuncMap,
) (*TestRenderer, error) {
	renderer, err := New(slog.New(slog.DiscardHandler), noop.NewTracerProvider(), viewFS, funcMap, false)
	if err != nil {
		return &TestRenderer{}, fmt.Errorf("could not create test renderer: %w", err)
	}

	return &TestRenderer{
		mu:          sync.Mutex{},
		renderer:    renderer,
		shipResults: ping(), // todo add test case for this setting
	}, nil
}

type TestRenderer struct {
	mu       sync.Mutex
	renderer *Renderer

	shipResults bool
}

func (r *TestRenderer) Render(
	t *testing.T,
	context string,
	name string,
	data interface{},
) (*RendererAssertions, error) {
	t.Helper()

	r.mu.Lock()

	rawTemplate := ""
	r.renderer.rawTemplate = func(t *template.Template) {
		rawTemplate = t.Tree.Root.String()
	}

	buf := &bytes.Buffer{}

	err := r.renderer.Render(t.Context(), buf, context, name, data)
	if err != nil {
		return &RendererAssertions{}, err
	}

	r.mu.Unlock()

	testName := t.Name()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return &RendererAssertions{}, fmt.Errorf("could not marshal data: %w", err)
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
			return &RendererAssertions{}, fmt.Errorf("could not marshal post body: %w", err)
		}

		resp, err := http.Post("http://localhost:3030/testcase", "application/json", bytes.NewBuffer(postData))
		if err != nil {
			return &RendererAssertions{}, fmt.Errorf("could not send testcase: %w", err)
		}
		resp.Body.Close() //nolint:wsl_v5
	}

	return &RendererAssertions{
		run:         run,
		t:           t,
		shipResults: r.shipResults,
	}, nil
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

// RendererAssertions is a helper that exposes a lot of TestRenderer-specific assertions for the use in tests.
// The interface follows stretchr/testify as close as possible.
//
//   - Every assert func returns a bool indicating whether the assertion was successful or not,
//     this is useful for if you want to go on making further assertions under certain conditions.
type RendererAssertions struct {
	t   *testing.T
	run testRun

	shipResults bool
}

// NotEmpty asserts that the html is not empty.
func (a *RendererAssertions) NotEmpty(msgAndArgs ...any) bool {
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

func (a *RendererAssertions) Contains(contains any, msgAndArgs ...any) bool {
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

func (a *RendererAssertions) sendAssertion(assertion assertion) {
	if !a.shipResults {
		return
	}

	postData, err := json.Marshal(assertion)
	if err != nil {
		panic("could not marshal post body: " + err.Error())
	}

	resp, err := http.Post("http://localhost:3030/testcase/assertion", "application/json", bytes.NewBuffer(postData))
	if err != nil {
		panic("could not send testcase: " + err.Error())
	}
	resp.Body.Close() //nolint:wsl_v5
}

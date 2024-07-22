package renderer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/go-arrower/arrower/alog"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"
)

func Test(
	viewFS fs.FS,
	funcMap template.FuncMap,
) (*TestRenderer, error) {
	r, err := New(alog.NewNoop(), noop.NewTracerProvider(), viewFS, funcMap, false)
	if err != nil {
		return &TestRenderer{}, fmt.Errorf("could not create test renderer: %w", err)
	}

	return &TestRenderer{
		mu:          sync.Mutex{},
		renderer:    r,
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
	ctx context.Context,
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

	err := r.renderer.Render(ctx, buf, context, name, data)
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

		_, err = http.Post("http://localhost:3030/testcase", "application/json", bytes.NewBuffer(postData))
		if err != nil {
			return &RendererAssertions{}, fmt.Errorf("could not send testcase: %w", err)
		}
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
	TestName string

	RunName  string
	Template template.HTML
	Data     string
	HTML     template.HTML
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
	TestName string
	RunName  string

	Name string
	Args []any
	Pass bool
}

func (a *RendererAssertions) sendAssertion(assertion assertion) {
	if !a.shipResults {
		return
	}

	postData, err := json.Marshal(assertion)
	if err != nil {
		panic("could not marshal post body: " + err.Error())
	}

	_, err = http.Post("http://localhost:3030/testcase/assertion", "application/json", bytes.NewBuffer(postData))
	if err != nil {
		panic("could not send testcase: " + err.Error())
	}
}

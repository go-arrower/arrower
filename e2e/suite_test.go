//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/e2e"
)

func TestTest(t *testing.T) {
	t.Parallel()

	svr := server()
	defer svr.Close()

	suite := e2e.Test(new(testing.T))
	assert.NotEmpty(t, suite)

	page := suite.Goto(svr.URL)
	page.IsOK()

	suite = e2e.Test(new(testing.T), e2e.WithBaseURL("http://localhost:3000"))
	page = suite.Goto(svr.URL)
	page.IsNotFound()
}

func TestSuite_Goto(t *testing.T) {
	t.Parallel()

	t.Run("with headers", func(t *testing.T) {
		t.Parallel()

		var captured capturedRequest

		svr := server(withCapture(&captured))
		defer svr.Close()

		e2e.Test(new(testing.T)).Goto(svr.URL, e2e.WithHeaders(map[string]string{
			"X-Custom": "test-value",
		}))

		assert.Equal(t, "test-value", captured.headers.Get("X-Custom"))
	})
}

func TestSuite_Get(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		json string
	}{
		"empty": {``},
		"obj":   {`{}`},
		"array": {`[]`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withJSON(tt.json))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Get(svr.URL)
			page.IsOK()
		})
	}
}

func TestSuite_Delete(t *testing.T) {
	t.Parallel()

	t.Run("with body", func(t *testing.T) {
		t.Parallel()

		var captured capturedRequest

		svr := server(withCapture(&captured))
		defer svr.Close()

		doc := e2e.Test(new(testing.T)).Delete(svr.URL, map[string]any{"id": "123"})
		doc.IsOK()
		assert.Contains(t, captured.body.Encode(), "123")
		assert.Contains(t, captured.headers.Get("Content-Type"), "application/json")
	})
}

func TestSuite_Download(t *testing.T) {
	t.Parallel()

	t.Run("happy path: returns body, status and headers", func(t *testing.T) {
		t.Parallel()

		svr := server(
			withHeader("Content-Type", "application/pdf"),
			withHeader("Content-Disposition", "attachment; filename=report.pdf"),
			withHTML("some binary content"),
		)
		defer svr.Close()

		dl := e2e.Test(new(testing.T)).Download(svr.URL)

		assert.NotEmpty(t, dl.Bytes())
		dl.Header("Content-Type").Contains("pdf")
		dl.Header("Content-Disposition").Contains("filename=report.pdf")
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		svr := server(withStatus(http.StatusNotFound))
		defer svr.Close()

		dl := e2e.Test(new(testing.T)).Download(svr.URL)

		dl.IsNotFound()
	})

	t.Run("with form data sends POST", func(t *testing.T) {
		t.Parallel()

		var captured capturedRequest

		svr := server(
			withCapture(&captured),
			withHeader("Content-Type", "text/csv"),
			withHeader("Content-Disposition", "attachment; filename=report.csv"),
			withHTML("id,name\n1,Alice\n"),
		)
		defer svr.Close()

		dl := e2e.Test(new(testing.T)).Download(svr.URL,
			e2e.WithFormData(map[string]any{
				"format": "csv",
				"year":   "2024",
			}),
		)

		assert.Equal(t, http.MethodPost, captured.method)
		assert.Equal(t, "csv", captured.body.Get("format"))
		assert.Equal(t, "2024", captured.body.Get("year"))

		assert.Contains(t, string(dl.Bytes()), "id,name")
		dl.Header("Content-Type").Contains("text/csv")
		dl.Header("Content-Disposition").Contains("report.csv")
	})
}

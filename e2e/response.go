//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/assert"
)

// response is the shared base for Page and Document.
// It holds the HTTP response and provides common assertions for status codes,
// headers, cookies, and body content.
type response struct {
	t        *testing.T
	client   *req.Client
	httpResp *req.Response
}

func (r response) Contains(text string, msgAndArgs ...any) bool {
	r.t.Helper()

	if strings.Contains(r.httpResp.String(), text) {
		return true
	}

	return fail(r.t, "response does not contain expected text", []kv{
		{"expected", fmt.Sprintf("%q", text)},
		{"body", truncate(r.httpResp.String(), 200)},
	}, msgAndArgs...)
}

func (r response) NotContains(text string, msgAndArgs ...any) bool {
	r.t.Helper()

	if strings.Contains(r.httpResp.String(), text) {
		return fail(r.t, "response contains text but should not", []kv{
			{"text", fmt.Sprintf("%q", text)},
		}, msgAndArgs...)
	}

	return true
}

func (r response) StatusCode(code int, msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.StatusCode != code {
		return fail(r.t, "status code mismatch", []kv{
			{"actual", fmt.Sprintf("%d (%s)", r.httpResp.StatusCode, http.StatusText(r.httpResp.StatusCode))},
			{"expected", fmt.Sprintf("%d (%s)", code, http.StatusText(code))},
		}, msgAndArgs...)
	}

	return true
}

func (r response) IsOK(msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.StatusCode != http.StatusOK {
		return fail(r.t, "status code mismatch", []kv{
			{"actual", fmt.Sprintf("%d (%s)", r.httpResp.StatusCode, http.StatusText(r.httpResp.StatusCode))},
			{"expected", "200 (OK)"},
		}, msgAndArgs...)
	}

	return true
}

func (r response) IsCreated(msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.StatusCode != http.StatusCreated {
		return fail(r.t, "status code mismatch", []kv{
			{"actual", fmt.Sprintf("%d (%s)", r.httpResp.StatusCode, http.StatusText(r.httpResp.StatusCode))},
			{"expected", "201 (Created)"},
		}, msgAndArgs...)
	}

	return true
}

func (r response) IsNotFound(msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.StatusCode != http.StatusNotFound {
		return fail(r.t, "status code mismatch", []kv{
			{"actual", fmt.Sprintf("%d (%s)", r.httpResp.StatusCode, http.StatusText(r.httpResp.StatusCode))},
			{"expected", "404 (Not Found)"},
		}, msgAndArgs...)
	}

	return true
}

func (r response) IsUnauthorized(msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.StatusCode != http.StatusUnauthorized {
		return fail(r.t, "status code mismatch", []kv{
			{"actual", fmt.Sprintf("%d (%s)", r.httpResp.StatusCode, http.StatusText(r.httpResp.StatusCode))},
			{"expected", "401 (Unauthorized)"},
		}, msgAndArgs...)
	}

	return true
}

func (r response) IsForbidden(msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.StatusCode != http.StatusForbidden {
		return fail(r.t, "status code mismatch", []kv{
			{"actual", fmt.Sprintf("%d (%s)", r.httpResp.StatusCode, http.StatusText(r.httpResp.StatusCode))},
			{"expected", "403 (Forbidden)"},
		}, msgAndArgs...)
	}

	return true
}

func (r response) HasCookie(name string, msgAndArgs ...any) bool {
	r.t.Helper()

	cookies, err := r.client.GetCookies(r.httpResp.Request.URL.String())
	assert.NoError(r.t, err, "failed to get cookies from client jar")

	for i := range cookies {
		if cookies[i].Name == name {
			return true
		}
	}

	return fail(r.t, "response does not have cookie", []kv{
		{"cookie", name},
		{"cookies", formatList(cookieNames(cookies))},
	}, msgAndArgs...)
}

func (r response) HasCookies(names []string, msgAndArgs ...any) bool {
	r.t.Helper()

	cookies, err := r.client.GetCookies(r.httpResp.Request.URL.String())
	assert.NoError(r.t, err, "failed to get cookies from client jar")

	if len(cookies) == 0 && len(names) > 0 {
		return fail(r.t, "response has no cookies", []kv{
			{"expected", fmt.Sprintf("%v", names)},
		}, msgAndArgs...)
	}

	var notFound []string

	for _, name := range names {
		var found bool

		for i := range cookies {
			if cookies[i].Name == name {
				found = true
				continue
			}
		}

		if !found {
			notFound = append(notFound, name)
		}
	}

	if len(notFound) > 0 {
		return fail(r.t, "response is missing cookies", []kv{
			{"missing", fmt.Sprintf("%v", notFound)},
		}, msgAndArgs...)
	}

	return true
}

func (r response) HasCookieTotal(total int, msgAndArgs ...any) bool {
	r.t.Helper()

	cookies, err := r.client.GetCookies(r.httpResp.Request.URL.String())
	assert.NoError(r.t, err, "failed to get cookies from client jar")

	if len(cookies) != total {
		return fail(r.t, "cookie count mismatch", []kv{
			{"actual", fmt.Sprintf("%d", len(cookies))},
			{"expected", fmt.Sprintf("%d", total)},
		}, msgAndArgs...)
	}

	return true
}

func (r response) IsPath(path string, msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.Response.Request.URL.Path != path {
		return fail(r.t, "path mismatch", []kv{
			{"actual", fmtEmpty(r.httpResp.Response.Request.URL.Path)},
			{"expected", path},
		}, msgAndArgs...)
	}

	return true
}

func (r response) URL() string {
	return r.httpResp.Response.Request.URL.String()
}

func (r response) ContentType(ctype string, msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.Header.Get("Content-Type") != ctype {
		return fail(r.t, "content type mismatch", []kv{
			{"actual", fmtEmpty(r.httpResp.Header.Get("Content-Type"))},
			{"expected", ctype},
		}, msgAndArgs...)
	}

	return true
}

func (r response) Header(key string) Header {
	return Header{key: key, resp: r}
}

type Header struct {
	resp response
	key  string
}

func (h Header) Exists(msgAndArgs ...any) bool {
	h.resp.t.Helper()

	key := textproto.CanonicalMIMEHeaderKey(h.key)
	_, exists := h.resp.httpResp.Header[key]

	if !exists {
		return fail(h.resp.t, "header does not exist", []kv{
			{"header", h.key},
			{"headers", formatList(headerKeys(h.resp.httpResp.Header))},
		}, msgAndArgs...)
	}

	return true
}

func (h Header) NotEmpty(msgAndArgs ...any) bool {
	h.resp.t.Helper()

	if h.resp.httpResp.Header.Get(h.key) == "" {
		return fail(h.resp.t, "header is empty but should not be", []kv{
			{"header", h.key},
		}, msgAndArgs...)
	}

	return true
}

func (h Header) Is(value string, msgAndArgs ...any) bool {
	h.resp.t.Helper()

	key := textproto.CanonicalMIMEHeaderKey(h.key)
	_, exists := h.resp.httpResp.Header[key]
	actual := h.resp.httpResp.Header.Get(h.key)

	if actual != value || !exists {
		return fail(h.resp.t, "header value mismatch", []kv{
			{"header", h.key},
			{"actual", fmtEmpty(actual)},
			{"expected", fmtEmpty(value)},
		}, msgAndArgs...)
	}

	return true
}

func (h Header) Contains(value string, msgAndArgs ...any) bool {
	h.resp.t.Helper()

	if !strings.Contains(h.resp.httpResp.Header.Get(h.key), value) {
		return fail(h.resp.t, "header does not contain expected value", []kv{
			{"header", h.key},
			{"actual", fmtEmpty(h.resp.httpResp.Header.Get(h.key))},
			{"expected", value},
		}, msgAndArgs...)
	}

	return true
}

func (h Header) NotContains(value string, msgAndArgs ...any) bool {
	h.resp.t.Helper()

	if strings.Contains(h.resp.httpResp.Header.Get(h.key), value) {
		return fail(h.resp.t, "header contains value but should not", []kv{
			{"header", h.key},
			{"value", value},
		}, msgAndArgs...)
	}

	return true
}

func (h Header) Values() []string {
	return h.resp.httpResp.Header[textproto.CanonicalMIMEHeaderKey(h.key)]
}

// type Headers struct {
// 	page *Page
// }
// func (hs *Headers) Has(keys ...string) bool {
// 	panic("not implemented")
// }
// func (hs *Headers) HasAll(keys ...string) bool {
// 	panic("not implemented")
// }
// func (hs *Headers) Count() int {
// 	panic("not implemented")
// }
// func (hs *Headers) Values(key string) []string {
// 	panic("not implemented")
// }

func (r response) String() string {
	return r.httpResp.String()
}

// Download represents an HTTP response from a file download request.
// It provides assertions for status codes, headers, and access to the raw body bytes.
type Download struct {
	response
	body []byte
}

// NewDownload creates a Download from an HTTP response.
func NewDownload(t *testing.T, client *req.Client, resp *req.Response, err error) Download {
	t.Helper()

	assert.NotEmpty(t, client, "client is nil")
	assert.NotEmpty(t, resp, "response is nil")
	assert.NoError(t, err)

	body, err := resp.ToBytes()
	assert.NoError(t, err)

	return Download{
		response: response{
			t:        t,
			client:   client,
			httpResp: resp,
		},
		body: body,
	}
}

// Bytes returns the raw response body.
func (d Download) Bytes() []byte {
	return d.body
}

// Into writes the response body into the given writer.
func (d Download) Into(w io.Writer) {
	d.t.Helper()

	_, err := w.Write(d.body)
	assert.NoError(d.t, err)
}

func fmtEmpty(s string) string {
	if s == "" {
		return "<empty>"
	}

	return s
}

// kv is a label/value pair rendered by fail for structured assertion output.
type kv struct {
	label string
	value string
}

// fail reports an assertion failure as a structured, multi-line message:
//
//	selection does not equal expected
//	    actual:    "Overview"
//	    expected:  "Data Overview"
//
// headline describes the mismatch; pairs carry the values (actual/expected)
// and context (selector/path/element/...). msgAndArgs is forwarded to testify.
func fail(t *testing.T, headline string, pairs []kv, msgAndArgs ...any) bool {
	t.Helper()

	var b strings.Builder
	b.WriteString(headline + "\n")

	for _, p := range pairs {
		b.WriteString(fmt.Sprintf("    %-12s %s\n", p.label+":", p.value))
	}

	return assert.Fail(t, b.String(), msgAndArgs...)
}

// truncate caps s to max bytes, appending "..." when truncated. Keeps failure
// output readable when dumping large response bodies or HTML snippets.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}

	return s[:max] + "..."
}

// formatList renders a string slice as ["a", "b"], or [] when empty. Used for
// candidate lists in not-found failures so the developer sees the alternatives
// present without opening the page in a browser.
func formatList(items []string) string {
	if len(items) == 0 {
		return "[]"
	}

	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = fmt.Sprintf("%q", s)
	}

	return "[" + strings.Join(quoted, ", ") + "]"
}

// cookieNames collects the Name of every cookie, for "found:" candidate output.
func cookieNames(cookies []*http.Cookie) []string {
	out := make([]string, 0, len(cookies))

	for i := range cookies {
		out = append(out, cookies[i].Name)
	}

	return out
}

// headerKeys collects response header keys, for "found:" candidate output.
func headerKeys(h http.Header) []string {
	out := make([]string, 0, len(h))

	for k := range h {
		out = append(out, k)
	}

	return out
}

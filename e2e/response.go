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

	return assert.Fail(r.t, "response does not contain: "+text, msgAndArgs...)
}

func (r response) NotContains(text string, msgAndArgs ...any) bool {
	r.t.Helper()

	if strings.Contains(r.httpResp.String(), text) {
		return assert.Fail(r.t, "response contains: "+text+", should not", msgAndArgs...)
	}

	return true
}

func (r response) StatusCode(code int, msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.StatusCode != code {
		return assert.Fail(r.t,
			fmt.Sprintf("response has status code: %d, should be: %d", r.httpResp.StatusCode, code),
			msgAndArgs...)
	}

	return true
}

func (r response) IsOK(msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.StatusCode != http.StatusOK {
		return assert.Fail(r.t,
			fmt.Sprintf("response has status code: %d, should be: 200 (OK)", r.httpResp.StatusCode),
			msgAndArgs...)
	}

	return true
}

func (r response) IsCreated(msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.StatusCode != http.StatusCreated {
		return assert.Fail(r.t,
			fmt.Sprintf("response has status code: %d, should be: 201 (Created)", r.httpResp.StatusCode),
			msgAndArgs...)
	}

	return true
}

func (r response) IsNotFound(msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.StatusCode != http.StatusNotFound {
		return assert.Fail(r.t,
			fmt.Sprintf("response has status code: %d, should be: 404 (Not Found)", r.httpResp.StatusCode),
			msgAndArgs...)
	}

	return true
}

func (r response) IsUnauthorized(msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.StatusCode != http.StatusUnauthorized {
		return assert.Fail(r.t,
			fmt.Sprintf("response has status code: %d, should be: 401 (Unauthorized)", r.httpResp.StatusCode),
			msgAndArgs...)
	}

	return true
}

func (r response) IsForbidden(msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.StatusCode != http.StatusForbidden {
		return assert.Fail(r.t,
			fmt.Sprintf("response has status code: %d, should be: 403 (Forbidden)", r.httpResp.StatusCode),
			msgAndArgs...)
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

	return assert.Fail(r.t, "response does not have cookie: "+name, msgAndArgs...)
}

func (r response) HasCookies(names []string, msgAndArgs ...any) bool {
	r.t.Helper()

	cookies, err := r.client.GetCookies(r.httpResp.Request.URL.String())
	assert.NoError(r.t, err, "failed to get cookies from client jar")

	if len(cookies) == 0 && len(names) > 0 {
		return assert.Fail(r.t, "response does not have any cookies", msgAndArgs...)
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
		return assert.Fail(r.t, fmt.Sprintf("response does not have cookies: %v", notFound), msgAndArgs...)
	}

	return true
}

func (r response) HasCookieTotal(total int, msgAndArgs ...any) bool {
	r.t.Helper()

	cookies, err := r.client.GetCookies(r.httpResp.Request.URL.String())
	assert.NoError(r.t, err, "failed to get cookies from client jar")

	if len(cookies) != total {
		return assert.Fail(r.t,
			fmt.Sprintf("response has %d cookies, should have %d", len(cookies), total),
			msgAndArgs...)
	}

	return true
}

func (r response) IsPath(path string, msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.Response.Request.URL.Path != path {
		return assert.Fail(r.t,
			fmt.Sprintf("path is: %s, should be: %s", fmtEmpty(r.httpResp.Response.Request.URL.Path), path),
			msgAndArgs...)
	}

	return true
}

func (r response) URL() string {
	return r.httpResp.Response.Request.URL.String()
}

func (r response) ContentType(ctype string, msgAndArgs ...any) bool {
	r.t.Helper()

	if r.httpResp.Header.Get("Content-Type") != ctype {
		return assert.Fail(r.t,
			fmt.Sprintf("response has content type: %s, should be: %s", r.httpResp.Header.Get("Content-Type"), ctype),
			msgAndArgs...)
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
		return assert.Fail(h.resp.t, "header does not exist: "+h.key, msgAndArgs...)
	}

	return true
}

func (h Header) NotEmpty(msgAndArgs ...any) bool {
	h.resp.t.Helper()

	if h.resp.httpResp.Header.Get(h.key) == "" {
		return assert.Fail(h.resp.t, fmt.Sprintf("header %s is empty, should not be", h.key), msgAndArgs...)
	}

	return true
}

func (h Header) Is(value string, msgAndArgs ...any) bool {
	h.resp.t.Helper()

	key := textproto.CanonicalMIMEHeaderKey(h.key)
	_, exists := h.resp.httpResp.Header[key]
	actual := h.resp.httpResp.Header.Get(h.key)

	if actual != value || !exists {
		return assert.Fail(h.resp.t,
			fmt.Sprintf("header is: %s, should be: %s", fmtEmpty(actual), fmtEmpty(value)),
			msgAndArgs...)
	}

	return true
}

func (h Header) Contains(value string, msgAndArgs ...any) bool {
	h.resp.t.Helper()

	if !strings.Contains(h.resp.httpResp.Header.Get(h.key), value) {
		return assert.Fail(h.resp.t, "header does not contain: "+value+", should not", msgAndArgs...)
	}

	return true
}

func (h Header) NotContains(value string, msgAndArgs ...any) bool {
	h.resp.t.Helper()

	if strings.Contains(h.resp.httpResp.Header.Get(h.key), value) {
		return assert.Fail(h.resp.t, "header does contain: "+value+", should not", msgAndArgs...)
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

//go:build e2e

package e2e_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/e2e"
)

func TestPage_StatusCode(t *testing.T) {
	t.Parallel()

	t.Run("status equal", func(t *testing.T) {
		t.Parallel()

		svr := server()
		defer svr.Close()

		page := e2e.Test(new(testing.T)).Goto(svr.URL)
		pass := page.StatusCode(http.StatusOK)
		assert.True(t, pass)
	})

	t.Run("status not equal", func(t *testing.T) {
		t.Parallel()

		svr := server()
		defer svr.Close()

		page := e2e.Test(new(testing.T)).Goto(svr.URL)
		pass := page.StatusCode(http.StatusInternalServerError)
		assert.False(t, pass)
	})
}

func TestPage_IsOK(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status int
		pass   bool
	}{
		"status ok":     {http.StatusOK, true},
		"status not ok": {http.StatusInternalServerError, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withStatus(tt.status))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.IsOK()
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestPage_IsCreated(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status int
		pass   bool
	}{
		"status created":     {http.StatusCreated, true},
		"status not created": {http.StatusOK, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withStatus(tt.status))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.IsCreated()
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestPage_IsNotFound(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status int
		pass   bool
	}{
		"status is not found": {http.StatusNotFound, true},
		"status found":        {http.StatusOK, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withStatus(tt.status))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.IsNotFound()
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestPage_IsUnauthorized(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status int
		pass   bool
	}{
		"status unauthorized": {http.StatusUnauthorized, true},
		"status authorized":   {http.StatusOK, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withStatus(tt.status))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.IsUnauthorized()
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestPage_IsForbidden(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status int
		pass   bool
	}{
		"status forbidden": {http.StatusForbidden, true},
		"status allowed":   {http.StatusOK, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withStatus(tt.status))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.IsForbidden()
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestPage_IsRedirected(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status   int
		location string
		pass     bool
	}{
		"status ok":                             {http.StatusOK, "", false},
		"server error":                          {http.StatusInternalServerError, "", false},
		"server error - with location":          {http.StatusInternalServerError, "/", false},
		"moved permanently":                     {http.StatusMovedPermanently, "/", true},
		"moved permanently - missing location":  {http.StatusMovedPermanently, "", false},
		"found":                                 {http.StatusFound, "/", true},
		"found - missing location":              {http.StatusFound, "", false},
		"see other":                             {http.StatusSeeOther, "/", true},
		"see other - missing location":          {http.StatusSeeOther, "", false},
		"temporary redirect":                    {http.StatusTemporaryRedirect, "/", true},
		"temporary redirect - missing location": {http.StatusTemporaryRedirect, "", false},
		"permanent redirect":                    {http.StatusPermanentRedirect, "/", true},
		"permanent redirect - missing location": {http.StatusPermanentRedirect, "", false},
		"not modified":                          {http.StatusNotModified, "", false},
		"not modified - with location":          {http.StatusNotModified, "/", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withStatus(tt.status), withLocation(tt.location))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.IsRedirected()
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestPage_RedirectsTo(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status   int
		location string
		pass     bool
	}{
		"moved permanently":                     {http.StatusMovedPermanently, "/", true},
		"moved permanently - missing location":  {http.StatusMovedPermanently, "", false},
		"found":                                 {http.StatusFound, "/", true},
		"found - missing location":              {http.StatusFound, "", false},
		"see other":                             {http.StatusSeeOther, "/", true},
		"see other - missing location":          {http.StatusSeeOther, "", false},
		"temporary redirect":                    {http.StatusTemporaryRedirect, "/", true},
		"temporary redirect - missing location": {http.StatusTemporaryRedirect, "", false},
		"permanent redirect":                    {http.StatusPermanentRedirect, "/", true},
		"permanent redirect - missing location": {http.StatusPermanentRedirect, "", false},
		"not modified":                          {http.StatusNotModified, "", false},
		"not modified - with location":          {http.StatusNotModified, "/", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withStatus(tt.status), withLocation(tt.location))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.RedirectsTo(tt.location)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestPage_ContentType(t *testing.T) {
	t.Parallel()

	t.Run("content type is text/html", func(t *testing.T) {
		t.Parallel()

		svr := server()
		defer svr.Close()

		page := e2e.Test(new(testing.T)).Goto(svr.URL)
		pass := page.ContentType("text/html")
		assert.True(t, pass)
	})

	t.Run("content type is application/json", func(t *testing.T) {
		t.Parallel()

		svr := server()
		defer svr.Close()

		page := e2e.Test(new(testing.T)).Goto(svr.URL)
		pass := page.ContentType("application/json")
		assert.False(t, pass)
	})
}

func TestPage_IsPath(t *testing.T) {
	t.Parallel()

	t.Run("direct call", func(t *testing.T) {
		t.Parallel()

		svr := server()
		defer svr.Close()

		page := e2e.Test(new(testing.T)).Goto(svr.URL)
		pass := page.IsPath("")
		assert.True(t, pass)

		page = e2e.Test(new(testing.T)).Goto(svr.URL + "/")
		pass = page.IsPath("/")
		assert.True(t, pass)
		pass = page.IsPath("/hello")
		assert.False(t, pass)
	})

	t.Run("redirect", func(t *testing.T) {
		t.Parallel()

		svr := server(withStatus(http.StatusSeeOther), withLocation("/hello"))
		defer svr.Close()

		page := e2e.Test(new(testing.T)).Goto(svr.URL)
		pass := page.IsPath("/hello", "should redirect to URL with path")
		assert.True(t, pass)
	})
}

func TestPage_HasQueryParam(t *testing.T) {
	t.Parallel()

	t.Run("direct call", func(t *testing.T) {
		t.Parallel()

		svr := server()
		defer svr.Close()

		page := e2e.Test(new(testing.T)).Goto(svr.URL + "/?hello=world")
		pass := page.HasQueryParam("hello", "world")
		assert.True(t, pass)
		pass = page.HasQueryParam("hello", "max")
		assert.False(t, pass)
	})

	t.Run("redirect", func(t *testing.T) {
		t.Parallel()

		svr := server(withStatus(http.StatusSeeOther), withLocation("/?hello=world"))
		defer svr.Close()

		page := e2e.Test(new(testing.T)).Goto(svr.URL + "/")
		pass := page.HasQueryParam("hello", "world", "should redirect to URL with query param")
		assert.True(t, pass)
	})
}

func TestScripts_Total(t *testing.T) {
	t.Parallel()

	t.Run("no scripts", func(t *testing.T) {
		t.Parallel()

		svr := server()
		defer svr.Close()

		page := e2e.Test(new(testing.T)).Goto(svr.URL)
		pass := page.Scripts().Total(0)
		assert.True(t, pass)
	})

	t.Run("head scripts", func(t *testing.T) {
		t.Parallel()

		svr := server(withHTML(`<html><head><script src="hello.js"></script></head></html>`))
		defer svr.Close()

		page := e2e.Test(new(testing.T)).Goto(svr.URL)
		pass := page.Scripts().Total(1)
		assert.True(t, pass)
	})

	t.Run("body scripts", func(t *testing.T) {
		t.Parallel()

		svr := server(withHTML(`<html><body><script src="hello.js"></script></body></html>`))
		defer svr.Close()

		page := e2e.Test(new(testing.T)).Goto(svr.URL)
		pass := page.Scripts().Total(1)
		assert.True(t, pass)
	})
}

func TestScripts_HasOnlyLocal(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		html string
		pass bool
	}{
		"no script":                   {`<html><head></head></html>`, true},
		"empty script":                {`<html><head><script src=""></script></head></html>`, true},
		"remote script":               {`<html><head><script src="https://example.com/hello.js"></script></head></html>`, false},
		"local relative":              {`<html><head><script src="hello.js"></script></head></html>`, true},
		"local relative with path":    {`<html><head><script src="../hello.js"></script></head></html>`, true},
		"local absolute":              {`<html><head><script src="/hello.js"></script></head></html>`, true},
		"local query params":          {`<html><head><script src="/hello.js?v=2"></script></head></html>`, true},
		"protocol relative":           {`<html><head><script src="//cdn.example.com/hello.js"></script></head></html>`, false},
		"different port on same host": {`<html><head><script src="http://127.0.0.1:9999/hello.js"></script></head></html>`, false},
		"mixed local and remote":      {`<html><head><script src="/local.js"></script><script src="https://example.com/remote.js"></script></head></html>`, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withHTML(tt.html))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.Scripts().HasOnlyLocal()
			assert.Equal(t, tt.pass, pass)
		})
	}

	t.Run("local script - same host", func(t *testing.T) {
		t.Parallel()

		// empty server to get access to local random port
		svr := httptest.NewServer(nil)
		defer svr.Close()

		// update server handler with HTML containing full script URL
		svr.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, fmt.Sprintf(`<html><head><script src="%s"></script></head></html>`, svr.URL+"/hello.js"))
		})

		page := e2e.Test(new(testing.T)).Goto(svr.URL)
		pass := page.Scripts().HasOnlyLocal()
		assert.True(t, pass, "should pass as host is same as server")
	})

	t.Run("local script - same host different protocol", func(t *testing.T) {
		t.Parallel()

		// empty server to get access to local random port
		svr := httptest.NewServer(nil)
		defer svr.Close()

		parsedURL, err := url.Parse(svr.URL)
		assert.NoError(t, err)

		parsedURL.Scheme = "https"

		// update server handler with HTML containing full script URL
		svr.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, fmt.Sprintf(`<html><head><script src="%s"></script></head></html>`, parsedURL.String()+"/hello.js"))
		})

		page := e2e.Test(new(testing.T)).Goto(svr.URL)
		pass := page.Scripts().HasOnlyLocal()
		assert.True(t, pass, "should pass as host is same as server")
	})
}

func TestScripts_HasLibrary(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		html    string
		library string
		pass    bool
	}{
		"no script":                   {`<html><head></head></html>`, "jquery", false},
		"library not present":         {`<html><head><script src="https://unpkg.com/htmx.org@2.0.8"></script></head></html>`, "jquery", false},
		"library present":             {`<html><head><script src="https://unpkg.com/htmx.org@2.0.8"></script></head></html>`, "htmx.org", true},
		"library present ignore case": {`<html><head><script src="https://unpkg.com/htmx.org@2.0.8"></script></head></html>`, "HTMX.org", true},
		"partial match":               {`<html><head><script src="https://unpkg.com/htmx.org@2.0.8"></script></head></html>`, "htmx", false},
		"exact version":               {`<html><head><script src="https://unpkg.com/htmx.org@2.0.8"></script></head></html>`, "htmx.org@2.0.8", true},
		"version mismatch":            {`<html><head><script src="https://unpkg.com/htmx.org@1.9.10"></script></head></html>`, "htmx.org@2.0.8", false},
		"local library":               {`<html><head><script src="htmx.org@2.0.8"></script></head></html>`, "htmx.org@2.0.8", true},
		"tailwind css cdn":            {`<script src="https://cdn.tailwindcss.com"></script>`, "cdn.tailwindcss.com", true},
		"same lib twice":              {`<script src="https://unpkg.com/htmx.org@2.0.8"></script><script src="https://unpkg.com/htmx.org@2.0.8"></script>`, "htmx.org", true},
		"mix local and cdn":           {`<script src="/local.js"></script><script src="https://unpkg.com/htmx.org@1.9.10"></script>`, "htmx.org", true},
		"local script - app":          {`<script src="/app.js"></script>`, "app", true},
		"local script - main":         {`<script src="/assets/main.js"></script>`, "main", true},
		"minified":                    {`<script src="/assets/main.min.js"></script>`, "main", true},
		"cdn":                         {`<script src="https://cdn.jsdelivr.net/npm/jquery@3.6.4/dist/jquery.min.js"></script>`, "jquery", true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withHTML(tt.html))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.Scripts().HasLibrary(tt.library)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

// TODO method to test the absence of a cookie, like: // assert.NotContains(t, resp.Header.Get("Set-Cookie"), compleroAuthCookieName, "should not set cookie")

func TestPage_HasCookie(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cookies    []*http.Cookie
		cookieName string
		pass       bool
	}{
		"no cookies":     {nil, "my-cookie", false},
		"with cookies":   {[]*http.Cookie{{Name: "my-cookie"}}, "my-cookie", true},
		"other cookies":  {[]*http.Cookie{{Name: "my-cookie"}}, "other-cookie", false},
		"case sensitive": {[]*http.Cookie{{Name: "My-Cookie"}}, "my-cookie", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withCookies(tt.cookies))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.HasCookie(tt.cookieName)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestPage_HasCookies(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cookies     []*http.Cookie
		cookieNames []string
		pass        bool
	}{
		"no cookies":               {nil, []string{}, true},
		"no cookies - empty list":  {[]*http.Cookie{{Name: "my-cookie"}}, []string{}, true},
		"no cookies - search list": {[]*http.Cookie{{}}, []string{"my-cookie"}, false},
		"partial match":            {[]*http.Cookie{{Name: "my-cookie"}}, []string{"my-cookie", "my-cookie-2"}, false},
		"with cookies":             {[]*http.Cookie{{Name: "my-cookie"}, {Name: "my-cookie-2"}, {Name: "my-cookie-3"}}, []string{"my-cookie", "my-cookie-2"}, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withCookies(tt.cookies))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.HasCookies(tt.cookieNames)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestPage_HasCookieTotal(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cookies []*http.Cookie
		total   int
		pass    bool
	}{
		"no cookies":   {nil, 0, true},
		"with cookies": {[]*http.Cookie{{Name: "my-cookie"}, {Name: "my-cookie-2"}}, 2, true},
		"wrong count":  {[]*http.Cookie{{Name: "my-cookie"}, {Name: "my-cookie-2"}}, 3, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withCookies(tt.cookies))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.HasCookieTotal(tt.total)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestPage_TestID(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		id     string
		exists bool
		html   string
	}{
		"not exists":           {"my-id", false, `<div></div>`},
		"exists playwright":    {"my-id", true, `<div data-testid="my-id"></div>`},
		"exists cypress":       {"my-id", true, `<div data-cy="my-id"></div>`},
		"exists multiple time": {"my-id", true, `<div data-testid="my-id"></div><div data-testid="my-id"></div>`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			testID := page.TestID(tt.id)

			assert.Equal(t, tt.exists, testID.Exists())
		})
	}
}

func TestPage_Form(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		selector string
		html     string
		exists   bool
	}{
		"no form":                     {"", `<p></p>`, false},
		"empty selector":              {"", `<form name="my-form" method="post" action="/action"></form>`, false},
		"by name":                     {"my-form", `<form name="my-form" id="my-form-id" method="post" action="/action"></form>`, true},
		"by wrong name":               {"my-form-2", `<form name="my-form" id="my-form" method="post" action="/action"></form>`, false},
		"by id":                       {"my-form", `<form id="my-form" method="post" action="/action"></form>`, true},
		"by action":                   {"/action", `<form method="post" action="/action"></form>`, true},
		"by test id":                  {"my-form", `<form data-testid="my-form" action="/submit"></form>`, true},
		"by cypress id":               {"my-form", `<form data-cy="my-form" action="/submit"></form>`, true},
		"multiple forms":              {"my-form", `<form name="my-form" method="post" action="/action"></form><form name="my-form-2" method="post" action="/action"></form>`, true},
		"multiple forms same name":    {"my-form", `<form name="my-form" method="post" action="/action"></form><form name="my-form" method="post" action="/action"></form>`, false},
		"htmx post":                   {"/action", `<form hx-post="/action"></form>`, true},
		"htmx get":                    {"/action", `<form hx-get="/action"></form>`, true},
		"htmx put":                    {"/action", `<form hx-put="/action"></form>`, true},
		"htmx patch":                  {"/action", `<form hx-patch="/action"></form>`, true},
		"htmx delete":                 {"/action", `<form hx-delete="/action"></form>`, true},
		"button hx-post":              {"/action", `<button hx-post="/action">Go</button>`, true},
		"button hx-get":               {"/action", `<button hx-get="/action">Go</button>`, true},
		"form with spaces in hx-post": {"editCompanyForm", `<form hx-post=" /requests/admin/companies/new " hx-encoding="multipart/form-data" id="editCompanyForm"></form>`, true},
		"id not found":                {"editCompanyForm2", `<form hx-post=" /requests/admin/companies/new " hx-encoding="multipart/form-data" id="editCompanyForm"></form>`, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			form := page.Form(tt.selector)

			assert.Equal(t, tt.exists, form.Exists())
		})
	}
}

func TestForm_Method(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		method string
		html   string
		pass   bool
	}{
		"empty method":     {"", `<form name="my-form" method="post" action="/hello"></form>`, false},
		"wrong method":     {"get", `<form name="my-form" method="post" action="/hello"></form>`, false},
		"correct method":   {"post", `<form name="my-form" method="post" action="/hello"></form>`, true},
		"case insensitive": {http.MethodPost, `<form name="my-form" method="post" action="/hello"></form>`, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			form := e2e.Test(new(testing.T)).Goto(svr.URL).Form("my-form")
			pass := form.Method(tt.method)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestForm_Action(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		action string
		html   string
		pass   bool
	}{
		"empty action":   {"", `<form name="my-form" method="post" action="/hello"></form>`, false},
		"wrong action":   {"/other", `<form name="my-form" method="post" action="/hello"></form>`, false},
		"correct action": {"/hello", `<form name="my-form" method="post" action="/hello"></form>`, true},
		"absolute url":   {"https://example.com/hello", `<form name="my-form" action="https://example.com/hello">`, true},
		"query params":   {"/hello?foo=bar", `<form name="my-form" action="/hello?foo=bar">`, true},
		"url fragment":   {"/hello#section", `<form name="my-form" action="/hello#section">`, true},
		"action missing": {"", `<form name="my-form">`, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			// form := e2e.Test(new(testing.T)).Goto(svr.URL).Form("my-form")
			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			form := page.Form("my-form")
			pass := form.Action(tt.action)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestForm_Total(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		html  string
		total int
		pass  bool
	}{
		"empty form":            {`<form name="my-form"></form>`, 0, true},
		"wrong empty form":      {`<form name="my-form"></form>`, 1, false},
		"multiple inputs":       {`<form name="my-form"><input name="email"><input name="other"></form><form name="my-form-2"><input name="email"></form>`, 2, true},
		"different input types": {`<form name="my-form"><input name="email"><textarea name="other"></textarea></form>`, 2, true},
		"wrong multiple inputs": {`<form name="my-form"><input name="email"><input name="other"></form><form name="my-form-2"><input name="email"></form>`, 1, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			form := e2e.Test(new(testing.T)).Goto(svr.URL).Form("my-form")
			pass := form.Total(tt.total)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestForm_HasField(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name string
		html string
		pass bool
	}{
		"field exists":  {"email", `<form name="my-form"><input name="email"></form>`, true},
		"field missing": {"missing", `<form name="my-form"><input name="email"></form>`, false},
		"empty name":    {"", `<form name="my-form"><input name="email"></form>`, false},
		"same name":     {"", `<form name="my-form"><input name="email"><input name="email"></form>`, false},

		"text input":     {"name", `<form name="my-form"><input type="text" name="name"></form>`, true},
		"email input":    {"email", `<form name="my-form"><input type="email" name="email"></form>`, true},
		"password input": {"password", `<form name="my-form"><input type="password" name="password"></form>`, true},
		"hidden input":   {"csrf", `<form name="my-form"><input type="hidden" name="csrf"></form>`, true},
		"checkbox":       {"remember", `<form name="my-form"><input type="checkbox" name="remember"></form>`, true},
		"radio":          {"gender", `<form name="my-form"><input type="radio" name="gender"></form>`, true},
		"file input":     {"avatar", `<form name="my-form"><input type="file" name="avatar"></form>`, true},
		"textarea":       {"bio", `<form name="my-form"><textarea name="bio"></textarea></form>`, true},
		"select":         {"country", `<form name="my-form"><select name="country"></select></form>`, true},
		"button":         {"submit", `<form name="my-form"><button name="submit"></button></form>`, true},

		"field in wrong form":            {"other-field", `<form name="my-form"><input name="my-field"></form><form name="other-form"><input name="other-field"></form>`, false},
		"field in correct form":          {"my-field", `<form name="my-form"><input name="my-field"></form><form name="other-form"><input name="other-field"></form>`, true},
		"multiple field in correct form": {"my-field", `<form name="my-form"><input name="my-field"><input name="other-field"></form><form name="other-form"><input name="other-field"></form>`, true},

		"special chars in name": {"user[email]", `<form name="my-form"><input name="user[email]"></form>`, true},
		"name with dash":        {"user-email", `<form name="my-form"><input name="user-email"></form>`, true},
		"name with underscore":  {"user_email", `<form name="my-form"><input name="user_email"></form>`, true},
		"case sensitive":        {"Email", `<form name="my-form"><input name="email"></form>`, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			form := e2e.Test(new(testing.T)).Goto(svr.URL).Form("my-form")
			pass := form.HasField(tt.name)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestForm_HasSubmit(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		html string
		pass bool
	}{
		"empty form":             {`<form name="my-form"></form>`, false},
		"submit without name":    {`<form name="my-form"><input type="submit"></form>`, true},
		"submit with name":       {`<form name="my-form"><input type="submit" name="submit"></form>`, true},
		"submit missing":         {`<form name="my-form"><input type="button" name="submit"></form>`, false},
		"same name":              {`<form name="my-form"><input type="submit" name="submit" value="1"><input type="submit" name="submit" value="2"></form>`, true},
		"multiple submits":       {`<form name="my-form"><input type="submit" name="submit"><input type="submit" name="submit-2"></form>`, true},
		"button element":         {`<form name="my-form"><button type="submit">Submit</button></form>`, true},
		"button with name":       {`<form name="my-form"><button type="submit" name="action">Save</button></form>`, true},
		"mixed types":            {`<form name="my-form"><input type="submit" name="save"><button type="submit" name="publish">Publish</button></form>`, true},
		"multiple without names": {`<form name="my-form"><input type="submit"><input type="submit"></form>`, false},
		"uppercase type":         {`<form name="my-form"><input type="SUBMIT" name="submit"></form>`, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			form := e2e.Test(new(testing.T)).Goto(svr.URL).Form("my-form")
			pass := form.HasSubmit()
			assert.Equal(t, tt.pass, pass)
		})
	}
}

//nolint:thelper // assert() field should be easy to read
func TestForm_Submit(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		html       string
		submitData map[string]any
		assert     func(t *testing.T, got *capturedRequest)
	}{
		"post method": {
			`<form name="my-form" method="post" action="/submit"><input name="email"></form>`,
			map[string]any{"email": "test@example.com"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, http.MethodPost, got.method)
				assert.Equal(t, "/submit", got.path)
				assert.Equal(t, "test@example.com", got.body.Get("email"))
			},
		},
		"post multiple values": {
			`<form name="my-form" method="post" action="/submit"><input name="email"></form>`,
			map[string]any{"emails": []string{"0@example.com", "1@example.com"}},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, http.MethodPost, got.method)
				assert.Equal(t, "/submit", got.path)
				assert.Equal(t, []string{"0@example.com", "1@example.com"}, got.body["emails"])
			},
		},
		"get method": {
			`<form name="my-form" method="get" action="/submit"><input name="search"></form>`,
			map[string]any{"search": "test"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, http.MethodGet, got.method)
				assert.Equal(t, "/submit", got.path)
			},
		},
		"htmx post": {
			`<form name="my-form" hx-post="/submit"><input name="email"></form>`,
			map[string]any{"email": "test@example.com"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, http.MethodPost, got.method)
				assert.Equal(t, "/submit", got.path)
			},
		},
		"htmx get": {
			`<form name="my-form" hx-get="/submit"><input name="q"></form>`,
			map[string]any{"q": "test"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, http.MethodGet, got.method)
				assert.Equal(t, "/submit", got.path)
			},
		},
		"htmx put": {
			`<form name="my-form" hx-put="/submit"><input name="data"></form>`,
			map[string]any{"data": "value"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, http.MethodPut, got.method)
				assert.Equal(t, "/submit", got.path)
			},
		},
		"htmx delete": {
			`<form name="my-form" hx-delete="/submit"><input name="id"></form>`,
			map[string]any{"id": "123"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, http.MethodDelete, got.method)
				assert.Equal(t, "/submit", got.path)
			},
		},
		"htmx patch": {
			`<form name="my-form" hx-patch="/submit"><input name="email"></form>`,
			map[string]any{"email": "test@example.com"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, http.MethodPatch, got.method)
				assert.Equal(t, "/submit", got.path)
			},
		},
		"htmx post with spaces in action": {
			`<form name="my-form" hx-post=" /submit "><input name="email"></form>`,
			map[string]any{"email": "test@example.com"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, "/submit", got.path)
			},
		},
		"htmx overrides standard action": {
			`<form name="my-form" method="get" action="/standard" hx-post="/submit"><input name="data"></form>`,
			map[string]any{"data": "value"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, http.MethodPost, got.method)
				assert.Equal(t, "/submit", got.path)
			},
		},
		"empty action resolves to current page": {
			`<form name="my-form" method="post"><input name="email"></form>`,
			map[string]any{"email": "test@example.com"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, "/", got.path)
				assert.Equal(t, "test@example.com", got.body.Get("email"))
			},
		},
		"relative path": {
			`<form name="my-form" method="post" action="submit"><input name="email"></form>`,
			map[string]any{"email": "test@example.com"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, "/submit", got.path)
			},
		},
		"special characters in data": {
			`<form name="my-form" method="post" action="/submit"><input name="email"></form>`,
			map[string]any{"email": "test+tag@example.com"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, "test+tag@example.com", got.body.Get("email"))
			},
		},
		"hx-encoding sets multipart": {
			`<form name="my-form" hx-post="/submit" hx-encoding="multipart/form-data"><input name="file"></form>`,
			map[string]any{"file": "data"},
			func(t *testing.T, got *capturedRequest) {
				assert.Contains(t, got.headers.Get("Content-Type"), "multipart/form-data")
			},
		},
		"hx-post sends HX-Request header": {
			`<form name="my-form" hx-post="/submit"><input name="email"></form>`,
			map[string]any{"email": "test@example.com"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, "true", got.headers.Get("Hx-Request"))
			},
		},
		"hx-headers sends custom headers": {
			`<form name="my-form" hx-post="/submit" hx-headers='{"X-CSRF-Token":"abc123"}'><input name="email"></form>`,
			map[string]any{"email": "test@example.com"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, "abc123", got.headers.Get("X-Csrf-Token"))
			},
		},
		"hx-method overrides method attribute": {
			`<form name="my-form" method="post" hx-method="put" action="/submit"><input name="email"></form>`,
			map[string]any{"email": "test@example.com"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, http.MethodPut, got.method)
			},
		},
		"hx-vals merged into submit data": {
			`<form name="my-form" hx-post="/submit" hx-vals='{"source":"newsletter"}'><input name="email"></form>`,
			map[string]any{"email": "test@example.com"},
			func(t *testing.T, got *capturedRequest) {
				assert.Equal(t, "test@example.com", got.body.Get("email"))
				assert.Equal(t, "newsletter", got.body.Get("source"))
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got capturedRequest

			svr := server(withBody(tt.html), withCapture(&got))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			page = page.Form("my-form").Submit(tt.submitData)

			page.IsOK()
			tt.assert(t, &got)
		})
	}
}

func TestForm_SubmitWithFile(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		file e2e.SubmitOption
	}{
		"file reader": {e2e.WithFileReader("myFile", "test.txt", bytes.NewReader([]byte("test data")))},
		"file bytes":  {e2e.WithFileBytes("myFile", "test.txt", []byte("test data"))},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := capturedRequest{fileNameToCapture: "myFile"}

			svr := server(
				withHTML(`<form name="my-form" method="post" enctype="multipart/form-data" action="/submit"><input type="file" name="myFile"/></form>`),
				withCapture(&got),
			)
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			page = page.Form("my-form").Submit(map[string]any{}, tt.file)

			page.IsOK()
			assert.Equal(t, "test data", string(got.fileBody))
		})
	}
}

func TestForm_ErrorDetection(t *testing.T) {
	t.Parallel()
	/*
	   Missing for SSR:
	   1. Tailwind form validation: ring-red-500, focus:ring-red-500, border-red-500
	   2. ARIA error refs (accessibility): aria-describedby="email-error" + element with id="email-error"
	   3. HTMX patterns: hx-validation-class, data-hx-validation
	   4. Fieldset errors: <fieldset class="errors"> patterns
	   5. Hidden accessibility errors: sr-only or visually-hidden with role="alert"
	*/
	tests := map[string]struct {
		html      string
		hasErrors bool
	}{
		"clean form":   {`<form name="my-form"><input name="email"></form>`, false},
		"valid fields": {`<form name="my-form"><input name="email" class="form-control"></form>`, false},
		// ARIA patterns for accessibility
		"aria-invalid true":     {`<form name="my-form"><input name="email" aria-invalid="true"></form>`, true},
		"aria-invalid multiple": {`<form name="my-form"><input aria-invalid="true" name="email"><input aria-invalid="true" name="password"></form>`, true},
		// common CSS classes
		"is-invalid class":       {`<form name="my-form"><input name="email" class="is-invalid"></form>`, true},
		"invalid-feedback text":  {`<form name="my-form"><input name="email"><div class="invalid-feedback">Error</div></form>`, true},
		"text-danger class":      {`<form name="my-form"><input name="email"><span class="text-danger">Error</span></form>`, true},
		"errorlist class":        {`<form name="my-form"><ul class="errorlist"><li>Error</li></ul></form>`, true},
		"errors class":           {`<form name="my-form"><div class="errors">Error</div></form>`, true},
		"form-group with errors": {`<form name="my-form"><div class="form-group errors"><input name="email"></div></form>`, true},
		"has-error class":        {`<form name="my-form"><input name="email" class="has-error"></form>`, true},
		"field-error class":      {`<form name="my-form"><div class="field-error">Error</div></form>`, true},
		"error class":            {`<form name="my-form"><div class="error">Error</div></form>`, true},
		"text-red class":         {`<form name="my-form"><span class="text-red">Error</span></form>`, true},
		"multiple error types":   {`<form name="my-form"><input class="is-invalid" name="email"><div class="error">Password required</div></form>`, true},
		// form level
		"alert":           {`<form name="my-form"><div class="alert">Passwords don't match</div></form>`, true},
		"form.error":      {`<form name="my-form" class="error"><input name="email"></form>`, true},
		"form.has-error":  {`<form name="my-form" class="has-error"><input name="email"></form>`, true},
		"form.has-errors": {`<form name="my-form" class="has-errors"><input name="email"></form>`, true},
		"multiple errors": {`<form name="my-form"><input class="is-invalid" name="email"><div class="error">Required</div></form>`, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			form := e2e.Test(new(testing.T)).Goto(svr.URL).Form("my-form")

			assert.Equal(t, tt.hasErrors, form.HasErrors())
			assert.Equal(t, !tt.hasErrors, form.HasNoErrors())
		})
	}
}

func TestForm_HasFieldError(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		html string
		pass bool
	}{
		"clean field":               {`<form name="my-form"><input name="my-input"></form>`, false},
		"valid field":               {`<form name="my-form"><input name="my-input" class="form-control"></form>`, false},
		"aria-invalid on field":     {`<form name="my-form"><input name="my-input" aria-invalid="true"></form>`, true},
		"aria-valid on field":       {`<form name="my-form"><input name="my-input" aria-invalid="false"></form>`, false},
		"is-invalid class on field": {`<form name="my-form"><input name="my-input" class="is-invalid"></form>`, true},
		"has-error class on field":  {`<form name="my-form"><input name="my-input" class="has-error"></form>`, true},
		"invalid-feedback div":      {`<form name="my-form"><input name="my-input"><div class="invalid-feedback">Error</div></form>`, true},
		"text-danger span":          {`<form name="my-form"><input name="my-input"><span class="text-danger">Error</span></form>`, true},
		"field not found":           {`<form name="my-form"><input name="other-input"></form>`, false},
		"other field has error":     {`<form name="my-form"><input name="my-input"><input name="other-input" class="is-invalid"></form>`, false},
		"field ok form has errors":  {`<form name="my-form"><input name="my-input"><div class="errors">Other error</div></form>`, true},
		// wrapper patterns
		"field in wrapper with error":       {`<form name="my-form"><div class="form-group"><input name="my-input" class="is-invalid"><div class="invalid-feedback">Error</div></div></form>`, true},
		"other field in wrapper with error": {`<form name="my-form"><div class="form-group"><input name="my-input"></div><div class="form-group"><input name="other-input" class="is-invalid"><div class="invalid-feedback">Error</div></div></form>`, false},
		"error separated by help text":      {`<form name="my-form"><div class="form-group"><input name="my-input"><small>Help text</small><div class="invalid-feedback">Error</div></div></form>`, true},
		"other field error separated":       {`<form name="my-form"><div class="form-group"><input name="my-input"></div><div class="form-group"><input name="other-input"><small>Help</small><div class="invalid-feedback">Error</div></div></form>`, false},
		"wrapper has error class":           {`<form name="my-form"><div class="form-group has-error"><input name="my-input"></div></form>`, true},
		"other wrapper has error class":     {`<form name="my-form"><div class="form-group"><input name="my-input"></div><div class="form-group has-error"><input name="other-input"></div></form>`, false},
		// Other form field types after target field
		"next textarea has error": {`<form name="my-form"><input name="my-input"><textarea name="other" class="is-invalid"></textarea></form>`, false},
		"next select has error":   {`<form name="my-form"><input name="my-input"><select name="other" class="is-invalid"></select></form>`, false},
		"next button has error":   {`<form name="my-form"><input name="my-input"><button name="other" class="is-invalid"></button></form>`, false},
		// Tailwind/common pattern: error as sibling of wrapper div
		"tailwind wrapper pattern": {`<form name="my-form"><div><div class="relative"><label for="login"><input name="my-input"></label></div><span class="text-red-500">Error message</span></div></form>`, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			form := e2e.Test(new(testing.T)).Goto(svr.URL).Form("my-form")
			pass := form.HasFieldError("my-input")
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestForm_HasFieldValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		html  string
		name  string
		value string
		pass  bool
	}{
		"exact value match":     {`<form name="my-form"><input name="email" value="test@example.com"></form>`, "email", "test@example.com", true},
		"different value":       {`<form name="my-form"><input name="email" value="test@example.com"></form>`, "email", "other@example.com", false},
		"no value attribute":    {`<form name="my-form"><input name="email"></form>`, "email", "", true},
		"empty value attribute": {`<form name="my-form"><input name="email" value=""></form>`, "email", "", true},
		"field not found":       {`<form name="my-form"><input name="other"></form>`, "email", "", false},

		"textarea with value": {`<form name="my-form"><textarea name="bio">Developer</textarea></form>`, "bio", "Developer", true},
		"textarea multiline": {`<form name="my-form"><textarea name="address">Line 1
Line 2</textarea></form>`, "address", "Line 1\nLine 2", true},

		"select with selected option": {`<form name="my-form"><select name="country"><option value="us">USA</option><option value="de" selected>Germany</option></select></form>`, "country", "de", true},
		"select no value attr":        {`<form name="my-form"><select name="country"><option selected>USA</option><option>Germany</option></select></form>`, "country", "USA", true},

		"checkbox checked":   {`<form name="my-form"><input type="checkbox" name="remember" value="true" checked></form>`, "remember", "true", true},
		"checkbox unchecked": {`<form name="my-form"><input type="checkbox" name="remember" value="true"></form>`, "remember", "", true},

		"radio selected":      {`<form name="my-form"><input type="radio" name="gender" value="male" checked><input type="radio" name="gender" value="female"></form>`, "gender", "male", true},
		"radio not selected":  {`<form name="my-form"><input type="radio" name="gender" value="male"><input type="radio" name="gender" value="female" checked></form>`, "gender", "male", false},
		"radio none selected": {`<form name="my-form"><input type="radio" name="gender" value="male"><input type="radio" name="gender" value="female"></form>`, "gender", "", true},

		"html entities decoding": {`<form name="my-form"><input name="text" value="a&lt;b"></form>`, "text", "a<b", true},
		"hidden input":           {`<form name="my-form"><input type="hidden" name="csrf" value="token123"></form>`, "csrf", "token123", true},
		"multiple fields":        {`<form name="my-form"><input name="tag" value="go"><input name="tag" value="java"></form>`, "tag", "go", true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			form := e2e.Test(new(testing.T)).Goto(svr.URL).Form("my-form")
			pass := form.HasFieldValue(tt.name, tt.value)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestElement_Total(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		selector string
		html     string
		expected int
		pass     bool
	}{
		"single match":   {"div", `<div></div>`, 1, true},
		"multiple match": {"li", `<ul><li>a</li><li>b</li><li>c</li></ul>`, 3, true},
		"no match":       {"span", `<div></div>`, 0, true},
		"wrong count":    {"div", `<div></div>`, 2, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			elem := e2e.Test(new(testing.T)).Goto(svr.URL).Find(tt.selector)
			pass := elem.Total(tt.expected)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestElement_Find(t *testing.T) {
	t.Parallel()

	html := `<ul><li>alpha</li><li>beta</li><li>gamma</li></ul>`

	t.Run("first, last, nth", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			fn       func(e2e.Element) e2e.Element
			expected string
		}{
			"first":       {func(el e2e.Element) e2e.Element { return el.First() }, "alpha"},
			"last":        {func(el e2e.Element) e2e.Element { return el.Last() }, "gamma"},
			"nth middle":  {func(el e2e.Element) e2e.Element { return el.Nth(1) }, "beta"},
			"nth first":   {func(el e2e.Element) e2e.Element { return el.Nth(0) }, "alpha"},
			"nth last":    {func(el e2e.Element) e2e.Element { return el.Nth(2) }, "gamma"},
			"nth inverse": {func(el e2e.Element) e2e.Element { return el.Nth(-1) }, "gamma"},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				svr := server(withBody(html))
				defer svr.Close()

				elem := e2e.Test(new(testing.T)).Goto(svr.URL).Find("li")
				assert.Equal(t, tt.expected, tt.fn(elem).Text())
			})
		}
	})

	t.Run("nth out of range", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			index int
			pass  bool
		}{
			"in range":     {0, true},
			"out of range": {1337, false},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				svr := server(withBody(html))
				defer svr.Close()

				elem := e2e.Test(new(testing.T)).Goto(svr.URL).
					Find("li").
					Nth(tt.index)

				if tt.pass {
					assert.NotEmpty(t, elem)
				} else {
					assert.Empty(t, elem)
				}
			})
		}
	})

	t.Run("ambiguous selector triggers assertion", func(t *testing.T) {
		t.Parallel()

		svr := server(withBody(`<ul><li>a</li><li>b</li></ul>`))
		defer svr.Close()

		page := e2e.Test(new(testing.T)).Goto(svr.URL)

		assert.Empty(t, page.Find("li").Text(), "Find() should fail the test and Text() never return any value")
		// First resolves ambiguity -> Text() returns real value
		assert.Equal(t, "a", page.Find("li").First().Text())
	})
}

func TestElement_HasAttr(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		html string
		attr string
		pass bool
	}{
		"attr with value":       {`<div id="my-id" data-attr="value"></div>`, "data-attr", true},
		"attr with empty value": {`<div id="my-id" data-attr=""></div>`, "data-attr", true},
		"attr without value":    {`<div id="my-id" data-attr></div>`, "data-attr", true},
		"no attribute":          {`<div id="my-id"></div>`, "data-attr", false},
		"different attr":        {`<div id="my-id" data-attr="value"></div>`, "other-attr", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			elem := e2e.Test(new(testing.T)).Goto(svr.URL).Find("#my-id")
			pass := elem.HasAttr(tt.attr)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestElement_Is(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tag  string
		pass bool
		html string
	}{
		"matching tag":            {"div", true, `<div id="my-id"></div>`},
		"wrong tag":               {"span", false, `<div id="my-id"></div>`},
		"case insensitive":        {"DIV", true, `<div id="my-id"></div>`},
		"element with attributes": {"input", true, `<input id="my-id" type="text" class="form-control">`},
		"self-closing element":    {"img", true, `<img id="my-id" src="test.jpg">`},
		"nested element":          {"span", true, `<div><p><span id="my-id"></span></p></div>`},
		"empty tag parameter":     {"", false, `<div id="my-id"></div>`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			elem := e2e.Test(new(testing.T)).Goto(svr.URL).Find("#my-id")
			pass := elem.Is(tt.tag)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestHeader_Exists(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		header    string
		value     string
		expHeader string
		pass      bool
	}{
		"exists":     {"Content-Type", "text/html", "Content-Type", true},
		"empty":      {"Content-Type", "", "Content-Type", true},
		"not exists": {"Content-Type", "text/html", "Location", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withHeader(tt.header, tt.value))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.Header(tt.expHeader).Exists()
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestHeader_NotEmpty(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		header    string
		value     string
		expHeader string
		pass      bool
	}{
		"not empty":  {"Content-Type", "text/html", "Content-Type", true},
		"empty":      {"Content-Type", "", "Content-Type", false},
		"not exists": {"Content-Type", "text/html", "Location", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withHeader(tt.header, tt.value))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.Header(tt.expHeader).NotEmpty()
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestHeader_Is(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		header    string
		value     string
		expHeader string
		expValue  string
		pass      bool
	}{
		"exists":                    {"Content-Type", "text/html", "Content-Type", "text/html", true},
		"empty":                     {"Content-Type", "", "Content-Type", "", true},
		"not exists":                {"Content-Type", "text/html", "Location", "", false},
		"wrong value":               {"Content-Type", "text/html", "Content-Type", "application/json", false},
		"case insensitive":          {"Content-Type", "text/html", "content-type", "text/html", true},
		"content-type with charset": {"Content-Type", "text/html; charset=utf-8", "Content-Type", "text/html; charset=utf-8", true},
		"charset case mismatch":     {"Content-Type", "text/html; charset=utf-8", "Content-Type", "text/html; charset=UTF-8", false},
		"partial match fail":        {"HX-Trigger", "event0,event1", "HX-Trigger", "event0", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withHeader(tt.header, tt.value))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.Header(tt.expHeader).Is(tt.expValue)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestHeader_Contains(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		header    string
		value     string
		expHeader string
		expValue  string
		pass      bool
	}{
		"empty":                  {"HX-Trigger", "event0,event1", "HX-Trigger", "", true},
		"contains":               {"HX-Trigger", "event0,event1", "HX-Trigger", "event0", true},
		"case insensitive":       {"HX-Trigger", "event0,event1", "hx-trigger", "event0", true},
		"case insensitive value": {"HX-Trigger", "event0,event1", "hx-trigger", "Event0", false},
		"contains not":           {"HX-Trigger", "event0,event1", "HX-Trigger", "otherEvent", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withHeader(tt.header, tt.value))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.Header(tt.expHeader).Contains(tt.expValue)
			assert.Equal(t, tt.pass, pass)

			pass = page.Header(tt.expHeader).NotContains(tt.expValue)
			assert.Equal(t, !tt.pass, pass)
		})
	}
}

func TestPage_Title(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		html  string
		title string
		pass  bool
	}{
		"matching title": {`<html><head><title>Dashboard</title></head><body></body></html>`, "Dashboard", true},
		"wrong title":    {`<html><head><title>Dashboard</title></head><body></body></html>`, "Login", false},
		"no title tag":   {`<html><head></head><body></body></html>`, "Anything", false},
		"empty title":    {`<html><head><title></title></head><body></body></html>`, "", true},
		"fragment":       {`<div>content</div>`, "", true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withHTML(tt.html))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.Title(tt.title)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestPage_HasLink(t *testing.T) { //nolint:dupl
	t.Parallel()

	tests := map[string]struct {
		html string
		text string
		href string
		pass bool
	}{
		"exact match":    {`<a href="/login">Login</a>`, "Login", "/login", true},
		"wrong text":     {`<a href="/login">Login</a>`, "Logout", "/login", false},
		"wrong href":     {`<a href="/login">Login</a>`, "Login", "/logout", false},
		"no links":       {`<div>no links</div>`, "Login", "/login", false},
		"multiple links": {`<a href="/a">A</a><a href="/b">B</a>`, "B", "/b", true},
		"whitespace":     {`<a href="/login"> Login </a>`, "Login", "/login", true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			pass := page.HasLink(tt.text, tt.href)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestPage_DumpHTML(t *testing.T) {
	t.Parallel()

	svr := server(withBody(`<p>hello</p>`))
	defer svr.Close()

	page := e2e.Test(new(testing.T)).Goto(svr.URL)

	dump := page.DumpHTML()
	assert.Contains(t, dump, "hello")
}

func TestElement_HasText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		html string
		text string
		pass bool
	}{
		"equal text":     {`<div id="el">hello</div>`, "hello", true},
		"different text": {`<div id="el">hello</div>`, "missing", false},
		"wrong selector": {`<div id="other">hello</div>`, "hello", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			elem := e2e.Test(new(testing.T)).Goto(svr.URL).Find("#el")
			pass := elem.HasText(tt.text)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestElement_Equals(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		html  string
		value string
		pass  bool
	}{
		"equal":           {`<div id="el">hello</div>`, "hello", true},
		"different value": {`<div id="el">hello</div>`, "other", false},
		"different elem":  {`<div id="other">hello</div>`, "hello", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			elem := e2e.Test(new(testing.T)).Goto(svr.URL).Find("#el")
			pass := elem.Equals(tt.value)
			assert.Equal(t, tt.pass, pass)

			pass = elem.NotEquals(tt.value)
			assert.Equal(t, !tt.pass, pass)
		})
	}
}

func TestPage_Table(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		selector string
		html     string
		exists   bool
	}{
		"no table":                  {"", `<p></p>`, false},
		"empty selector":            {"", `<table name="my-table"></table>`, false},
		"by id":                     {"my-table", `<table id="my-table"></table>`, true},
		"by wrong id":               {"my-table-2", `<table id="my-table"></table>`, false},
		"by test id":                {"my-table", `<table data-testid="my-table"></table>`, true},
		"by cypress id":             {"my-table", `<table data-cy="my-table"></table>`, true},
		"by caption":                {"My Table", `<table id="my-table"><caption>My Table</caption></table>`, true},
		"by css selector":           {"div table", `<div><table id="my-table"><caption>My Table</caption></table></div>`, true},
		"by css selector missing":   {"div table", `<table id="my-table"><caption>My Table</caption></table>`, false},
		"multiple tables":           {"my-table", `<table id="my-table"></table><table id="my-table-2"></table>`, true},
		"multiple tables same name": {"my-table", `<table id="my-table"></table><table id="my-table"></table>`, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			page := e2e.Test(new(testing.T)).Goto(svr.URL)
			table := page.Table(tt.selector)

			assert.Equal(t, tt.exists, table.Exists())
		})
	}
}

func TestTable_Empty(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		html string
		pass bool
	}{
		"empty table":                 {`<table></table>`, true},
		"empty table content":         {`<table><tr><td></td><td></td></tr></table>`, true},
		"empty table with whitespace": {`<table><tr><td> </td></tr></table>`, true},
		"empty table with message":    {`<table><tr><td colspan="3">No data</td></tr></table>`, true},
		"not empty":                   {`<table><tr><td>hello</td></tr></table>`, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			table := e2e.Test(new(testing.T)).
				Goto(svr.URL).
				Table("table")

			assert.Equal(t, tt.pass, table.Empty())
			assert.Equal(t, !tt.pass, table.NotEmpty())
		})
	}
}

func TestTable_Headers(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		headers []string
		html    string
		pass    bool
	}{
		"single header":                      {[]string{"Name"}, `<table><thead><tr><th>Name</th></tr></thead></table>`, true},
		"multiple headers":                   {[]string{"Name", "Email", "Role"}, `<table><thead><tr><th>Name</th><th>Email</th><th>Role</th></tr></thead></table>`, true},
		"wrong header text":                  {[]string{"Name", "Phone"}, `<table><thead><tr><th>Name</th><th>Email</th></tr></thead></table>`, false},
		"wrong header count too many":        {[]string{"Name", "Email", "Role"}, `<table><thead><tr><th>Name</th><th>Email</th></tr></thead></table>`, false},
		"wrong header count too few":         {[]string{"Name"}, `<table><thead><tr><th>Name</th><th>Email</th></tr></thead></table>`, false},
		"td in thead still header":           {[]string{"Name"}, `<table><thead><tr><td>Name</td></tr></thead></table>`, true},
		"header from first row no thead":     {[]string{"Name", "Email"}, `<table><tr><th>Name</th><th>Email</th></tr><tr><td>Alice</td><td>a@b.com</td></tr></table>`, true},
		"no table content":                   {[]string{"Name"}, `<table></table>`, false},
		"empty headers empty table":          {nil, `<table></table>`, true},
		"empty headers expected empty table": {nil, `<table><thead><tr><th>Name</th></tr></thead></table>`, false},
		"headers with whitespace":            {[]string{"Name", "Email"}, `<table><thead><tr><th> Name </th><th> Email </th></tr></thead></table>`, true},
		"header case mismatch":               {[]string{"name", "email"}, `<table><thead><tr><th>Name</th><th>Email</th></tr></thead></table>`, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			table := e2e.Test(new(testing.T)).
				Goto(svr.URL).
				Table("table")

			pass := table.Headers(tt.headers...)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestTable_Rows(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		n    int
		html string
		pass bool
	}{
		"empty table":                {0, `<table></table>`, true},
		"only thead":                 {0, `<table><thead><tr><th>Name</th></tr></thead></table>`, true},
		"single row":                 {1, `<table><tbody><tr><td>Alice</td></tr></tbody></table>`, true},
		"multiple rows":              {2, `<table><tbody><tr><td>Alice</td></tr><tr><td>Bob</td></tr></tbody></table>`, true},
		"wrong count":                {2, `<table><tbody><tr><td>Alice</td></tr></tbody></table>`, false},
		"excludes tfoot":             {1, `<table><tbody><tr><td>Alice</td></tr></tbody><tfoot><tr><td>Total</td></tr></tfoot></table>`, true},
		"excludes thead":             {1, `<table><thead><tr><th>Name</th></tr></thead><tbody><tr><td>Alice</td></tr></tbody></table>`, true},
		"no tbody counts tr":         {1, `<table><tr><td>Alice</td></tr></table>`, true},
		"th scope row is row":        {2, `<table><thead><tr><th>Items</th><th>Cost</th></tr></thead><tbody><tr><th scope="row">Donuts</th><td>3,000</td></tr><tr><th scope="row">Stationery</th><td>18,000</td></tr></tbody></table>`, true},
		"colspan row counts":         {1, `<table><tbody><tr><td colspan="3">No data</td></tr></tbody></table>`, true},
		"row fewer cols than header": {1, `<table><thead><tr><th>A</th><th>B</th><th>C</th></tr></thead><tbody><tr><td>a</td><td>b</td></tr></tbody></table>`, true},
		"row more cols than header":  {1, `<table><thead><tr><th>A</th></tr></thead><tbody><tr><td>a</td><td>b</td><td>c</td></tr></tbody></table>`, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			table := e2e.Test(new(testing.T)).
				Goto(svr.URL).
				Table("table")

			pass := table.Rows(tt.n)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestTable_Cols(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		n    int
		html string
		pass bool
	}{
		"empty table":                           {0, `<table></table>`, true},
		"from thead":                            {3, `<table><thead><tr><th>Name</th><th>Email</th><th>Role</th></tr></thead></table>`, true},
		"wrong count":                           {2, `<table><thead><tr><th>Name</th><th>Email</th><th>Role</th></tr></thead></table>`, false},
		"from first tr":                         {2, `<table><tr><th>Name</th><th>Email</th></tr><tr><td>Alice</td><td>a@b.com</td></tr></table>`, true},
		"from data row":                         {3, `<table><tbody><tr><td>a</td><td>b</td><td>c</td></tr></tbody></table>`, true},
		"colspan ignored":                       {1, `<table><tbody><tr><td colspan="3">No data</td></tr></tbody></table>`, true},
		"mixed th td thead":                     {2, `<table><thead><tr><th>Name</th><td>Extra</td></tr></thead></table>`, true},
		"cols from header ignores row mismatch": {3, `<table><thead><tr><th>A</th><th>B</th><th>C</th></tr></thead><tbody><tr><td>a</td></tr></tbody></table>`, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			table := e2e.Test(new(testing.T)).
				Goto(svr.URL).
				Table("table")

			pass := table.Cols(tt.n)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestTable_RowCount(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		expected int
		html     string
	}{
		"empty table":    {0, `<table></table>`},
		"only thead":     {0, `<table><thead><tr><th>Name</th></tr></thead></table>`},
		"single row":     {1, `<table><tbody><tr><td>Alice</td></tr></tbody></table>`},
		"multiple rows":  {3, `<table><tbody><tr><td>a</td></tr><tr><td>b</td></tr><tr><td>c</td></tr></tbody></table>`},
		"excludes tfoot": {1, `<table><tbody><tr><td>Alice</td></tr></tbody><tfoot><tr><td>Total</td></tr></tfoot></table>`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			table := e2e.Test(new(testing.T)).
				Goto(svr.URL).
				Table("table")

			assert.Equal(t, tt.expected, table.RowCount())
		})
	}
}

func TestTable_ColCount(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		expected int
		html     string
	}{
		"empty table":   {0, `<table></table>`},
		"from thead":    {3, `<table><thead><tr><th>A</th><th>B</th><th>C</th></tr></thead></table>`},
		"from first tr": {2, `<table><tr><th>A</th><th>B</th></tr><tr><td>a</td><td>b</td></tr></table>`},
		"from data row": {3, `<table><tbody><tr><td>a</td><td>b</td><td>c</td></tr></tbody></table>`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withBody(tt.html))
			defer svr.Close()

			table := e2e.Test(new(testing.T)).
				Goto(svr.URL).
				Table("table")

			assert.Equal(t, tt.expected, table.ColCount())
		})
	}
}

func TestTable_Row(t *testing.T) {
	t.Parallel()

	html := `<table>
		<thead><tr><th>Name</th><th>Email</th></tr></thead>
		<tbody>
			<tr><td>Alice</td><td>alice@example.com</td></tr>
			<tr><td>Bob</td><td>bob@example.com</td></tr>
		</tbody>
		<tfoot><tr><td>Total</td><td>2</td></tr></tfoot>
	</table>`

	t.Run("returns row element", func(t *testing.T) {
		t.Parallel()

		svr := server(withBody(html))
		defer svr.Close()

		table := e2e.Test(new(testing.T)).
			Goto(svr.URL).
			Table("table")

		row := table.Row(0)
		assert.True(t, row.Exists())
		assert.Contains(t, row.Text(), "Alice")
		assert.Contains(t, row.Text(), "alice@example.com")
	})

	t.Run("second row", func(t *testing.T) {
		t.Parallel()

		svr := server(withBody(html))
		defer svr.Close()

		table := e2e.Test(new(testing.T)).
			Goto(svr.URL).
			Table("table")

		row := table.Row(1)
		assert.Contains(t, row.Text(), "Bob")
		assert.Contains(t, row.Text(), "bob@example.com")
	})

	t.Run("out of range returns empty element", func(t *testing.T) {
		t.Parallel()

		svr := server(withBody(html))
		defer svr.Close()

		table := e2e.Test(new(testing.T)).
			Goto(svr.URL).
			Table("table")

		row := table.Row(99)
		assert.False(t, row.Exists())
	})

	t.Run("th scope row accessible", func(t *testing.T) {
		t.Parallel()

		scopeHTML := `<table>
			<thead><tr><th>Items</th><th>Cost</th></tr></thead>
			<tbody><tr><th scope="row">Donuts</th><td>3,000</td></tr></tbody>
		</table>`

		svr := server(withBody(scopeHTML))
		defer svr.Close()

		table := e2e.Test(new(testing.T)).
			Goto(svr.URL).
			Table("table")

		row := table.Row(0)
		assert.Contains(t, row.Text(), "Donuts")
	})
}

func TestTable_Col(t *testing.T) {
	t.Parallel()

	html := `<table>
		<thead><tr><th>Name</th><th>Email</th><th>Role</th></tr></thead>
		<tbody>
			<tr><td>Alice</td><td>alice@example.com</td><td>Admin</td></tr>
			<tr><th scope="row">Bob</th><td>bob@example.com</td><td>User</td></tr>
		</tbody>
	</table>`

	t.Run("access col by index on row", func(t *testing.T) {
		t.Parallel()

		svr := server(withBody(html))
		defer svr.Close()

		table := e2e.Test(new(testing.T)).
			Goto(svr.URL).
			Table("table")

		assert.Equal(t, "Alice", table.Row(0).Col(0).Text())
		assert.Equal(t, "alice@example.com", table.Row(0).Col(1).Text())
		assert.Equal(t, "Admin", table.Row(0).Col(2).Text())
	})

	t.Run("th scope row is col 0", func(t *testing.T) {
		t.Parallel()

		svr := server(withBody(html))
		defer svr.Close()

		table := e2e.Test(new(testing.T)).
			Goto(svr.URL).
			Table("table")

		assert.Equal(t, "Bob", table.Row(1).Col(0).Text())
		assert.Equal(t, "bob@example.com", table.Row(1).Col(1).Text())
	})

	t.Run("out of range col returns empty element", func(t *testing.T) {
		t.Parallel()

		svr := server(withBody(html))
		defer svr.Close()

		table := e2e.Test(new(testing.T)).
			Goto(svr.URL).
			Table("table")

		assert.False(t, table.Row(0).Col(99).Exists())
	})

	t.Run("out of range row returns empty element", func(t *testing.T) {
		t.Parallel()

		svr := server(withBody(html))
		defer svr.Close()

		table := e2e.Test(new(testing.T)).
			Goto(svr.URL).
			Table("table")

		assert.False(t, table.Row(99).Col(0).Exists())
	})
}

type capturedRequest struct {
	method            string
	path              string
	headers           http.Header
	body              url.Values // populated only for application/x-www-form-urlencoded
	rawBody           string     // always populated for non-GET requests
	fileNameToCapture string     // set to capture file values
	fileName          string
	fileSize          int64
	fileHeader        textproto.MIMEHeader
	fileBody          []byte
}

type serverOption func(*serverConfig)

type serverConfig struct {
	status   int
	cookies  []*http.Cookie
	headers  map[string]string
	html     string
	captured *capturedRequest
}

func withStatus(status int) serverOption {
	return func(cfg *serverConfig) {
		cfg.status = status
	}
}

func withLocation(location string) serverOption {
	return func(cfg *serverConfig) {
		cfg.headers["Location"] = location
	}
}

func withCookies(cookies []*http.Cookie) serverOption {
	return func(cfg *serverConfig) {
		cfg.cookies = cookies
	}
}

func withHeader(header string, value string) serverOption {
	return func(cfg *serverConfig) {
		cfg.headers[header] = value
	}
}

func withHTML(html string) serverOption {
	return func(cfg *serverConfig) {
		cfg.html = html
	}
}

func withBody(body string) serverOption {
	return func(cfg *serverConfig) {
		cfg.html = `<html><body>` + body + `</body></html>`
	}
}

func withCapture(capture *capturedRequest) serverOption {
	return func(cfg *serverConfig) {
		cfg.captured = capture
	}
}

func server(opts ...serverOption) *httptest.Server { //nolint:gocognit,gocyclo,cyclop
	cfg := &serverConfig{
		status: http.StatusOK,
		html:   `<html><body><h1>Hello World</h1></body></html>`,
		headers: map[string]string{
			"Content-Type": "text/html",
		},
		captured: &capturedRequest{},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		for i := range cfg.cookies {
			http.SetCookie(w, cfg.cookies[i])
		}

		for header, value := range cfg.headers {
			w.Header().Set(header, value)
		}

		w.WriteHeader(cfg.status)

		cfg.captured.method = req.Method
		cfg.captured.path = req.URL.Path
		cfg.captured.headers = req.Header.Clone()

		if req.URL.Path == "/submit" || req.Method != http.MethodGet { //nolint:nestif
			if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
				file, handler, err := req.FormFile(cfg.captured.fileNameToCapture)
				if err == nil {
					defer file.Close()

					cfg.captured.fileName = handler.Filename
					cfg.captured.fileSize = handler.Size
					cfg.captured.fileHeader = handler.Header

					cfg.captured.fileBody, err = io.ReadAll(file)
					if err != nil {
						panic(err)
					}
				}
			} else {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					panic(err)
				}

				cfg.captured.rawBody = string(body)

				if req.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
					cfg.captured.body, err = url.ParseQuery(cfg.captured.rawBody)
					if err != nil {
						panic(err)
					}
				}
			}
		}

		io.WriteString(w, cfg.html)
	}))
}

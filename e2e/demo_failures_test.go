//go:build e2e_demo

// This file is excluded from normal `go test -tags e2e` runs
// It deliberately triggers every assertion's failure
// path against real HTML/HTTP so the actual formatted error output
// can be inspected from a user's perspective.
//
// Run:
//
//	go test -tags 'e2e e2e_demo' -v -run TestDemo ./e2e/
//
// Every subtest is expected to FAIL — that is the point. Use -v to read the
// rendered messages. Zoom in with, e.g.:
//
//	go test -tags 'e2e e2e_demo' -v -run 'TestDemo/Element_Equals' ./e2e/

package e2e

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imroc/req/v3"
)

// pageFromHTML builds a Page from an HTML string served over a real HTTP
// round-trip, so failures carry the page URL/status context too.
func pageFromHTML(t *testing.T, body string) Page {
	t.Helper()

	return pageFromHandler(t, http.StatusOK, map[string]string{"Content-Type": "text/html"}, "<html><body>"+body+"</body></html>")
}

// pageFromHandler builds a Page through a real HTTP round-trip (for assertions
// that read the response: status, headers, query params, body).
func pageFromHandler(t *testing.T, status int, headers map[string]string, body string) Page {
	t.Helper()

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}

		if status == 0 {
			status = http.StatusOK
		}

		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(svr.Close)

	client := req.C()
	resp, err := client.R().Get(svr.URL)

	return NewPage(t, client, resp, err)
}

func docFromHandler(t *testing.T, body string) Document {
	t.Helper()

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, body)
	}))
	t.Cleanup(svr.Close)

	client := req.C()
	resp, err := client.R().Get(svr.URL)

	return NewJSON(t, client, resp, err)
}

func TestDemo(t *testing.T) {
	// === Element: value / structure mismatches ===

	t.Run("Element_Equals", func(t *testing.T) {
		pageFromHTML(t, `<h1 class="title">Overview</h1>`).Find("h1").Equals("Data Overview")
	})

	t.Run("Element_Contains", func(t *testing.T) {
		pageFromHTML(t, `<h1>User Dashboard</h1>`).Find("h1").Contains("Data Overview")
	})

	t.Run("Element_NotContains", func(t *testing.T) {
		pageFromHTML(t, `<h1>Overview</h1>`).Find("h1").NotContains("Overview")
	})

	t.Run("Element_NotEquals", func(t *testing.T) {
		pageFromHTML(t, `<h1>Overview</h1>`).Find("h1").NotEquals("Overview")
	})

	t.Run("Element_HasAttr_missing", func(t *testing.T) {
		pageFromHTML(t, `<input id="email" type="text" name="email">`).Find("#email").HasAttr("placeholder")
	})

	t.Run("Element_AttrIs_mismatch", func(t *testing.T) {
		pageFromHTML(t, `<input id="email" type="text" name="email">`).Find("#email").AttrIs("type", "email")
	})

	t.Run("Element_Is_tagMismatch", func(t *testing.T) {
		pageFromHTML(t, `<h1 id="el">Title</h1>`).Find("#el").Is("span")
	})

	t.Run("Element_Total_countMismatch", func(t *testing.T) {
		pageFromHTML(t, `<ul><li>a</li><li>b</li></ul>`).Find("li").Total(5)
	})

	t.Run("Element_Exists_missing", func(t *testing.T) {
		pageFromHTML(t, `<div>x</div>`).Find(".nope").Exists()
	})

	t.Run("Element_ambiguousSelection", func(t *testing.T) {
		pageFromHTML(t, `<ul><li>a</li><li>b</li></ul>`).Find("li").Text() // assertSingle fires
	})

	t.Run("Element_Nth_outOfRange", func(t *testing.T) {
		pageFromHTML(t, `<ul><li>a</li></ul>`).Find("li").Nth(99)
	})

	// === Page: DOM lookups ===

	t.Run("Page_Title_mismatch", func(t *testing.T) {
		pageFromHTML(t, `<title>Dashboard</title>`).Title("Login")
	})

	t.Run("Page_HasLink_missing", func(t *testing.T) {
		pageFromHTML(t, `<a href="/login">Login</a>`).HasLink("Logout", "/logout")
	})

	t.Run("Page_TestID_notFound", func(t *testing.T) {
		pageFromHTML(t, `<div data-testid="overview"></div><div data-cy="contact-form"></div>`).
			TestID("data-overview")
	})

	// Warnings log (test continues) — visible with -v.

	t.Run("Page_TestID_duplicateWarning", func(t *testing.T) {
		pageFromHTML(t, `<div data-testid="overview"></div><div data-testid="overview"></div>`).
			TestID("overview").Exists()
	})

	// === Form ===

	t.Run("Form_notFound", func(t *testing.T) {
		pageFromHTML(t, `<form name="real-form" action="/save"><input name="email"></form>`).Form("missing-form")
	})

	t.Run("Form_Method_mismatch", func(t *testing.T) {
		pageFromHTML(t, `<form name="f" method="post" action="/a"></form>`).Form("f").Method("get")
	})

	t.Run("Form_Action_mismatch", func(t *testing.T) {
		pageFromHTML(t, `<form name="f" method="post" action="/save"></form>`).Form("f").Action("/other")
	})

	t.Run("Form_Total_mismatch", func(t *testing.T) {
		pageFromHTML(t, `<form name="f"><input name="a"><input name="b"></form>`).Form("f").Total(5)
	})

	t.Run("Form_HasField_missing", func(t *testing.T) {
		pageFromHTML(t, `<form name="f"><input name="email"><input name="password"></form>`).
			Form("f").HasField("username")
	})

	t.Run("Form_HasSubmit_none", func(t *testing.T) {
		pageFromHTML(t, `<form name="f"><input name="email"></form>`).Form("f").HasSubmit()
	})

	t.Run("Form_HasFieldError_fieldNotFound", func(t *testing.T) {
		pageFromHTML(t, `<form name="f"><input name="email"></form>`).Form("f").HasFieldError("username")
	})

	t.Run("Form_HasFieldError_noError", func(t *testing.T) {
		pageFromHTML(t, `<form name="f"><input name="email"></form>`).Form("f").HasFieldError("email")
	})

	t.Run("Form_HasFieldValue_mismatch", func(t *testing.T) {
		pageFromHTML(t, `<form name="f"><input name="email" value="a@b.com"></form>`).
			Form("f").HasFieldValue("email", "other@example.com")
	})

	t.Run("Form_HasFieldValue_fieldNotFound", func(t *testing.T) {
		pageFromHTML(t, `<form name="f"><input name="email"></form>`).Form("f").HasFieldValue("username", "x")
	})

	t.Run("Form_HasErrors_noneButShould", func(t *testing.T) {
		pageFromHTML(t, `<form name="f"><input name="email"></form>`).Form("f").HasErrors()
	})

	t.Run("Form_HasNoErrors_hasButShouldNot", func(t *testing.T) {
		pageFromHTML(t, `<form name="f"><input name="email" class="is-invalid"></form>`).Form("f").HasNoErrors()
	})

	// === Table ===

	t.Run("Table_notFound", func(t *testing.T) {
		pageFromHTML(t, `<table id="real-table"><caption>Real</caption></table>`).Table("missing-table")
	})

	t.Run("Table_Headers_mismatch", func(t *testing.T) {
		pageFromHTML(t, `<table><thead><tr><th>Name</th><th>Email</th></tr></thead></table>`).
			Table("table").Headers("Name", "Phone")
	})

	t.Run("Table_Rows_mismatch", func(t *testing.T) {
		pageFromHTML(t, `<table><tbody><tr><td>a</td></tr></tbody></table>`).Table("table").Rows(3)
	})

	t.Run("Table_Cols_mismatch", func(t *testing.T) {
		pageFromHTML(t, `<table><thead><tr><th>A</th><th>B</th></tr></thead></table>`).Table("table").Cols(3)
	})

	t.Run("Table_NotEmpty_empty", func(t *testing.T) {
		pageFromHTML(t, `<table></table>`).Table("table").NotEmpty()
	})

	// === Response: status / headers / body ===

	t.Run("Response_StatusCode_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusInternalServerError, nil, `<html><body></body></html>`).StatusCode(http.StatusOK)
	})

	t.Run("Response_IsOK_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusNotFound, nil, `<html><body></body></html>`).IsOK()
	})

	t.Run("Response_ContentType_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, map[string]string{"Content-Type": "text/html"}, `<html><body></body></html>`).
			ContentType("application/json")
	})

	t.Run("Response_HasCookie_missing", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, nil, `<html><body></body></html>`).HasCookie("session")
	})

	t.Run("Response_Header_missing", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, map[string]string{"Content-Type": "text/html"}, `<html><body></body></html>`).
			Header("X-Request-Id").Exists()
	})

	t.Run("Response_Header_Is_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK,
			map[string]string{"Content-Type": "text/html; charset=utf-8"}, `<html><body></body></html>`).
			Header("Content-Type").Is("application/json")
	})

	t.Run("Response_Contains_missing", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, nil, `<html><body>hello world</body></html>`).Contains("goodbye")
	})

	// === JSON (Document) ===

	t.Run("JSON_FieldEquals_mismatch", func(t *testing.T) {
		docFromHandler(t, `{"user":{"name":"Alice","email":"a@b.com"}}`).
			FieldEquals("user.name", "Bob")
	})

	t.Run("JSON_HasField_missing", func(t *testing.T) {
		docFromHandler(t, `{"id":1,"name":"Alice"}`).HasField("phone")
	})

	t.Run("JSON_Total_mismatch", func(t *testing.T) {
		docFromHandler(t, `[1,2,3]`).Total(5)
	})

	// === Response: status variants / redirect / cookies / url ===

	t.Run("Response_IsCreated_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, nil, `<html><body></body></html>`).IsCreated()
	})

	t.Run("Response_IsNotFound_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, nil, `<html><body></body></html>`).IsNotFound()
	})

	t.Run("Response_IsUnauthorized_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, nil, `<html><body></body></html>`).IsUnauthorized()
	})

	t.Run("Response_IsForbidden_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, nil, `<html><body></body></html>`).IsForbidden()
	})

	t.Run("Response_IsRedirected_notRedirect", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, nil, `<html><body></body></html>`).IsRedirected()
	})

	t.Run("Response_RedirectsTo_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusFound, map[string]string{"Location": "/here"}, `<html><body></body></html>`).
			RedirectsTo("/there")
	})

	t.Run("Response_HasCookies_missing", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, nil, `<html><body></body></html>`).HasCookies([]string{"session", "csrf"})
	})

	t.Run("Response_HasCookieTotal_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, nil, `<html><body></body></html>`).HasCookieTotal(2)
	})

	t.Run("Response_IsPath_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, nil, `<html><body></body></html>`).IsPath("/hello")
	})

	t.Run("Response_HasQueryParam_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK, nil, `<html><body></body></html>`).HasQueryParam("q", "search")
	})

	// === Header: contains / not contains ===

	t.Run("Response_Header_Contains_mismatch", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK,
			map[string]string{"Content-Type": "text/html; charset=utf-8"}, `<html><body></body></html>`).
			Header("Content-Type").Contains("json")
	})

	t.Run("Response_Header_NotContains_violation", func(t *testing.T) {
		pageFromHandler(t, http.StatusOK,
			map[string]string{"Content-Type": "text/html"}, `<html><body></body></html>`).
			Header("Content-Type").NotContains("html")
	})

	// === Scripts ===

	t.Run("Scripts_Total_mismatch", func(t *testing.T) {
		pageFromHTML(t, `<script src="/app.js"></script><script src="/vendor.js"></script>`).
			Scripts().Total(5)
	})

	t.Run("Scripts_HasLibrary_missing", func(t *testing.T) {
		pageFromHTML(t, `<script src="/app.js"></script>`).Scripts().HasLibrary("jquery")
	})

	t.Run("Scripts_HasOnlyLocal_violation", func(t *testing.T) {
		pageFromHTML(t, `<script src="https://cdn.example.com/lib.js"></script>`).Scripts().HasOnlyLocal()
	})

	// === JSON extras ===

	t.Run("JSON_NotHasField_violation", func(t *testing.T) {
		docFromHandler(t, `{"id":1,"name":"Alice"}`).NotHasField("name")
	})

	t.Run("JSON_FieldContains_mismatch", func(t *testing.T) {
		docFromHandler(t, `{"name":"Alice"}`).FieldContains("name", "Bob")
	})

	t.Run("JSON_FieldLen_mismatch", func(t *testing.T) {
		docFromHandler(t, `{"items":[1,2,3]}`).FieldLen("items", 5)
	})

	t.Run("JSON_IsArray_violation", func(t *testing.T) {
		docFromHandler(t, `{"id":1}`).IsArray()
	})

	t.Run("JSON_IsObject_violation", func(t *testing.T) {
		docFromHandler(t, `[1,2,3]`).IsObject()
	})
}

//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

type Page struct {
	response
	document *goquery.Document
}

// NewPage
// It takes in the error so that callers can save one assertion line in their test case:
//
//	resp, err = client.R().Get("some/page")
//	page := e2e.NewPage(t, client, resp, err)
func NewPage(t *testing.T, client *req.Client, resp *req.Response, err error) Page {
	t.Helper()

	assert.NotEmpty(t, client, "client is nil")
	assert.NotEmpty(t, resp, "response is nil")
	assert.NoError(t, err)

	htmlBody := &bytes.Buffer{}
	doc, err := goquery.NewDocumentFromReader(io.TeeReader(resp.Body, htmlBody))
	assert.NoError(t, err)

	assertHTMLIsValid(t, htmlBody)

	return Page{
		response: response{
			t:        t,
			client:   client,
			httpResp: resp,
		},
		document: doc,
	}
}

// Goto navigates to a URL, preserving the page's client and cookies
// This allows cookie persistence across page navigations (simulating same browser tab)
// Use suite.Goto() for a fresh session (new client, no cookies).
func (p Page) Goto(url string, opts ...GotoOption) Page {
	p.t.Helper()

	for _, opt := range opts {
		p.client = opt(p.client)
	}

	resp, err := p.client.R().Get(url)

	return NewPage(p.t, p.client, resp, err)
}

func (p Page) IsRedirected(msgAndArgs ...any) bool {
	p.t.Helper()

	isRedirectStatus := p.httpResp.StatusCode > http.StatusMultipleChoices && p.httpResp.StatusCode < http.StatusBadRequest
	isNotModified := p.httpResp.StatusCode == http.StatusNotModified // not a "real" redirect
	hasLocation := p.httpResp.Header.Get("Location") != ""

	if isRedirectStatus && !isNotModified && hasLocation {
		return true
	}

	return fail(p.t, "expected redirect status (3xx)", []kv{
		{"actual", fmt.Sprintf("%d (%s)", p.httpResp.StatusCode, http.StatusText(p.httpResp.StatusCode))},
		{"expected", "3xx (Redirect)"},
	}, msgAndArgs...)
}

func (p Page) RedirectsTo(path string, msgAndArgs ...any) bool {
	p.t.Helper()

	if !p.IsRedirected() || p.httpResp.Header.Get("Location") != path {
		return fail(p.t, "redirect location mismatch", []kv{
			{"actual", fmtEmpty(p.httpResp.Header.Get("Location"))},
			{"expected", path},
		}, msgAndArgs...)
	}

	return true
}

func (p Page) Find(selector string) Element {
	sel := p.document.Find(selector)
	return Element{t: p.t, selection: sel, selector: selector, httpResp: p.httpResp}
}

func (p Page) TestID(id string) Element {
	p.t.Helper()

	matches := p.document.Find(`[data-testid="` + id + `"]`)
	if matches.Length() > 0 {
		if matches.Length() > 1 {
			warn(p.t, "duplicate testid, using first match", []kv{
				{"testid", id},
				{"count", strconv.Itoa(matches.Length())},
				{"hint", "testids should be unique"},
			})
		}

		return Element{t: p.t, selection: matches.First(), selector: `[data-testid="` + id + `"]`, httpResp: p.httpResp}
	}

	matches = p.document.Find(`[data-cy="` + id + `"]`)
	if matches.Length() > 0 {
		if matches.Length() > 1 {
			warn(p.t, "duplicate testid, using first match", []kv{
				{"testid", id},
				{"count", strconv.Itoa(matches.Length())},
				{"hint", "testids should be unique"},
			})
		}

		return Element{t: p.t, selection: matches.First(), selector: `[data-cy="` + id + `"]`, httpResp: p.httpResp}
	}

	fail(p.t, "no element found with test id", []kv{
		{"testid", id},
		{"data-testid", formatList(collectAttrValues(p.document, "[data-testid]", "data-testid"))},
		{"data-cy", formatList(collectAttrValues(p.document, "[data-cy]", "data-cy"))},
	})

	return Element{t: p.t, selection: nil, selector: "", httpResp: p.httpResp}
}

func (p Page) HasQueryParam(key, value string, msgAndArgs ...any) bool {
	p.t.Helper()

	if p.httpResp.Response.Request.URL.Query().Get(key) != value {
		return fail(p.t, "query param mismatch", []kv{
			{"param", key},
			{"actual", fmtEmpty(p.httpResp.Response.Request.URL.Query().Get(key))},
			{"expected", value},
		}, msgAndArgs...)
	}

	return true
}

// Title asserts the page's <title> equals the expected value.
func (p Page) Title(title string, msgAndArgs ...any) bool {
	p.t.Helper()
	actual := p.document.Find("title").Text()

	if actual != title {
		return fail(p.t, "page title mismatch", []kv{
			{"actual", fmtEmpty(actual)},
			{"expected", title},
		}, msgAndArgs...)
	}

	return true
}

// HasLink asserts that an anchor with the given text content and href exists.
func (p Page) HasLink(text, href string, msgAndArgs ...any) bool {
	p.t.Helper()

	links := p.document.Find("a")
	for i := range links.Nodes {
		link := links.Eq(i)
		linkText := strings.TrimSpace(link.Text())
		linkHref, _ := link.Attr("href")

		if linkText == text && linkHref == href {
			return true
		}
	}

	return fail(p.t, "no link with text and href", []kv{
		{"text", text},
		{"href", href},
	}, msgAndArgs...)
}

// DumpHTML returns the full HTML of the page for debugging.
func (p Page) DumpHTML() string {
	p.t.Helper()

	html, err := p.document.Html()
	if err != nil {
		return ""
	}

	return html
}

// Download makes a GET request and returns a Download for asserting on binary responses.
// It reuses the page's client (preserves cookies, same session).
func (p Page) Download(url string, opts ...DownloadOption) Download {
	cfg := downloadConfig{method: http.MethodGet, formData: map[string]any{}}
	for _, opt := range opts {
		opt(&cfg)
	}

	var (
		resp *req.Response
		err  error
	)

	req := p.client.R()

	switch cfg.method {
	case http.MethodPost:
		req.SetFormDataAnyType(cfg.formData)
		resp, err = req.Post(url)
	default:
		resp, err = req.Get(url)
	}

	return NewDownload(p.t, p.client, resp, err)
}

type Form struct {
	t         *testing.T
	page      Page
	selection *goquery.Selection
}

// kv builds the kv pairs shared by every Form failure: the form
// identifier, then caller-supplied pairs, then the originating page context.
func (f Form) kv(extra ...kv) []kv {
	pairs := make([]kv, 0, 1+len(extra))
	pairs = append(pairs, kv{"form", formIdentifier(f.selection)})
	pairs = append(pairs, extra...)
	pairs = append(pairs, pageContext(f.page.httpResp)...)

	return pairs
}

// Form finds a unique form element by name, id, action, or htmx action attribute.
// Selector precedence (first unique match wins): name > id > action > hx-* > css selector
// Returns nil if no form or multiple forms match the given selector.
func (p Page) Form(nameOrIDOrAction string) Form {
	for _, selector := range []string{
		"form[name='" + nameOrIDOrAction + "']",
		"form[id='" + nameOrIDOrAction + "']",
		"form[action='" + nameOrIDOrAction + "']",
		"[hx-get='" + nameOrIDOrAction + "']",
		"[hx-post='" + nameOrIDOrAction + "']",
		"[hx-put='" + nameOrIDOrAction + "']",
		"[hx-patch='" + nameOrIDOrAction + "']",
		"[hx-delete='" + nameOrIDOrAction + "']",
		"[data-testid='" + nameOrIDOrAction + "']",
		"[data-cy='" + nameOrIDOrAction + "']",
		nameOrIDOrAction,
	} {
		s := p.document.Find(selector)

		if s.Length() == 1 {
			return Form{
				t:         p.t,
				page:      p,
				selection: s,
			}
		}
	}

	p.t.Helper()
	fail(p.t, "no unique form found", []kv{
		{"selector", nameOrIDOrAction},
		{"forms", formatList(collectFormIdentifiers(p.document))},
	})

	return Form{t: p.t, page: Page{}, selection: nil}
}

func (f Form) String() string {
	f.t.Helper()

	if f.selection == nil {
		return ""
	}

	t, err := f.selection.Html()
	assert.NoError(f.t, err)

	return t
}

func (f Form) Exists(msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil || f.selection.Length() == 0 {
		return fail(f.t, "form does not exist", nil, msgAndArgs...)
	}

	return true
}

func (f Form) Method(method string, msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	if m, ok := f.selection.Attr("method"); !ok || !strings.EqualFold(m, method) {
		return fail(f.t, "form method mismatch", f.kv(
			kv{"actual", fmtEmpty(m)},
			kv{"expected", method},
		), msgAndArgs...)
	}

	return true
}

func (f Form) Action(action string, msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	if a, ok := f.selection.Attr("action"); !ok || a != action {
		return fail(f.t, "form action mismatch", f.kv(
			kv{"actual", fmtEmpty(a)},
			kv{"expected", action},
		), msgAndArgs...)
	}

	return true
}

func (f Form) Total(total int, msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	if s := f.selection.Find("input, textarea, select, button"); s.Length() != total {
		return fail(f.t, "form element count mismatch", f.kv(
			kv{"actual", strconv.Itoa(s.Length())},
			kv{"expected", strconv.Itoa(total)},
		), msgAndArgs...)
	}

	return true
}

func (f Form) HasField(name string, msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	selector := `input[name='` + name + `'], ` +
		`textarea[name='` + name + `'], ` +
		`select[name='` + name + `'], ` +
		`button[name='` + name + `']`
	if s := f.selection.Find(selector); s.Length() > 0 {
		return true
	}

	return fail(f.t, "form has no field", f.kv(
		kv{"field", name},
		kv{"fields", formatList(collectFieldNames(f.selection))},
	), msgAndArgs...)
}

func (f Form) HasSubmit(msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	submits := f.selection.Find("input[type='submit'], input[type='SUBMIT'], button[type='submit'], button[type='SUBMIT']")
	if submits.Length() == 0 {
		return fail(f.t, "form has no submit button", f.kv(
			kv{"controls", formatList(collectButtons(f.selection))},
		), msgAndArgs...)
	}

	// check for duplicate unnamed submits (more than 1 unnamed submit is ambiguous)
	unnamedCount := 0

	submits.Each(func(_ int, sel *goquery.Selection) {
		if _, hasName := sel.Attr("name"); !hasName {
			unnamedCount++
		}
	})

	if unnamedCount > 1 {
		return fail(f.t, "form has ambiguous unnamed submit buttons", f.kv(
			kv{"count", strconv.Itoa(unnamedCount)},
			kv{"hint", "submit buttons must be unique or named"},
		), msgAndArgs...)
	}

	return true
}

//nolint:gochecknoglobals // shared error selector configuration
var errorSelectors = []string{
	// ARIA accessibility standard
	"[aria-invalid='true']",
	"[aria-invalid='TRUE']",

	// PHP/Laravel
	".is-invalid",
	".invalid-feedback",
	".text-danger",

	// Django
	".errorlist",
	".errors",

	// Generic/bootstrap/common
	".has-error",
	".has-errors",
	".field-error",
	".error",
	".text-red",
	".validation-error",
	".text-red-500",
	".alert",
}

func (f Form) HasNoErrors(msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	for _, selector := range errorSelectors {
		if f.selection.Is(selector) || f.selection.Find(selector).Length() > 0 {
			return fail(f.t, "form has validation errors but should not", f.kv(
				kv{"matched", selector},
			), msgAndArgs...)
		}
	}

	return true
}

func (f Form) HasErrors(msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	for _, selector := range errorSelectors {
		if f.selection.Is(selector) || f.selection.Find(selector).Length() > 0 {
			return true
		}
	}

	return fail(f.t, "form has no validation errors but should", f.kv(), msgAndArgs...)
}

func (f Form) HasFieldError(name string, msgAndArgs ...any) bool { //nolint:gocognit,gocyclo,cyclop,funlen // DOM walk
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	field := f.selection.Find(fmt.Sprintf("[name='%s']", name))
	if field.Length() != 1 {
		return fail(f.t, "field not found in form", f.kv(
			kv{"field", name},
			kv{"fields", formatList(collectFieldNames(f.selection))},
		), msgAndArgs...)
	}

	// check if selection matches any error selector
	hasError := func(s *goquery.Selection) bool {
		for _, selector := range errorSelectors {
			if s.Is(selector) {
				return true
			}
		}

		return false
	}

	if hasError(field) {
		return true
	}

	// check next siblings
	for next := field.Next(); next.Length() > 0; next = next.Next() {
		if next.Is("input[name], textarea[name], select[name], button[name]") {
			break
		}

		if hasError(next) {
			return true
		}
	}

	// check ancestors and their siblings (for wrapper patterns)
	for wrapper := field.Parent(); wrapper.Length() > 0 && !wrapper.Is("form"); wrapper = wrapper.Parent() {
		// check if wrapper itself has error class
		if hasError(wrapper) {
			return true
		}

		// check wrapper's next siblings for error classes
		// only if wrapper + siblings don't contain other fields (stay within field group)
		for next := wrapper.Next(); next.Length() > 0; next = next.Next() {
			// stop if we hit another form field (different field group)
			if next.Is("input[name], textarea[name], select[name], button[name]") {
				break
			}

			// stop if sibling contains other form fields (different field group)
			if next.Find("input[name], textarea[name], select[name], button[name]").Length() > 0 {
				break
			}

			if hasError(next) {
				return true
			}
		}

		// stop if multiple fields in this wrapper (reached group boundary)
		if wrapper.Find("input[name], textarea[name], select[name], button[name]").Length() > 1 {
			break
		}
	}

	return fail(f.t, "field has no validation errors but should", f.kv(
		kv{"field", name},
	), msgAndArgs...)
}

func (f Form) HasFieldValue(name string, value string, msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	fields := f.selection.Find(fmt.Sprintf("[name='%s']", name))
	if fields.Length() == 0 {
		return fail(f.t, "field not found in form", f.kv(
			kv{"field", name},
			kv{"fields", formatList(collectFieldNames(f.selection))},
		), msgAndArgs...)
	}

	allValues := make([]string, len(fields.Nodes))

	for i := range fields.Nodes {
		val := fieldValue(fields.Eq(i))
		allValues[i] = val

		if val == value {
			return true
		}
	}

	valuesStr := fmt.Sprintf("%v", allValues)
	if len(allValues) == 1 {
		valuesStr = fmt.Sprintf("'%s'", allValues[0])
	}

	return fail(f.t, "field value mismatch", f.kv(
		kv{"field", name},
		kv{"actual", valuesStr},
		kv{"expected", value},
	), msgAndArgs...)
}

// fieldValue extracts the submitted value of a single form control, by type.
// Checkbox/radio report their value only when checked; select falls back to the
// first option when none is explicitly selected.
func fieldValue(field *goquery.Selection) string {
	if field.Is("input[type='checkbox'], input[type='CHECKBOX']") {
		if _, hasChecked := field.Attr("checked"); hasChecked {
			return field.AttrOr("value", "on")
		}

		return ""
	}

	if field.Is("input[type='radio'], input[type='RADIO']") {
		if _, hasChecked := field.Attr("checked"); hasChecked {
			return field.AttrOr("value", "")
		}

		return ""
	}

	if field.Is("textarea") {
		return field.Text()
	}

	if field.Is("select") {
		selectedOption := field.Find("option[selected]")
		if selectedOption.Length() > 0 {
			return selectedOption.AttrOr("value", selectedOption.Text())
		}

		firstOption := field.Find("option").First()
		if firstOption.Length() > 0 {
			return firstOption.AttrOr("value", firstOption.Text())
		}

		return ""
	}

	return field.AttrOr("value", "")
}

type SubmitOption func(request *req.Request)

func WithFileReader(paramName string, filename string, reader io.Reader) SubmitOption {
	return func(req *req.Request) {
		req.SetFileReader(paramName, filename, reader)
		req.EnableForceMultipart()
	}
}

func WithFileBytes(paramName string, filename string, content []byte) SubmitOption {
	return func(req *req.Request) {
		req.SetFileBytes(paramName, filename, content)
		req.EnableForceMultipart()
	}
}

func (f Form) Submit(data map[string]any, opts ...SubmitOption) Page { //nolint:gocognit,gocyclo,cyclop,funlen
	f.t.Helper()

	if f.selection == nil {
		return Page{}
	}

	action := strings.TrimSpace(f.selection.AttrOr("action", ""))
	action = strings.TrimSpace(f.selection.AttrOr("hx-get", action))
	action = strings.TrimSpace(f.selection.AttrOr("hx-post", action))
	action = strings.TrimSpace(f.selection.AttrOr("hx-put", action))
	action = strings.TrimSpace(f.selection.AttrOr("hx-patch", action))
	action = strings.TrimSpace(f.selection.AttrOr("hx-delete", action))

	method := f.selection.AttrOr("method", http.MethodPost)
	if v := strings.TrimSpace(f.selection.AttrOr("hx-get", "")); v != "" { //nolint:nestif
		method = http.MethodGet
	} else if v := strings.TrimSpace(f.selection.AttrOr("hx-post", "")); v != "" {
		method = http.MethodPost
	} else if v := strings.TrimSpace(f.selection.AttrOr("hx-put", "")); v != "" {
		method = http.MethodPut
	} else if v := strings.TrimSpace(f.selection.AttrOr("hx-patch", "")); v != "" {
		method = http.MethodPatch
	} else if v := strings.TrimSpace(f.selection.AttrOr("hx-delete", "")); v != "" {
		method = http.MethodDelete
	} else if v := strings.TrimSpace(f.selection.AttrOr("hx-method", "")); v != "" {
		method = strings.ToUpper(v)
	}

	baseURL, err := url.Parse(f.page.httpResp.Request.URL.String())
	assert.NoError(f.t, err)

	actionURL, err := url.Parse(action)
	assert.NoError(f.t, err)

	fullURL := baseURL.ResolveReference(actionURL).String()

	// merge hx-vals into data
	if valsJSON := strings.TrimSpace(f.selection.AttrOr("hx-vals", "")); valsJSON != "" {
		var vals map[string]string
		if err = json.Unmarshal([]byte(valsJSON), &vals); err == nil {
			for k, v := range vals {
				if _, exists := data[k]; !exists {
					data[k] = v
				}
			}
		}
	}

	var resp *req.Response

	req := f.page.client.R()

	// hx-headers: parse and set custom headers.
	if headersJSON := strings.TrimSpace(f.selection.AttrOr("hx-headers", "")); headersJSON != "" {
		var headers map[string]string
		if err = json.Unmarshal([]byte(headersJSON), &headers); err == nil {
			req = req.SetHeaders(headers)
		}
	}

	if f.selection.AttrOr("hx-post", "") != "" ||
		f.selection.AttrOr("hx-get", "") != "" ||
		f.selection.AttrOr("hx-put", "") != "" ||
		f.selection.AttrOr("hx-patch", "") != "" ||
		f.selection.AttrOr("hx-delete", "") != "" {
		req = req.SetHeader("HX-Request", "true")
	}

	enc := strings.TrimSpace(f.selection.AttrOr("hx-encoding", ""))
	if enc == "" {
		enc = strings.TrimSpace(f.selection.AttrOr("enctype", ""))
	}

	for _, opt := range opts {
		opt(req)
	}

	switch strings.ToUpper(method) {
	case http.MethodGet:
		setAnyTypeQueryParams(req, data)
		resp, err = req.Get(fullURL)
	case http.MethodPut:
		if enc == multipartFormData {
			req.EnableForceMultipart()
		}

		setAnyTypeFormData(req, data)
		resp, err = req.Put(fullURL)
	case http.MethodDelete:
		setAnyTypeQueryParams(req, data)
		resp, err = req.Delete(fullURL)
	case http.MethodPatch:
		if enc == multipartFormData {
			req.EnableForceMultipart()
		}

		setAnyTypeFormData(req, data)
		resp, err = req.Patch(fullURL)
	default:
		if enc == multipartFormData {
			req.EnableForceMultipart()
		}

		setAnyTypeFormData(req, data)
		resp, err = req.Post(fullURL)
	}

	assert.NoError(f.t, err)

	return NewPage(f.t, f.page.client, resp, err)
}

func setAnyTypeFormData(r *req.Request, data map[string]any) { //nolint:varnamelen
	if r.FormData == nil {
		r.FormData = url.Values{}
	}

	for k, v := range data {
		if slice, ok := v.([]string); ok {
			for _, s := range slice {
				r.FormData.Add(k, s)
			}
		} else {
			r.FormData.Set(k, fmt.Sprint(v))
		}
	}
}

func setAnyTypeQueryParams(r *req.Request, data map[string]any) { //nolint:varnamelen
	if r.QueryParams == nil {
		r.QueryParams = url.Values{}
	}

	for k, v := range data {
		if slice, ok := v.([]string); ok {
			for _, s := range slice {
				r.QueryParams.Add(k, s)
			}
		} else {
			r.QueryParams.Set(k, fmt.Sprint(v))
		}
	}
}

// func (f Form) HasEnctype(type_ string) bool         { panic("implement me") }
// func (f Form) Target(target string) bool         { panic("implement me") }
// func (f Form) HasNoValidate() bool                  { panic("implement me") } // form novalidate property
// func (f Form) HasAcceptCharset(charset string) bool { panic("implement me") }
// func (f Form) IsPasswordField(name string) bool     { panic("implement me") }
// func (f Form) IsEmailField(name string) bool        { panic("implement me") }
// func (f Form) IsCheckbox(name string) bool          { panic("implement me") }
// func (f Form) IsRequired(name string) bool          { panic("implement me") }
// func (f Form) IsDisabled(name string) bool          { panic("implement me") }
// func (f Form) IsReadOnly(name string) bool          { panic("implement me") }
// func (f Form) HasValue(name, value string) bool     { panic("implement me") }
// func (f Form) HasHXTarget(target string) bool       { panic("implement me") }
// func (f Form) HasHXSwap(swap string) bool           { panic("implement me") }
// func (f Form) SubmitButtonCount() int               { panic("implement me") }
// func (f Form) HasCSRFToken() bool                   { panic("implement me") }

// === Modals/Dialogues ===

// func (p Page) Modal(selector string) *Modal     { panic("not implemented") }
// func (p Page) HasOpenModal() bool               { panic("not implemented") }
// func (p Page) HasAlert(text string) bool        { panic("not implemented") }
// func (p Page) HasFlash(type_, text string) bool { panic("not implemented") }

// === Pagination ===

// func (p Page) HasPagination() bool     { panic("not implemented") }
// func (p Page) Pagination() *Pagination { panic("not implemented") }
// func (p Page) HasNextPage() bool       { panic("not implemented") }
// func (p Page) HasPrevPage() bool       { panic("not implemented") }

// === Navigation/URL ===

// func (p Page) IsURL(url string, msgAndArgs ...any) bool { panic("not implemented") }

// === HTMX Specific ===

// func (p Page) IsHXResponse() bool             { panic("not implemented") }
// func (p Page) HasHXTrigger(event string) bool { panic("not implemented") }
// func (p Page) HXRedirect(url string) bool     { panic("not implemented") }
// func (p Page) HXSwap(swap string) bool        { panic("not implemented") }
//

func (p Page) Scripts() Scripts {
	return Scripts{t: p.t, page: p}
}

type Scripts struct {
	t    *testing.T
	page Page
}

// Total asserts that exactly `n` script tags with src attributes exist.
func (s Scripts) Total(total int, msgAndArgs ...any) bool {
	s.t.Helper()

	if s.page.document.Find("script[src]").Size() != total {
		return fail(s.t, "script tag count mismatch", append([]kv{
			{"actual", strconv.Itoa(s.page.document.Find("script[src]").Size())},
			{"expected", strconv.Itoa(total)},
		}, pageContext(s.page.httpResp)...), msgAndArgs...)
	}

	return true
}

// HasLibrary asserts that a specific library is loaded.
// If searchLib doesn't include a version, match any version: "htmx.org" matches "htmx.org@2.0.8".
func (s Scripts) HasLibrary(searchLib string, msgAndArgs ...any) bool {
	s.t.Helper()

	libraries := s.libraries()
	for _, loaded := range libraries {
		if strings.EqualFold(loaded, searchLib) {
			return true
		}

		if !strings.Contains(searchLib, "@") { // no version specified => match any
			libWithoutVersion := strings.Split(loaded, "@")[0]
			if strings.EqualFold(libWithoutVersion, searchLib) {
				return true
			}
		}
	}

	return fail(s.t, "loaded libraries do not contain expected", append([]kv{
		{"actual", fmt.Sprintf("%v", libraries)},
		{"expected", searchLib},
	}, pageContext(s.page.httpResp)...), msgAndArgs...)
}

// HasOnlyLibraries asserts that ONLY the specified libraries are loaded.
// func (s Scripts) HasOnlyLibraries(expected string, msgAndArgs ...any) bool {
// 	s.t.Helper()

// libraries := s.Libraries()
//
// expectedMap := make(map[string]bool)
// for _, lib := range expected {
// 	expectedMap[lib] = true
// }
//
// for _, loaded := range libraries {
// 	if !expectedMap[loaded] {
// 		return assert.Fail(s.t, fmt.Sprintf("unexpected library loaded: %s (expected: %v)", loaded, expected))
// 	}
// }
//
// return true

// panic("not implemented")
// }

// HasOnlyLocal asserts that all scripts are local (no external CDNs).
func (s Scripts) HasOnlyLocal(msgAndArgs ...any) bool {
	s.t.Helper()

	sources := s.sources()
	for _, src := range sources {
		if !s.isLocalScript(src) {
			return fail(s.t, "external script found but should be local only", append([]kv{
				{"src", src},
			}, pageContext(s.page.httpResp)...), msgAndArgs...)
		}
	}

	return true
}

func (s Scripts) isLocalScript(src string) bool {
	// local paths: "/app.js", "../lib.js", "app.js"
	if !strings.Contains(src, "://") && !strings.HasPrefix(src, "//") {
		return true
	}

	if strings.Contains(src, "://") {
		u, err := url.Parse(src)
		if err == nil && u.Host == s.page.httpResp.Request.URL.Host {
			return true
		}
	}

	return false
}

func (s Scripts) sources() []string {
	var sources []string

	s.page.document.Find("script[src]").Each(func(_ int, sel *goquery.Selection) {
		if src, exists := sel.Attr("src"); exists {
			sources = append(sources, src)
		}
	})

	return sources
}

func (s Scripts) libraries() []string {
	sources := s.sources()

	var libraries []string

	for _, src := range sources {
		lib := s.extractLibraryName(src)
		if lib != "" {
			libraries = append(libraries, lib)
		}
	}

	return libraries
}

func (s Scripts) extractLibraryName(src string) string {
	if s.isLocalScript(src) {
		parts := strings.Split(src, "/")
		filename := parts[len(parts)-1]

		extensions := []string{".min.js", ".js", ".mjs"}
		for _, ext := range extensions {
			filename = strings.TrimSuffix(filename, ext)
		}

		return filename
	}

	var (
		cdnPatterns = []*regexp.Regexp{
			regexp.MustCompile(`unpkg\.com/([^/?]+)`),
			regexp.MustCompile(`jsdelivr\.net/(?:npm/)?@?([^/?]+)`),
			regexp.MustCompile(`skypack\.dev/([^/?]+)`),
			regexp.MustCompile(`cdnjs\.cloudflare\.com/ajax/libs/([^/?]+)`),
			regexp.MustCompile(`cdn\.tailwindcss\.com`),
			regexp.MustCompile(`chromestatus\.com`),
		}
		hostPattern = regexp.MustCompile(`^https?://([^/]+)`)
	)

	for _, pattern := range cdnPatterns {
		if matches := pattern.FindStringSubmatch(src); len(matches) > 1 {
			lib := matches[1]
			return lib
		}
	}

	if matches := hostPattern.FindStringSubmatch(src); len(matches) > 1 {
		return matches[1]
	}

	return src
}

type Element struct {
	t         *testing.T
	selection *goquery.Selection
	selector  string        // css selector that produced this element (best-effort, for failure context)
	httpResp  *req.Response // response this element came from, for page URL/status context; nil if none
}

// kv builds the kv pairs shared by every Element failure: the selector
// (when known), then caller-supplied pairs, then the originating page context.
func (e Element) kv(extra ...kv) []kv {
	pairs := make([]kv, 0, 1+len(extra))
	if e.selector != "" {
		pairs = append(pairs, kv{"selector", e.selector})
	}

	pairs = append(pairs, extra...)
	pairs = append(pairs, pageContext(e.httpResp)...)

	return pairs
}

func (e Element) Find(selector string) Element {
	sel := e.selection.Find(selector)

	child := selector
	if e.selector != "" {
		child = e.selector + " " + selector
	}

	return Element{t: e.t, selection: sel, selector: child, httpResp: e.httpResp}
}

func (e Element) String() string {
	return strings.TrimSpace(e.selection.Text())
}

func (e Element) Exists(msgAndArgs ...any) bool {
	e.t.Helper()

	if e.selection == nil || e.selection.Length() == 0 {
		return fail(e.t, "element does not exist", e.kv(), msgAndArgs...)
	}

	return true
}

// Total asserts that the selection matches exactly the expected number of elements.
func (e Element) Total(expected int, msgAndArgs ...any) bool {
	e.t.Helper()

	if e.selection == nil {
		return expected == 0
	}

	if e.selection.Length() != expected {
		return fail(e.t, "element count mismatch", e.kv(
			kv{"actual", strconv.Itoa(e.selection.Length())},
			kv{"expected", strconv.Itoa(expected)},
		), msgAndArgs...)
	}

	return true
}

func (e Element) Length() int {
	return e.selection.Length()
}

// assertSingle fails the test if the selection contains more than one element.
// Returns false if ambiguous, so callers can return zero values.
func (e Element) assertSingle() bool {
	e.t.Helper()

	if e.selection == nil {
		return false
	}

	if e.selection.Length() > 1 {
		fail(e.t, "selector matched multiple elements, expected 1", e.kv(
			kv{"count", strconv.Itoa(e.selection.Length())},
			kv{"hint", "use .First(), .Last(), or .Nth() to disambiguate"},
		))

		return false
	}

	return true
}

// First returns a new Element scoped to the first match.
func (e Element) First() Element {
	return Element{t: e.t, selection: e.selection.First(), selector: e.selector, httpResp: e.httpResp}
}

// Last returns a new Element scoped to the last match.
func (e Element) Last() Element {
	return Element{t: e.t, selection: e.selection.Last(), selector: e.selector, httpResp: e.httpResp}
}

// Nth returns a new Element scoped to the match at the given index.
// Fails if index is out of range.
func (e Element) Nth(i int) Element {
	e.t.Helper()

	sel := e.selection.Eq(i)
	if sel.Length() == 0 {
		fail(e.t, "index out of range", e.kv(
			kv{"index", strconv.Itoa(i)},
			kv{"length", strconv.Itoa(e.selection.Length())},
		))

		return Element{}
	}

	return Element{t: e.t, selection: sel, selector: e.selector, httpResp: e.httpResp}
}

// === Existence ===

// func (e Element) NotExists() bool { panic("not implemented") }
// func (e Element) IsVisible() bool { panic("not implemented") }
// func (e Element) IsHidden() bool  { panic("not implemented") }
// func (e Element) IsEnabled() bool { panic("not implemented") }
// func (e Element) IsDisabled() bool {
// 	panic("not implemented")
// }

// === Select/Dropdown ===

// func (e Element) HasSelectedOption(value string) bool {
// 	panic("not implemented")
// }
// func (e Element) HasOptions(options ...string) bool {
// 	panic("not implemented")
// }
// func (e Element) HasOptionCount(count int) bool { panic("not implemented") }

// === Accessibility ===

// func (e Element) HasRole(role string) bool { panic("not implemented") }
// func (e Element) HasAccessibleName(name string) bool {
// 	panic("not implemented")
// }
// func (e Element) HasAccessibleDescription(text string) bool {
// 	panic("not implemented")
// }

func (e Element) Text() string {
	e.t.Helper()

	if !e.assertSingle() {
		return ""
	}

	return strings.TrimSpace(e.selection.Text())
}

// func (e Element) Attr(name, value string) bool  { panic("not implemented") }

func (e Element) Attr(name string) string {
	if !e.HasAttr(name) {
		return ""
	}

	return e.selection.AttrOr(name, "")
}

func (e Element) HasAttr(name string, msgAndArgs ...any) bool {
	e.t.Helper()

	if !e.assertSingle() {
		return false
	}

	if _, ok := e.selection.Attr(name); !ok {
		return fail(e.t, "element does not have attribute", e.kv(
			kv{"attribute", name},
			kv{"element", outerHTML(e.selection)},
		), msgAndArgs...)
	}

	return true
}

// func (e Element) HasAttrValue(name, value string) bool {
// 	panic("not implemented")
// }

// === Value/Input ===

// func (e Element) HasValue(value string) bool { panic("not implemented") }
// func (e Element) IsEmpty() bool              { panic("not implemented") }
// func (e Element) IsEditable() bool           { panic("not implemented") }
// func (e Element) IsReadOnly() bool           { panic("not implemented") }
// func (e Element) IsRequired() bool           { panic("not implemented") }
// func (e Element) HasPlaceholder(text string) bool {
// 	panic("not implemented")
// }

// === CSS/Classes ===

// func (e Element) HasClass(class string) bool { panic("not implemented") }
// func (e Element) HasCSS(prop, value string) bool {
// 	panic("not implemented")
// }
// func (e Element) HasID(id string) bool { panic("not implemented") }

// === Count/Collections ===

// func (e Element) HasCount(n int) bool { panic("not implemented") }

// === HTMX Attributes ===

// func (e Element) HasHXGet(url string) bool    { panic("not implemented") }
// func (e Element) HasHXPost(url string) bool   { panic("not implemented") }
// func (e Element) HasHXPut(url string) bool    { panic("not implemented") }
// func (e Element) HasHXDelete(url string) bool { panic("not implemented") }
// func (e Element) HasHXTarget(target string) bool {
// 	panic("not implemented")
// }
// func (e Element) HasHXSwap(swap string) bool { panic("not implemented") }
// func (e Element) HasHXTrigger(event string) bool {
// 	panic("not implemented")
// }

// === Form Specific ===

// func (e Element) IsInput() bool    { panic("not implemented") }
// func (e Element) IsButton() bool   { panic("not implemented") }
// func (e Element) IsSelect() bool   { panic("not implemented") }
// func (e Element) IsCheckbox() bool { panic("not implemented") }
// func (e Element) IsRadio() bool    { panic("not implemented") }
// func (e Element) IsFile() bool     { panic("not implemented") }

func (e Element) Contains(contains string, msgAndArgs ...any) bool {
	e.t.Helper()

	actual := e.Text()
	if strings.Contains(actual, contains) {
		return true
	}

	return fail(e.t, "selection does not contain expected text", e.kv(
		kv{"actual", fmt.Sprintf("%q", actual)},
		kv{"expected", fmt.Sprintf("%q", contains)},
		kv{"element", outerHTML(e.selection)},
	), msgAndArgs...)
}

func (e Element) NotContains(notContains string, msgAndArgs ...any) bool {
	e.t.Helper()

	if strings.Contains(e.Text(), notContains) {
		return fail(e.t, "selection contains text but should not", e.kv(
			kv{"text", fmt.Sprintf("%q", notContains)},
			kv{"element", outerHTML(e.selection)},
		), msgAndArgs...)
	}

	return true
}

func (e Element) Equals(expected string, msgAndArgs ...any) bool {
	e.t.Helper()

	actual := e.Text()
	if expected != actual {
		return fail(e.t, "selection does not equal expected", e.kv(
			kv{"actual", fmt.Sprintf("%q", actual)},
			kv{"expected", fmt.Sprintf("%q", expected)},
			kv{"element", outerHTML(e.selection)},
		), msgAndArgs...)
	}

	return true
}

func (e Element) NotEquals(expected string, msgAndArgs ...any) bool {
	e.t.Helper()

	actual := e.Text()
	if expected == actual {
		return fail(e.t, "selection equals but should not", e.kv(
			kv{"actual", fmt.Sprintf("%q", actual)},
			kv{"element", outerHTML(e.selection)},
		), msgAndArgs...)
	}

	return true
}

func (e Element) Is(elemType string, msgAndArgs ...any) bool {
	e.t.Helper()

	if !e.assertSingle() {
		return false
	}

	if elemType == "" {
		return false
	}

	if e.selection.Length() == 0 {
		return fail(e.t, "element not found", e.kv(
			kv{"expected", elemType},
		), msgAndArgs...)
	}

	actualTag := goquery.NodeName(e.selection)
	if strings.EqualFold(actualTag, elemType) {
		return true
	}

	return fail(e.t, "element tag mismatch", e.kv(
		kv{"actual", actualTag},
		kv{"expected", elemType},
		kv{"element", outerHTML(e.selection)},
	), msgAndArgs...)
}

func (e Element) AttrIs(attrName string, value string, msgAndArgs ...any) bool {
	e.t.Helper()

	if !e.assertSingle() {
		return false
	}

	if value == e.selection.AttrOr(attrName, "") {
		return true
	}

	return fail(e.t, "attribute value mismatch", e.kv(
		kv{"attribute", attrName},
		kv{"actual", fmt.Sprintf("%q", e.selection.AttrOr(attrName, ""))},
		kv{"expected", fmt.Sprintf("%q", value)},
		kv{"element", outerHTML(e.selection)},
	), msgAndArgs...)
}

type Table struct {
	t     *testing.T
	table *goquery.Selection
	page  Page
}

// kv builds the kv pairs shared by every Table failure: the table
// identifier, then caller-supplied pairs, then the originating page context.
func (t Table) kv(extra ...kv) []kv {
	pairs := make([]kv, 0, 1+len(extra))
	pairs = append(pairs, kv{"table", tableIdentifier(t.table)})
	pairs = append(pairs, extra...)
	pairs = append(pairs, pageContext(t.page.httpResp)...)

	return pairs
}

func (p Page) Table(idOrCaption string) Table {
	for _, selector := range []string{
		"table#" + idOrCaption,
		"table:has(caption:contains('" + idOrCaption + "'))",
		"table[data-testid='" + idOrCaption + "']",
		"table[data-cy='" + idOrCaption + "']",
		idOrCaption,
	} {
		s := p.document.Find(selector)
		if s.Length() == 1 {
			return Table{t: p.t, table: s, page: p}
		}
	}

	p.t.Helper()
	fail(p.t, "no unique table found", []kv{
		{"selector", idOrCaption},
		{"tables", formatList(collectTableIdentifiers(p.document))},
	})

	return Table{t: p.t, table: nil, page: p}
}

func (t Table) Exists(msgAndArgs ...any) bool {
	t.t.Helper()

	if t.table == nil || t.table.Length() == 0 {
		return fail(t.t, "table does not exist", t.kv(), msgAndArgs...)
	}

	return true
}

func (t Table) Empty(msgAndArgs ...any) bool {
	t.t.Helper()

	if rows := t.table.Find("tr"); rows.Length() >= 0 {
		if strings.TrimSpace(rows.Text()) == "" {
			return true
		}

		td := rows.Find("td")
		if _, ok := td.Attr("colspan"); ok && td.Length() == 1 {
			return true
		}
	}

	return fail(t.t, "table is not empty but should be", t.kv(), msgAndArgs...)
}

func (t Table) NotEmpty(msgAndArgs ...any) bool {
	t.t.Helper()

	if rows := t.table.Find("tr"); rows.Length() >= 0 {
		if strings.TrimSpace(rows.Text()) == "" {
			return fail(t.t, "table is empty but should not be", t.kv(), msgAndArgs...)
		}

		td := rows.Find("td")
		if _, ok := td.Attr("colspan"); ok && td.Length() == 1 {
			return fail(t.t, "table is empty but should not be", t.kv(), msgAndArgs...)
		}
	}

	return true
}

// Headers asserts that the table's column headers match exactly.
// Looks in <thead> first, falls back to first <tr> with <th> cells.
// Does NOT consider <th scope="row"> in <tbody>, those are row labels.
//
// For convenience at the call site, it does not accept msgAndArgs parameter,
// so wrapping the headers in a slice is not required.
func (t Table) Headers(headers ...string) bool {
	t.t.Helper()

	if t.table == nil {
		return false
	}

	headerCells := t.table.Find("thead tr:first-child").Find("th, td")
	if headerCells.Length() == 0 { // fallback: if no thead first row is header
		headerCells = t.table.Find("tr:first-child").Find("th")
	}

	if headerCells.Length() != len(headers) {
		return fail(t.t, "table header cell count mismatch", t.kv(
			kv{"actual", strconv.Itoa(headerCells.Length())},
			kv{"expected", strconv.Itoa(len(headers))},
		))
	}

	for i, expected := range headers {
		actual := strings.TrimSpace(headerCells.Eq(i).Text())
		if actual != expected {
			return fail(t.t, "table header mismatch", t.kv(
				kv{"index", strconv.Itoa(i)},
				kv{"actual", fmt.Sprintf("%q", actual)},
				kv{"expected", fmt.Sprintf("%q", expected)},
			))
		}
	}

	return true
}

// Rows asserts the table has exactly n data rows.
// Counts <tbody><tr> only. Excludes <thead> and <tfoot>.
// Falls back to <tr> when no <tbody> present.
func (t Table) Rows(n int, msgAndArgs ...any) bool { //nolint:varnamelen
	t.t.Helper()

	if t.table == nil {
		return n == 0
	}

	rows := t.table.Find("tbody tr")
	if rows.Length() == 0 && t.table.Find("thead").Length() == 0 {
		rows = t.table.Find("tr")
	}

	if rows.Length() != n {
		return fail(t.t, "table row count mismatch", t.kv(
			kv{"actual", strconv.Itoa(rows.Length())},
			kv{"expected", strconv.Itoa(n)},
		), msgAndArgs...)
	}

	return true
}

// Cols asserts the table has exactly n columns.
// Counts from <thead> cells first, falls back to first data row cells.
// Colspan cells count as 1, not the spanned value.
func (t Table) Cols(n int, msgAndArgs ...any) bool { //nolint:varnamelen
	t.t.Helper()

	if t.table == nil {
		return n == 0
	}

	cells := t.table.Find("thead tr:first-child").Find("th, td")
	if cells.Length() == 0 {
		cells = t.table.Find("tr:first-child").Find("th, td")
	}

	if cells.Length() != n {
		return fail(t.t, "table column count mismatch", t.kv(
			kv{"actual", strconv.Itoa(cells.Length())},
			kv{"expected", strconv.Itoa(n)},
		), msgAndArgs...)
	}

	return true
}

func (t Table) RowCount() int {
	if t.table == nil {
		return 0
	}

	rows := t.table.Find("tbody tr")
	if rows.Length() == 0 && t.table.Find("thead").Length() == 0 {
		rows = t.table.Find("tr")
	}

	return rows.Length()
}

func (t Table) ColCount() int {
	if t.table == nil {
		return 0
	}

	cells := t.table.Find("thead tr:first-child").Find("th, td")
	if cells.Length() == 0 {
		cells = t.table.Find("tr:first-child").Find("th, td")
	}

	return cells.Length()
}

func (t Table) Row(index int) Row {
	if t.table == nil {
		return Row{Element: Element{t: t.t, selection: nil, selector: "", httpResp: t.page.httpResp}}
	}

	rows := t.table.Find("tbody tr")
	if rows.Length() == 0 && t.table.Find("thead").Length() == 0 {
		rows = t.table.Find("tr")
	}

	if index < 0 || index >= rows.Length() {
		return Row{Element: Element{t: t.t, selection: nil, selector: "", httpResp: t.page.httpResp}}
	}

	return Row{
		Element: Element{
			t:         t.t,
			selection: rows.Eq(index),
			selector:  "",
			httpResp:  t.page.httpResp,
		},
	}
}

type Row struct {
	Element
}

func (r Row) Col(col int) Element {
	if r.selection == nil {
		return Element{t: r.t, selection: nil, selector: r.selector, httpResp: r.httpResp}
	}

	cells := r.selection.Find("th, td")
	if col < 0 || col >= cells.Length() {
		return Element{t: r.t, selection: nil, selector: r.selector, httpResp: r.httpResp}
	}

	return Element{t: r.t, selection: cells.Eq(col), selector: r.selector, httpResp: r.httpResp}
}

// func (t Table) HasRow(texts ...string) bool { panic("not implemented") }
// func (t Table) HasHeader(text string) bool  { panic("not implemented") }
// func (t Table) HasHeader
// func (t Table) HasBody
// func (t Table) HasFoot

// === Modal Helper ===

// type Modal struct {
// 	*Element
// }
// func (m *Modal) Close() *Page               { panic("not implemented") }
// func (m *Modal) Confirm() *Page             { panic("not implemented") }
// func (m *Modal) Cancel() *Page              { panic("not implemented") }
// func (m *Modal) HasTitle(title string) bool { panic("not implemented") }
// func (m *Modal) HasBody(text string) bool   { panic("not implemented") }

// === Pagination Helper ===

// type Pagination struct {
// 	*Element
// }
// func (pg *Pagination) NextPage() *Page { panic("not implemented") }
// func (pg *Pagination) PrevPage() *Page { panic("not implemented") }
// func (pg *Pagination) HasCurrentPage(n int) bool {
// 	panic("not implemented")
// }
//
// func (pg *Pagination) HasTotalPages(n int) bool {
// 	panic("not implemented")
// }

const (
	// unnamedIdentifier is returned by form/table identifier helpers when the
	// element has no useful id.
	unnamedIdentifier = "<unnamed>"
	// multipartFormData is the enctype value for multipart form submissions.
	multipartFormData = "multipart/form-data"
	// htmlSnippetLen caps how many runes of an element's outer HTML are dumped.
	htmlSnippetLen = 300
)

// outerHTML renders the matched nodes' outer HTML, truncated for log output.
// Returns "<no match>" for an empty selection so failures stay self-explanatory.
func outerHTML(sel *goquery.Selection) string {
	if sel == nil || sel.Length() == 0 {
		return "<no match>"
	}

	var b strings.Builder

	sel.Each(func(_ int, s *goquery.Selection) {
		_ = html.Render(&b, s.Nodes[0])
	})

	return truncate(b.String(), htmlSnippetLen)
}

// collectAttrValues returns the non-empty values of attr from every node
// matching selector. Used to list present testids / data-cy in not-found output.
func collectAttrValues(doc *goquery.Document, selector, attr string) []string {
	var out []string

	doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
		if v, ok := s.Attr(attr); ok && v != "" {
			out = append(out, v)
		}
	})

	return out
}

// formIdentifier returns the most useful identifier for a form: name, id,
// data-testid, data-cy, action, or unnamedIdentifier.
func formIdentifier(sel *goquery.Selection) string {
	if sel == nil || sel.Length() == 0 {
		return unnamedIdentifier
	}

	for _, attr := range []string{"name", "id", "data-testid", "data-cy"} {
		if v, ok := sel.Attr(attr); ok && v != "" {
			return v
		}
	}

	if a, ok := sel.Attr("action"); ok && a != "" {
		return a
	}

	return unnamedIdentifier
}

// collectFormIdentifiers lists every form on the page by identifier.
func collectFormIdentifiers(doc *goquery.Document) []string {
	var out []string

	doc.Find("form").Each(func(_ int, s *goquery.Selection) {
		out = append(out, formIdentifier(s))
	})

	return out
}

// collectFieldNames lists the [name] of every form field within the selection.
func collectFieldNames(sel *goquery.Selection) []string {
	var out []string

	sel.Find("input[name], textarea[name], select[name], button[name]").Each(func(_ int, s *goquery.Selection) {
		if v, ok := s.Attr("name"); ok && v != "" {
			out = append(out, v)
		}
	})

	return out
}

// tableIdentifier returns the most useful identifier for a table: id,
// data-testid, data-cy, caption, or unnamedIdentifier.
func tableIdentifier(sel *goquery.Selection) string {
	if sel == nil || sel.Length() == 0 {
		return unnamedIdentifier
	}

	for _, attr := range []string{"id", "data-testid", "data-cy"} {
		if v, ok := sel.Attr(attr); ok && v != "" {
			return v
		}
	}

	if c := strings.TrimSpace(sel.Find("caption").Text()); c != "" {
		return c
	}

	return unnamedIdentifier
}

// collectTableIdentifiers lists every table on the page by identifier.
func collectTableIdentifiers(doc *goquery.Document) []string {
	var out []string

	doc.Find("table").Each(func(_ int, s *goquery.Selection) {
		out = append(out, tableIdentifier(s))
	})

	return out
}

// describeControl renders a button/input as a short "<tag type=t name=n>" token.
func describeControl(sel *goquery.Selection) string {
	tag := strings.ToLower(goquery.NodeName(sel))
	name, _ := sel.Attr("name")
	typ, _ := sel.Attr("type")

	switch {
	case typ != "" && name != "":
		return fmt.Sprintf("<%s type=%s name=%s>", tag, typ, name)
	case typ != "":
		return fmt.Sprintf("<%s type=%s>", tag, typ)
	case name != "":
		return fmt.Sprintf("<%s name=%s>", tag, name)
	default:
		return fmt.Sprintf("<%s>", tag)
	}
}

// collectButtons lists every button/input within the selection, for "found:"
// output when a form lacks a submit button.
func collectButtons(sel *goquery.Selection) []string {
	var out []string

	sel.Find("button, input").Each(func(_ int, s *goquery.Selection) {
		out = append(out, describeControl(s))
	})

	return out
}

func assertHTMLIsValid(t *testing.T, htmlBody *bytes.Buffer) {
	t.Helper()

	// Note: There is an HTML parser in the Go standard library,
	// but it is too lenient:
	// It is more like a browser accepting broken HTML as valid.
	// The XML parser can be configured to parse HTML, see:
	// https://stackoverflow.com/questions/31788 134/html-validation-with-golang/52410528#52410528

	// XML does not support HTMX
	// `::` violates XML's fundamental naming rules
	// (colons are reserved for namespaces like xmlns:prefix).
	// Transform hx-on::event to hx-on--event (XML-safe)
	transformed := regexp.MustCompile(`hx-on::([a-z-]+)`).ReplaceAll(
		htmlBody.Bytes(),
		[]byte("hx-on--$1"),
	)

	// Escape XML special characters in ALL attribute values
	// This sanitises inline JavaScript.
	// Process & first, then < and >
	transformed = regexp.MustCompile(`="([^"]*)"`).ReplaceAllFunc(
		transformed,
		func(match []byte) []byte {
			content := string(match[2 : len(match)-1]) // Remove =" and "

			// Only escape if contains problematic characters
			if strings.ContainsAny(content, "<>&") {
				// Order matters: escape & first to avoid double-escaping
				content = strings.ReplaceAll(content, "&", "&amp;")
				content = strings.ReplaceAll(content, "<", "&lt;")
				content = strings.ReplaceAll(content, ">", "&gt;")
			}

			return []byte(`="` + content + `"`)
		},
	)

	// Remove all script tag content (keep the tags but empty them)
	transformed = regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`).ReplaceAll(
		transformed,
		[]byte(""),
	)
	transformed = regexp.MustCompile(`<script[^>]*/>`).ReplaceAll(
		transformed,
		[]byte(""),
	)

	decoder := xml.NewDecoder(bytes.NewReader(transformed))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	for {
		_, err := decoder.Token()
		if err == nil {
			continue
		}

		if errors.Is(err, io.EOF) {
			break
		}

		xerr := &xml.SyntaxError{}
		errors.As(err, &xerr)

		lines := strings.Split(htmlBody.String(), "\n")
		for i := xerr.Line - 3; i <= xerr.Line+3; i++ { //nolint:mnd // show 3 lines of context
			t.Log(lines[i])
		}

		t.Fatalf("Error parsing html: %s", err)
	}
}

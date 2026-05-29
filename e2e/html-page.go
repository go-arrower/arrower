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
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/assert"
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

func (p Page) IsRedirected(msgAndArgs ...any) bool {
	p.t.Helper()

	isRedirectStatus := p.httpResp.StatusCode > http.StatusMultipleChoices && p.httpResp.StatusCode < http.StatusBadRequest
	isNotModified := p.httpResp.StatusCode == http.StatusNotModified // not a "real" redirect
	hasLocation := p.httpResp.Header.Get("Location") != ""

	if isRedirectStatus && !isNotModified && hasLocation {
		return true
	}

	return assert.Fail(p.t,
		fmt.Sprintf("page has status code: %d, should be: 3xx (Redirect)", p.httpResp.StatusCode),
		msgAndArgs...)
}

func (p Page) RedirectsTo(path string, msgAndArgs ...any) bool {
	p.t.Helper()

	if !p.IsRedirected() || p.httpResp.Header.Get("Location") != path {
		return assert.Fail(p.t,
			fmt.Sprintf("page redirects to: %s, should be: %s", fmtEmpty(p.httpResp.Header.Get("Location")), path),
			msgAndArgs...)
	}

	return true
}

func (p Page) Find(selector string) Element {
	sel := p.document.Find(selector)
	return Element{t: p.t, selection: sel}
}

func (p Page) TestID(id string) Element {
	p.t.Helper()

	matches := p.document.Find(`[data-testid="` + id + `"]`)
	if matches.Length() > 0 {
		if matches.Length() > 1 {
			p.t.Logf(
				"WARNING: duplicate testid '%s' found %d times - using first match. Testids should be unique.",
				id, matches.Length(),
			)
		}

		return Element{t: p.t, selection: matches.First()}
	}

	matches = p.document.Find(`[data-cy="` + id + `"]`)
	if matches.Length() > 0 {
		if matches.Length() > 1 {
			p.t.Logf(
				"WARNING: duplicate testid '%s' found %d times - using first match. Testids should be unique.",
				id, matches.Length(),
			)
		}

		return Element{t: p.t, selection: matches.First()}
	}

	assert.Fail(p.t, "no element found with test id: "+id)

	return Element{t: p.t, selection: nil}
}

type Form struct {
	t         *testing.T
	page      Page
	selection *goquery.Selection
}

// Form finds a unique form element by name, id, action, or htmx action attribute.
// Selector precedence (first unique match wins): name > id > action > hx-*
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
	assert.Fail(p.t, "no unique form found with name, id, or action: "+nameOrIDOrAction)

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
		return assert.Fail(f.t, "form does not exist", msgAndArgs...)
	}

	return true
}

func (f Form) Method(method string, msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	if m, ok := f.selection.Attr("method"); !ok || !strings.EqualFold(m, method) {
		return assert.Fail(f.t,
			fmt.Sprintf("form method is: %s, should be: %s", m, method),
			msgAndArgs...)
	}

	return true
}

func (f Form) Action(action string, msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	if a, ok := f.selection.Attr("action"); !ok || a != action {
		return assert.Fail(f.t,
			fmt.Sprintf("form action is: %s, should be: %s", a, action),
			msgAndArgs...)
	}

	return true
}

func (f Form) Total(total int, msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	if s := f.selection.Find("input, textarea, select, button"); s.Length() != total {
		return assert.Fail(f.t,
			fmt.Sprintf("form has %d elements, should have %d", s.Length(), total),
			msgAndArgs...)
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

	return assert.Fail(f.t, "form has no field: "+name, msgAndArgs...)
}

func (f Form) HasSubmit(msgAndArgs ...any) bool {
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	submits := f.selection.Find("input[type='submit'], input[type='SUBMIT'], button[type='submit'], button[type='SUBMIT']")
	if submits.Length() == 0 {
		return assert.Fail(f.t, "form has no submit button", msgAndArgs...)
	}

	// check for duplicate unnamed submits (more than 1 unnamed submit is ambiguous)
	unnamedCount := 0

	submits.Each(func(_ int, sel *goquery.Selection) {
		if _, hasName := sel.Attr("name"); !hasName {
			unnamedCount++
		}
	})

	if unnamedCount > 1 {
		return assert.Fail(f.t,
			fmt.Sprintf("form has %d unnamed submit buttons (must be unique or named)", unnamedCount),
			msgAndArgs...)
	}

	return true
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
		if enc == "multipart/form-data" {
			req.EnableForceMultipart()
		}

		setAnyTypeFormData(req, data)
		resp, err = req.Put(fullURL)
	case http.MethodDelete:
		setAnyTypeQueryParams(req, data)
		resp, err = req.Delete(fullURL)
	case http.MethodPatch:
		if enc == "multipart/form-data" {
			req.EnableForceMultipart()
		}

		setAnyTypeFormData(req, data)
		resp, err = req.Patch(fullURL)
	default:
		if enc == "multipart/form-data" {
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
			return assert.Fail(f.t,
				fmt.Sprintf("form has validation errors (found: %s), should not have any", selector),
				msgAndArgs...)
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

	return assert.Fail(f.t, "form has no validation errors, should have", msgAndArgs...)
}

func (f Form) HasFieldError(name string, msgAndArgs ...any) bool { //nolint:gocognit,gocyclo,cyclop
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	field := f.selection.Find(fmt.Sprintf("[name='%s']", name))
	if field.Length() != 1 {
		return assert.Fail(f.t, "field not found: "+name, msgAndArgs...)
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

	return assert.Fail(f.t, fmt.Sprintf("field '%s' has no validation errors", name), msgAndArgs...)
}

func (f Form) HasFieldValue(name string, value string, msgAndArgs ...any) bool { //nolint:gocognit,gocyclo,cyclop
	f.t.Helper()

	if f.selection == nil {
		return false
	}

	fields := f.selection.Find(fmt.Sprintf("[name='%s']", name))
	if fields.Length() == 0 {
		return assert.Fail(f.t, "field not found: "+name, msgAndArgs...)
	}

	allValues := make([]string, len(fields.Nodes))

	for i := range fields.Nodes {
		field := fields.Eq(i)

		var fieldValue string

		if field.Is("input[type='checkbox'], input[type='CHECKBOX']") { //nolint:gocritic,nestif,lll // fp field.Is() doesn't fit switch pattern
			if _, hasChecked := field.Attr("checked"); hasChecked {
				fieldValue = field.AttrOr("value", "on")
			}
		} else if field.Is("input[type='radio'], input[type='RADIO']") {
			if _, hasChecked := field.Attr("checked"); hasChecked {
				fieldValue = field.AttrOr("value", "")
			}
		} else if field.Is("textarea") {
			fieldValue = field.Text()
		} else if field.Is("select") {
			selectedOption := field.Find("option[selected]")
			if selectedOption.Length() > 0 {
				fieldValue = selectedOption.AttrOr("value", selectedOption.Text())
			} else {
				firstOption := field.Find("option").First()
				if firstOption.Length() > 0 {
					fieldValue = firstOption.AttrOr("value", firstOption.Text())
				}
			}
		} else {
			fieldValue = field.AttrOr("value", "")
		}

		allValues[i] = fieldValue

		if fieldValue == value {
			return true
		}
	}

	valuesStr := fmt.Sprintf("%v", allValues)
	if len(allValues) == 1 {
		valuesStr = fmt.Sprintf("'%s'", allValues[0])
	}

	return assert.Fail(f.t,
		fmt.Sprintf("field '%s' has value(s) %s, should be '%s'", name, valuesStr, value),
		msgAndArgs...)
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

// === Tables ===

// func (p Page) Table(selector string) *Table { panic("not implemented") }
// func (p Page) TableWithCaption(text string) *Table {
// 	panic("not implemented")
// }

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

func (p Page) HasQueryParam(key, value string, msgAndArgs ...any) bool {
	p.t.Helper()

	if p.httpResp.Response.Request.URL.Query().Get(key) != value {
		return assert.Fail(p.t,
			fmt.Sprintf("query param %s is: %s, should be: %s",
				key, fmtEmpty(p.httpResp.Response.Request.URL.Query().Get(key)), value),
			msgAndArgs...)
	}

	return true
}

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
		return assert.Fail(s.t,
			fmt.Sprintf("expected %d script tags, found %d", total, s.page.document.Find("script[src]").Size()),
			msgAndArgs...)
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

	return assert.Fail(s.t, fmt.Sprintf("libraries %v do not contain: %s", libraries, searchLib), msgAndArgs...)
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
			return assert.Fail(s.t, "external script found: "+src, msgAndArgs...)
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
}

func (e Element) String() string {
	return strings.TrimSpace(e.selection.Text())
}

func (e Element) Exists(msgAndArgs ...any) bool {
	e.t.Helper()

	if e.selection == nil || e.selection.Length() == 0 {
		return assert.Fail(e.t, "element does not exist", msgAndArgs...)
	}

	return true
}

func (e Element) Find(selector string) Element {
	sel := e.selection.Find(selector)
	return Element{t: e.t, selection: sel}
}

// Total asserts that the selection matches exactly the expected number of elements.
func (e Element) Total(expected int, msgAndArgs ...any) bool {
	e.t.Helper()

	if e.selection == nil {
		return expected == 0
	}

	if e.selection.Length() != expected {
		return assert.Fail(e.t,
			fmt.Sprintf("element count is %d, should be %d", e.selection.Length(), expected),
			msgAndArgs...)
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
		assert.Fail(e.t, fmt.Sprintf(
			"selector matched %d elements, expected 1. Use .First(), .Last(), or .Nth() to disambiguate.",
			e.selection.Length(),
		))

		return false
	}

	return true
}

// First returns a new Element scoped to the first match.
func (e Element) First() Element {
	return Element{t: e.t, selection: e.selection.First()}
}

// Last returns a new Element scoped to the last match.
func (e Element) Last() Element {
	return Element{t: e.t, selection: e.selection.Last()}
}

// Nth returns a new Element scoped to the match at the given index.
// Fails if index is out of range.
func (e Element) Nth(i int) Element {
	e.t.Helper()

	sel := e.selection.Eq(i)
	if sel.Length() == 0 {
		assert.Fail(e.t, fmt.Sprintf("index %d out of range (selection has %d elements)", i, e.selection.Length()))
		return Element{}
	}

	return Element{t: e.t, selection: sel}
}

// === Existence ===

// func (e Element) Exists() bool    { panic("not implemented") }
// func (e Element) NotExists() bool { panic("not implemented") }
// func (e Element) IsVisible() bool { panic("not implemented") }
// func (e Element) IsHidden() bool  { panic("not implemented") }
// func (e Element) IsEnabled() bool { panic("not implemented") }
// func (e Element) IsDisabled() bool {
// 	panic("not implemented")
// }
// func (e Element) IsSelected() bool {
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

func (e Element) HasText(text string, msgAndArgs ...any) bool {
	e.t.Helper()

	if strings.Contains(strings.TrimSpace(e.Text()), text) {
		return true
	}

	return assert.Fail(e.t, "element does not contain: "+text, msgAndArgs...)
}

func (e Element) TextEquals(text string, msgAndArgs ...any) bool {
	e.t.Helper()

	if assert.Equal(e.t, text, strings.TrimSpace(e.Text())) {
		return true
	}

	return assert.Fail(e.t, "element does not equal: "+text, msgAndArgs...)
}

func (e Element) TextContains(text string, msgAndArgs ...any) bool {
	e.t.Helper()

	if strings.Contains(strings.TrimSpace(e.Text()), text) {
		return true
	}

	return assert.Fail(e.t, "element does not contain: "+text, msgAndArgs...)
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
		return assert.Fail(e.t, "element does not have attribute: "+name, msgAndArgs...)
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

// === Collections ===

// func (e Element) Length() int {
// 	return e.Selection.Length()
// }
// func (e Element) Each(fn func(*Element)) { panic("not implemented") }

func (e Element) Contains(contains string, msgAndArgs ...any) bool {
	e.t.Helper()

	if strings.Contains(strings.TrimSpace(e.Text()), contains) {
		return true
	}

	return assert.Fail(e.t, "selection does not contain: "+contains, msgAndArgs...)
}

func (e Element) NotContains(notContains string, msgAndArgs ...any) bool {
	e.t.Helper()

	if strings.Contains(strings.TrimSpace(e.Text()), notContains) {
		return assert.Fail(e.t, "selection contains: "+notContains+", should not be", msgAndArgs...)
	}

	return true
}

func (e Element) Equals(expected string, msgAndArgs ...any) bool {
	e.t.Helper()

	if expected != strings.TrimSpace(e.Text()) {
		return assert.Fail(e.t, "selection does not equal: "+expected, msgAndArgs...)
	}

	return true
}

func (e Element) NotEquals(expected string, msgAndArgs ...any) bool {
	e.t.Helper()

	if expected == strings.TrimSpace(e.Text()) {
		return assert.Fail(e.t, "selection does equal: "+expected+", should not", msgAndArgs...)
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
		return assert.Fail(e.t, fmt.Sprintf("element %s not found", elemType), msgAndArgs...)
	}

	actualTag := goquery.NodeName(e.selection)
	if strings.EqualFold(actualTag, elemType) {
		return true
	}

	return assert.Fail(e.t,
		fmt.Sprintf("element is '%s', should be '%s'", actualTag, elemType),
		msgAndArgs...)
}

func (e Element) AttrIs(attrName string, value string, msgAndArgs ...any) bool {
	e.t.Helper()

	if !e.assertSingle() {
		return false
	}

	if value == e.selection.AttrOr(attrName, "") {
		return true
	}

	return assert.Fail(e.t, "selection does not contain attribute: "+attrName+"="+value, msgAndArgs...)
}

// === Table Helper ===

// type Table struct {
// 	*Element
// }
// func (t *Table) HasRowCount(count int) bool  { panic("not implemented") }
// func (t *Table) HasRow(texts ...string) bool { panic("not implemented") }
// func (t *Table) Row(index int) Element      { panic("not implemented") }
// func (t *Table) Cell(row, col int) Element  { panic("not implemented") }
// func (t *Table) HasHeader(text string) bool  { panic("not implemented") }
// func (t *Table) HasHeaders(headers ...string) bool {
// 	panic("not implemented")
// }
// func (t *Table) HasColumnCount(count int) bool { panic("not implemented") }

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

// === Page-Level Assertions ===

// Title asserts the page's <title> equals the expected value.
func (p Page) Title(title string, msgAndArgs ...any) bool {
	p.t.Helper()
	actual := p.document.Find("title").Text()

	if actual != title {
		return assert.Fail(p.t, fmt.Sprintf("page title is: %s, should be: %s", fmtEmpty(actual), title), msgAndArgs...)
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

	return assert.Fail(p.t, fmt.Sprintf("page has no link with text %q and href %q", text, href), msgAndArgs...)
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

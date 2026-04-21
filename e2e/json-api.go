//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/assert"
)

// Document represents an HTTP response from a JSON API endpoint.
// It provides assertions for status codes, headers, cookies, and JSON body content.
// Use suite.Get(), suite.Post(), etc. to obtain a Document.
type Document struct {
	response
	body json.RawMessage
}

// NewJSON creates a Document from an HTTP response.
// It parses the body as JSON. The test fails if the body is not valid JSON.
//
// It takes in the error so that callers can save one assertion line in their test case:
//
//	resp, err = client.R().Get("some/resource")
//	doc := e2e.NewJSON(t, client, resp, err)
func NewJSON(t *testing.T, client *req.Client, resp *req.Response, err error) Document {
	t.Helper()

	assert.NotEmpty(t, client, "client is nil")
	assert.NotEmpty(t, resp, "response is nil")
	assert.NoError(t, err)

	raw := json.RawMessage(resp.String())
	assert.NoError(t, json.Unmarshal(raw, &raw), "response body is not valid JSON")

	return Document{
		response: response{
			t:        t,
			client:   client,
			httpResp: resp,
		},
		body: raw,
	}
}

// IsArray asserts that the response body is a JSON array.
func (r Document) IsArray(msgAndArgs ...any) bool {
	r.t.Helper()

	var arr []any
	if err := json.Unmarshal(r.body, &arr); err != nil {
		return assert.Fail(r.t, "response body is not a JSON array", msgAndArgs...)
	}

	return true
}

// IsObject asserts that the response body is a JSON object.
func (r Document) IsObject(msgAndArgs ...any) bool {
	r.t.Helper()

	var obj map[string]any
	if err := json.Unmarshal(r.body, &obj); err != nil {
		return assert.Fail(r.t, "response body is not a JSON object", msgAndArgs...)
	}

	return true
}

// HasField asserts that the JSON body contains the given field.
// Supports dot notation paths: "name", "user.email", "items.0.id".
func (r Document) HasField(path string, msgAndArgs ...any) bool {
	r.t.Helper()

	_, ok := r.valueAtPath(path)
	if !ok {
		return assert.Fail(r.t, "response does not have field: "+path, msgAndArgs...)
	}

	return true
}

// NotHasField asserts that the JSON body does not contain the given field.
// Supports dot notation paths: "name", "user.email", "items.0.id".
func (r Document) NotHasField(path string, msgAndArgs ...any) bool {
	r.t.Helper()

	_, ok := r.valueAtPath(path)
	if ok {
		return assert.Fail(r.t, "response has field: "+path+", should not", msgAndArgs...)
	}

	return true
}

// FieldEquals asserts that the JSON field at the given path equals the expected value.
// Path uses dot notation: "user.name", "items.0.id".
// Handles int → float64 coercion automatically (JSON numbers are float64).
func (r Document) FieldEquals(path string, expected any, msgAndArgs ...any) bool {
	r.t.Helper()

	actual, ok := r.valueAtPath(path)
	if !ok {
		return assert.Fail(r.t, "field not found at path: "+path, msgAndArgs...)
	}

	if !jsonEqual(actual, expected) {
		return assert.Fail(r.t,
			fmt.Sprintf("field at %s is: %v, should be: %v", path, actual, expected),
			msgAndArgs...)
	}

	return true
}

// FieldContains asserts that the JSON field at the given path contains the given substring.
func (r Document) FieldContains(path string, text string, msgAndArgs ...any) bool {
	r.t.Helper()

	actual, ok := r.valueAtPath(path)
	if !ok {
		return assert.Fail(r.t, "field not found at path: "+path, msgAndArgs...)
	}

	str, isString := actual.(string)
	if !isString {
		return assert.Fail(r.t, fmt.Sprintf("field at %s is not a string", path), msgAndArgs...)
	}

	if !strings.Contains(str, text) {
		return assert.Fail(r.t,
			fmt.Sprintf("field at %s does not contain: %s", path, text),
			msgAndArgs...)
	}

	return true
}

// FieldLen asserts that the array or object at the given path has the expected length.
func (r Document) FieldLen(path string, expected int, msgAndArgs ...any) bool {
	r.t.Helper()

	actual, ok := r.valueAtPath(path)
	if !ok {
		return assert.Fail(r.t, "field not found at path: "+path, msgAndArgs...)
	}

	count, ok := countElements(actual)
	if !ok {
		return assert.Fail(r.t, fmt.Sprintf("field at %s is not an array or object", path), msgAndArgs...)
	}

	if count != expected {
		return assert.Fail(r.t,
			fmt.Sprintf("field at %s has %d elements, should have %d", path, count, expected),
			msgAndArgs...)
	}

	return true
}

// Total asserts the count of top-level elements.
// For arrays: element count. For objects: key count.
func (r Document) Total(expected int, msgAndArgs ...any) bool {
	r.t.Helper()

	count, ok := r.countTopLevel()
	if !ok {
		return assert.Fail(r.t, "response body is not a JSON array or object", msgAndArgs...)
	}

	if count != expected {
		return assert.Fail(r.t,
			fmt.Sprintf("response has %d elements, should have %d", count, expected),
			msgAndArgs...)
	}

	return true
}

// JSON unmarshals the response body into the provided value.
func (r Document) JSON(v any) {
	r.t.Helper()
	assert.NoError(r.t, json.Unmarshal(r.body, v))
}

// Field returns the raw value at the given path.
// Returns nil if the path does not exist.
// Path uses dot notation: "user.name", "items.0.id".
func (r Document) Field(path string) any {
	val, _ := r.valueAtPath(path)

	return val
}

// valueAtPath navigates the JSON body using dot notation.
// Returns (nil, false) if the path does not exist.
func (r Document) valueAtPath(path string) (any, bool) {
	var data any
	if err := json.Unmarshal(r.body, &data); err != nil {
		return nil, false
	}

	if path == "" {
		return data, true
	}

	return navigatePath(data, strings.Split(path, "."))
}

// navigatePath walks a parsed JSON value using path segments.
func navigatePath(current any, parts []string) (any, bool) {
	for _, part := range parts {
		switch node := current.(type) {
		case map[string]any:
			var exists bool

			current, exists = node[part]
			if !exists {
				return nil, false
			}
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil, false
			}

			current = node[idx]
		default:
			return nil, false
		}
	}

	return current, true
}

// countTopLevel returns the element count of the top-level JSON value.
func (r Document) countTopLevel() (int, bool) {
	var data any
	if err := json.Unmarshal(r.body, &data); err != nil {
		return 0, false
	}

	return countElements(data)
}

// countElements returns the length of a JSON array or object.
func countElements(v any) (int, bool) {
	switch val := v.(type) {
	case []any:
		return len(val), true
	case map[string]any:
		return len(val), true
	default:
		return 0, false
	}
}

// jsonEqual compares two values, handling JSON number coercion.
// JSON unmarshals numbers as float64; this allows int comparison to work.
func jsonEqual(actual, expected any) bool {
	if actual == expected {
		return true
	}

	// JSON numbers are float64 — coerce integer types
	if af, ok := actual.(float64); ok {
		if ef, ok := toFloat64(expected); ok {
			return af == ef
		}
	}

	return reflect.DeepEqual(actual, expected)
}

// toFloat64 converts numeric types to float64 for JSON number comparison.
//
//nolint:gocyclo,cyclop
func toFloat64(val any) (float64, bool) {
	switch num := val.(type) {
	case int:
		return float64(num), true
	case int8:
		return float64(num), true
	case int16:
		return float64(num), true
	case int32:
		return float64(num), true
	case int64:
		return float64(num), true
	case uint:
		return float64(num), true
	case uint8:
		return float64(num), true
	case uint16:
		return float64(num), true
	case uint32:
		return float64(num), true
	case uint64:
		return float64(num), true
	case float32:
		return float64(num), true
	case float64:
		return num, true
	default:
		return 0, false
	}
}

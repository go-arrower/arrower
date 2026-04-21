//go:build e2e

package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/e2e"
)

func withJSON(body string) serverOption {
	return func(cfg *serverConfig) {
		cfg.headers["Content-Type"] = "application/json"
		cfg.html = body
	}
}

func TestDocument_IsArray(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body string
		pass bool
	}{
		"empty array":     {`[]`, true},
		"non-empty array": {`[1,2,3]`, true},
		"object":          {`{}`, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withJSON(tt.body))
			defer svr.Close()

			doc := e2e.Test(new(testing.T)).Get(svr.URL)
			pass := doc.IsArray()
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestDocument_IsObject(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body string
		pass bool
	}{
		"empty object":     {`{}`, true},
		"non-empty object": {`{"a":1}`, true},
		"array":            {`[]`, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withJSON(tt.body))
			defer svr.Close()

			doc := e2e.Test(new(testing.T)).Get(svr.URL)
			pass := doc.IsObject()
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestDocument_JSON(t *testing.T) {
	t.Parallel()

	t.Run("object into struct", func(t *testing.T) {
		t.Parallel()

		svr := server(withJSON(`{"name":"Alice","age":30}`))
		defer svr.Close()

		var result struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		e2e.Test(new(testing.T)).Get(svr.URL).JSON(&result)

		assert.Equal(t, "Alice", result.Name)
		assert.Equal(t, 30, result.Age)
	})

	t.Run("array into slice", func(t *testing.T) {
		t.Parallel()

		svr := server(withJSON(`[1,2,3]`))
		defer svr.Close()

		var result []int

		e2e.Test(new(testing.T)).Get(svr.URL).JSON(&result)

		assert.Equal(t, []int{1, 2, 3}, result)
	})
}

func TestDocument_Field(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body     string
		path     string
		expected any
	}{
		"top-level string": {`{"name":"Alice"}`, "name", "Alice"},
		"nested field":     {`{"user":{"name":"Alice"}}`, "user.name", "Alice"},
		"array index":      {`{"items":[10,20,30]}`, "items.0", float64(10)},
		"missing field":    {`{}`, "missing", nil},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withJSON(tt.body))
			defer svr.Close()

			doc := e2e.Test(new(testing.T)).Get(svr.URL)
			assert.Equal(t, tt.expected, doc.Field(tt.path))
		})
	}
}

func TestDocument_HasField(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body string
		path string
		pass bool
	}{
		"top-level exists":      {`{"name":"Alice"}`, "name", true},
		"top-level missing":     {`{"name":"Alice"}`, "age", false},
		"nested path exists":    {`{"user":{"name":"Alice"}}`, "user.name", true},
		"nested path missing":   {`{"user":{"name":"Alice"}}`, "user.email", false},
		"partial path exists":   {`{"user":{"name":"Alice"}}`, "user", true},
		"nested key not global": {`{"user":{"name":"Alice"}}`, "name", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withJSON(tt.body))
			defer svr.Close()

			doc := e2e.Test(new(testing.T)).Get(svr.URL)
			pass := doc.HasField(tt.path)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestDocument_NotHasField(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body string
		path string
		pass bool
	}{
		"top-level missing":   {`{"name":"Alice"}`, "age", true},
		"top-level exists":    {`{"name":"Alice"}`, "name", false},
		"nested path missing": {`{"user":{"name":"Alice"}}`, "user.email", true},
		"nested path exists":  {`{"user":{"name":"Alice"}}`, "user.name", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withJSON(tt.body))
			defer svr.Close()

			doc := e2e.Test(new(testing.T)).Get(svr.URL)
			pass := doc.NotHasField(tt.path)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestDocument_FieldEquals(t *testing.T) { //nolint:dupl
	t.Parallel()

	tests := map[string]struct {
		body     string
		path     string
		expected any
		pass     bool
	}{
		"string match":    {`{"name":"Alice"}`, "name", "Alice", true},
		"int match":       {`{"age":30}`, "age", 30, true},
		"nested match":    {`{"user":{"name":"Alice"}}`, "user.name", "Alice", true},
		"array element":   {`{"items":[10,20,30]}`, "items.0", 10, true},
		"string mismatch": {`{"name":"Alice"}`, "name", "Bob", false},
		"field missing":   {`{}`, "name", "Alice", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withJSON(tt.body))
			defer svr.Close()

			doc := e2e.Test(new(testing.T)).Get(svr.URL)
			pass := doc.FieldEquals(tt.path, tt.expected)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestDocument_FieldContains(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body string
		path string
		text string
		pass bool
	}{
		"contains":      {`{"bio":"Go developer"}`, "bio", "Go", true},
		"not contains":  {`{"bio":"Go developer"}`, "bio", "Rust", false},
		"nested":        {`{"user":{"bio":"Go dev"}}`, "user.bio", "Go", true},
		"field missing": {`{}`, "bio", "Go", false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withJSON(tt.body))
			defer svr.Close()

			doc := e2e.Test(new(testing.T)).Get(svr.URL)
			pass := doc.FieldContains(tt.path, tt.text)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestDocument_Total(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body     string
		expected int
		pass     bool
	}{
		"empty object":       {`{}`, 0, true},
		"object with keys":   {`{"a":1,"b":2}`, 2, true},
		"wrong object count": {`{"a":1}`, 2, false},
		"empty array":        {`[]`, 0, true},
		"array with items":   {`[1,2,3]`, 3, true},
		"wrong array count":  {`[1,2]`, 3, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withJSON(tt.body))
			defer svr.Close()

			doc := e2e.Test(new(testing.T)).Get(svr.URL)
			pass := doc.Total(tt.expected)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

func TestDocument_FieldLen(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body     string
		path     string
		expected int
		pass     bool
	}{
		"root object":             {`{"items":[]}`, "", 1, true},
		"nested array length":     {`{"items":[1,2,3]}`, "items", 3, true},
		"empty nested array":      {`{"items":[]}`, "items", 0, true},
		"wrong array length":      {`{"items":[1]}`, "items", 2, false},
		"nested object key count": {`{"meta":{"a":1,"b":2}}`, "meta", 2, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svr := server(withJSON(tt.body))
			defer svr.Close()

			doc := e2e.Test(new(testing.T)).Get(svr.URL)
			pass := doc.FieldLen(tt.path, tt.expected)
			assert.Equal(t, tt.pass, pass)
		})
	}
}

package aassert_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/aassert"
)

func TestNumFields(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		s    any
		n    int
		pass bool
	}{
		"nil":             {nil, 0, false},
		"bool":            {false, 0, false},
		"int":             {0, 0, false},
		"string":          {"", 0, false},
		"slice":           {[]int{}, 0, false},
		"map":             {[]map[int]int{}, 0, false},
		"simple struct":   {StructA{}, 1, true},
		"simple miscount": {StructA{}, 1337, false},
		"ptr to struct":   {&StructA{}, 1, true},
		"with slice":      {slice{}, 3, true},
		"with map":        {dictionary{}, 3, true},
		"complex struct":  {complexStruct{}, 13, true},
		"complex nested":  {complexNested{}, 8, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pass := aassert.NumFields(new(testing.T), tt.n, tt.s)
			// pass = aassert.NumFields(t, tt.s, tt.n) // uncomment to see t.Log() output from the assertion function.
			assert.Equal(t, tt.pass, pass)
		})
	}
}

type (
	ID            int
	complexNested struct {
		StructC
	}
	complexStruct struct {
		ID      ID
		Str     string
		private int //nolint:unused
		StructA
		structB
		A structB
		B []StructA
		C map[int]StructA
	}
	dictionary struct {
		Str string
		As  map[int]StructA
	}
	slice struct {
		Str string
		As  []StructA
	}
	StructC struct {
		A []StructA
		B map[int]StructA
		C *StructA
		D int
	}
	structB struct {
		Str    string
		IntPtr *int
		StructA
	}
	StructA struct {
		Str     string
		private int //nolint:unused
	}
)

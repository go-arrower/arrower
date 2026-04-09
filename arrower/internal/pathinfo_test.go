package internal_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/internal"
)

func TestNewPathInfo(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input        string
		expectedRel  string
		expectedDisp string
		expectError  bool
	}{
		"current directory": {
			input:        ".",
			expectedRel:  "",
			expectedDisp: ".",
			expectError:  false,
		},
		"parent directory": {
			input:        "..",
			expectedRel:  "..",
			expectedDisp: "..",
			expectError:  false,
		},
		"relative path": {
			input:        "../arrower",
			expectedRel:  "../arrower",
			expectedDisp: "../arrower",
			expectError:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pi, err := internal.NewPathInfo(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRel, pi.RelativePath)
			assert.Equal(t, tt.expectedDisp, pi.DisplayPath)
			// AbsPath will be system-dependent, just check it's not empty
			assert.NotEmpty(t, pi.AbsPath)
		})
	}
}

func TestNewPathInfos(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		paths       []string
		expectError bool
		expectedLen int
	}{
		"single path": {
			paths:       []string{"."},
			expectError: false,
			expectedLen: 1,
		},
		"multiple paths": {
			paths:       []string{".", "../arrower"},
			expectError: false,
			expectedLen: 2,
		},
		"empty slice": {
			paths:       []string{},
			expectError: false,
			expectedLen: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pis, err := internal.NewPathInfos(tt.paths)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, pis, tt.expectedLen)
		})
	}
}

func TestPathInfos_FormatFilePath(t *testing.T) {
	t.Parallel()

	paths := internal.PathInfos{
		{AbsPath: "/home/user/project", RelativePath: "", DisplayPath: "."},
		{AbsPath: "/home/user/arrower", RelativePath: "../arrower", DisplayPath: "../arrower"},
	}

	tests := map[string]struct {
		input    string
		expected string
	}{
		"file in current directory": {
			input:    "/home/user/project/main.go",
			expected: "main.go",
		},
		"file in arrower directory": {
			input:    "/home/user/arrower/contexts/auth/app.go",
			expected: "../arrower/contexts/auth/app.go",
		},
		"file in subdirectory": {
			input:    "/home/user/project/internal/app.go",
			expected: "internal/app.go",
		},
		"file not in watched paths": {
			input:    "/other/path/file.go",
			expected: "/other/path/file.go",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := paths.FormatFilePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

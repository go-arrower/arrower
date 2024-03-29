//go:build integration

package generate_test

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/internal/generate"
)

var update = flag.Bool("update", false, "update golden files")

func TestParseArgs(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args    []string
		parsed  []string
		appType generate.CodeType
		err     error
	}{
		"nil": {
			nil,
			nil,
			generate.Unknown,
			generate.ErrInvalidArguments,
		},
		"empty": {
			[]string{},
			nil,
			generate.Unknown,
			generate.ErrInvalidArguments,
		},
		"too many args": {
			[]string{"too", "many"},
			nil,
			generate.Unknown,
			generate.ErrInvalidArguments,
		},
		"dash case": {
			[]string{" say-Hello"},
			[]string{"say", "hello"},
			generate.Unknown,
			nil,
		},
		"camel case": {
			[]string{"sayHello "},
			[]string{"say", "hello"},
			generate.Unknown,
			nil,
		},
	}

	for name, tt := range tests {
		tt := tt

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			args, appType, err := generate.ParseArgs(tt.args)
			assert.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.parsed, args)
			assert.Equal(t, tt.appType, appType)
		})
	}
}

//nolint:goconst // use the testdata folder without const
func TestGenerate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args     []string
		cType    generate.CodeType
		expFiles []string
		expErr   error
	}{
		// "usecase": {
		// 	[]string{"helloWorld"},
		//	generate.Unknown,
		//	[]string{"hello-world.usecase.go", "hello-world.usecase_test.go"},
		//	nil,
		// },
		"request": {
			[]string{"helloWorld"},
			generate.Request,
			[]string{"hello-world.request.go", "hello-world.request_test.go"},
			nil,
		},
		"command": {
			[]string{"say-hello"},
			generate.Command,
			[]string{"say-hello.command.go", "say-hello.command_test.go"},
			nil,
		},
		"query": {
			[]string{"getHelloWorld"},
			generate.Query,
			[]string{"get-hello-world.query.go", "get-hello-world.query_test.go"},
			nil,
		},
		"job": {
			[]string{"greet"},
			generate.Job,
			[]string{"greet.job.go", "greet.job_test.go"},
			nil,
		},
	}

	for name, tt := range tests {
		tt := tt

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			files, err := generate.Generate(dir, tt.args, tt.cType)
			assert.NoError(t, err)

			// path & file names as expected
			assert.Equal(t, tt.expFiles[0], files[0])
			assert.Equal(t, tt.expFiles[1], files[1])

			// files got created
			assert.FileExists(t, dir+"/"+tt.expFiles[0])
			assert.FileExists(t, dir+"/"+tt.expFiles[1])

			if *update {
				input, err := os.ReadFile(dir + "/" + tt.expFiles[0])
				assert.NoError(t, err)
				err = os.WriteFile("testdata/"+tt.expFiles[0], input, 0o644) //nolint:gosec // same permissions as default desktop behaviour
				assert.NoError(t, err)

				input, err = os.ReadFile(dir + "/" + tt.expFiles[1])
				assert.NoError(t, err)
				err = os.WriteFile("testdata/"+tt.expFiles[1], input, 0o644) //nolint:gosec // same permissions as default desktop behaviour
				assert.NoError(t, err)
			}

			{ // content is as expected
				goldenCode := "testdata/" + tt.expFiles[0]
				goldenTest := "testdata/" + tt.expFiles[1]

				expected, err := os.ReadFile(goldenCode)
				assert.NoError(t, err)
				actual, err := os.ReadFile(dir + "/" + files[0])
				assert.NoError(t, err)
				assert.Equal(t, string(expected), string(actual))

				expected, err = os.ReadFile(goldenTest)
				assert.NoError(t, err)
				actual, err = os.ReadFile(dir + "/" + files[1])
				assert.NoError(t, err)
				assert.Equal(t, string(expected), string(actual))
			}
		})
	}
}

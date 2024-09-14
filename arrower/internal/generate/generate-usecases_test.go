//go:build integration

package generate_test

import (
	"flag"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arrower/internal/generate"
)

var update = flag.Bool("update", false, "update golden files")

func TestParseArgs(t *testing.T) {
	t.Parallel()

	dir := newTempProject(t)

	tests := map[string]struct {
		rawArgs    []string
		parsedArgs generate.ParsedArgs
		err        error
	}{
		"nil": {
			nil,
			generate.ParsedArgs{
				Context:  "",
				Args:     nil,
				CodeType: generate.Unknown,
			},
			generate.ErrInvalidArguments,
		},
		"empty": {
			[]string{},
			generate.ParsedArgs{
				Context:  "",
				Args:     nil,
				CodeType: generate.Unknown,
			},
			generate.ErrInvalidArguments,
		},
		"too many args": {
			[]string{"too", "many", "args"},
			generate.ParsedArgs{
				Context:  "",
				Args:     nil,
				CodeType: generate.Unknown,
			},
			generate.ErrInvalidArguments,
		},
		"dash case": {
			[]string{" say-Hello"},
			generate.ParsedArgs{
				Context:  "",
				Args:     []string{"say", "hello"},
				CodeType: generate.Usecase,
			},
			nil,
		},
		"camel case": {
			[]string{"sayHello "},
			generate.ParsedArgs{
				Context:  "",
				Args:     []string{"say", "hello"},
				CodeType: generate.Usecase,
			},
			nil,
		},
		"existing context": {
			[]string{"admin ", "sayHello "},
			generate.ParsedArgs{
				Context:  "admin",
				Args:     []string{"say", "hello"},
				CodeType: generate.Usecase,
			},
			nil,
		},
		"unknown context": {
			[]string{"non-existing", "sayHello "},
			generate.ParsedArgs{
				Context:  "",
				Args:     nil,
				CodeType: generate.Unknown,
			},
			generate.ErrInvalidArguments,
		},
		"detect request": {
			[]string{"sayHelloRequest "},
			generate.ParsedArgs{
				Context:  "",
				Args:     []string{"say", "hello"},
				CodeType: generate.Request,
			},
			nil,
		},
		"detect command": {
			[]string{"sayHelloCommand"},
			generate.ParsedArgs{
				Context:  "",
				Args:     []string{"say", "hello"},
				CodeType: generate.Command,
			},
			nil,
		},
		"detect query": {
			[]string{"say-hello-query"},
			generate.ParsedArgs{
				Context:  "",
				Args:     []string{"say", "hello"},
				CodeType: generate.Query,
			},
			nil,
		},
		"detect job": {
			[]string{" sayHelloJob"},
			generate.ParsedArgs{
				Context:  "",
				Args:     []string{"say", "hello"},
				CodeType: generate.Job,
			},
			nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			args, err := generate.ParseArgs(dir, tt.rawArgs)
			assert.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.parsedArgs, args)
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
		"usecase": {
			[]string{"helloArrower"},
			generate.Unknown,
			[]string{"hello-arrower.usecase.go", "hello-arrower.usecase_test.go"},
			nil,
		},
		"request": {
			[]string{"helloWorld"},
			generate.Request,
			[]string{"hello-world.usecase.go", "hello-world.usecase_test.go"},
			nil,
		},
		"command": {
			[]string{"say-hello"},
			generate.Command,
			[]string{"say-hello.usecase.go", "say-hello.usecase_test.go"},
			nil,
		},
		"query": {
			[]string{"getHelloWorld"},
			generate.Query,
			[]string{"get-hello-world.usecase.go", "get-hello-world.usecase_test.go"},
			nil,
		},
		"job": {
			[]string{"greet"},
			generate.Job,
			[]string{"greet.usecase.go", "greet.usecase_test.go"},
			nil,
		},
		"detect query": {
			[]string{"getSomethingQuery"},
			0,
			[]string{"get-something.usecase.go", "get-something.usecase_test.go"},
			nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := newTempProject(t)

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

	t.Run("detect proper folder", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			folderPath string
			err        error
		}{
			"no application folder found": {
				"",
				nil,
			},
			"application in root": {
				"application",
				nil,
			},
			"arrower shared": {
				"shared/application/",
				nil,
			},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				dir := newTempProject(t)
				err := os.MkdirAll(path.Join(dir, tt.folderPath), os.ModePerm)
				assert.NoError(t, err)

				files, err := generate.Generate(dir, []string{"example"}, generate.Request)
				assert.ErrorIs(t, err, tt.err)

				// files got created
				assert.FileExists(t, path.Join(dir, files[0]))
				assert.Equal(t, path.Join(tt.folderPath, "example.usecase.go"), files[0])
				assert.FileExists(t, path.Join(dir, files[1]))
				assert.Equal(t, path.Join(tt.folderPath, "example.usecase_test.go"), files[1])
			})
		}
	})

	t.Run("detect context folders", func(t *testing.T) {
		t.Parallel()

		dir := newTempProject(t)

		files, err := generate.Generate(dir, []string{"admin", "example"}, generate.Request)
		assert.NoError(t, err)

		// files got created
		assert.FileExists(t, path.Join(dir, files[0]))
		assert.Equal(t, "contexts/admin/internal/application/example.usecase.go", files[0])
		assert.FileExists(t, path.Join(dir, files[1]))
		assert.Equal(t, "contexts/admin/internal/application/example.usecase_test.go", files[1])
	})
}

func newTempProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	err := os.WriteFile(dir+"/go.mod", []byte(`module example/app`), 0o600)
	assert.NoError(t, err)

	err = os.MkdirAll(path.Join(dir, "contexts/admin/internal/application"), os.ModePerm)
	assert.NoError(t, err)

	return dir
}

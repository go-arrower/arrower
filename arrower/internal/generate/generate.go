package generate

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"os"
	"regexp"
	"strings"

	"github.com/fatih/camelcase"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var ErrInvalidArguments = errors.New("invalid arguments")

// CodeType indicates which kind of code should be generated, e.g. usecase or controller.
//
//go:generate stringer -type=CodeType
type CodeType int

const (
	Unknown CodeType = iota
	Usecase
	Request
	Command
	Query
	Job
)

func ParseArgs(args []string) ([]string, CodeType, error) {
	if len(args) != 1 {
		return nil, Unknown, ErrInvalidArguments
	}

	camelCaseRE := regexp.MustCompile(`^[a-z]+(?:[A-Z][a-z]+)*$`)
	arg := strings.TrimSpace(args[0])

	var parsed []string

	if camelCaseRE.MatchString(arg) { //nolint:gocritic // don't want to rewrite to switch/case
		parsed = camelcase.Split(arg)
	} else if strings.Contains(arg, "-") {
		parsed = strings.Split(arg, "-")
	} else {
		return nil, Unknown, fmt.Errorf("%w: could not detect kebab-case or camelCase", ErrInvalidArguments)
	}

	for i, p := range parsed {
		parsed[i] = strings.ToLower(strings.TrimSpace(p))
	}

	return parsed, Unknown, nil
}

func Generate(calledFromPath string, args []string, cType CodeType) ([]string, error) {
	arg, parsedType, err := ParseArgs(args)
	if err != nil {
		return nil, fmt.Errorf("could not parse args: %w", err)
	}

	if cType == Unknown { // if no type is set, use the parsed one
		cType = parsedType
	}

	fileContent, err := renderFiles(arg, cType)
	if err != nil {
		return nil, fmt.Errorf("could not render usecase: %w", err)
	}

	codeFile := strings.Join(arg, "-") + "." + strings.ToLower(cType.String()) + ".go"
	testFile := strings.Join(arg, "-") + "." + strings.ToLower(cType.String()) + "_test.go"

	err = saveFiles(map[string][]byte{
		calledFromPath + "/" + codeFile: fileContent[0],
		calledFromPath + "/" + testFile: fileContent[1],
	})
	if err != nil {
		return nil, fmt.Errorf("could not save usecase: %w", err)
	}

	return []string{codeFile, testFile}, nil
}

type renderData struct {
	CodeTemplate string
	TestTemplate string

	ParamName  string
	ParamType  string
	ReturnType string

	PkgName         string
	ConstructorName string
	HandlerName     string
	Type            CodeType
}

//nolint:funlen // long but straight forward to read
func renderFiles(arg []string, cType CodeType) ([][]byte, error) {
	data := renderData{ //nolint:exhaustruct // not shared fields are set below
		PkgName: "application",
		Type:    cType,
	}
	switch data.Type {
	case Command:
		data.CodeTemplate = commandTemplate
		data.TestTemplate = commandTestTemplate
		data.ConstructorName = camelName(arg) + data.Type.String()
		data.HandlerName = arg[0] + camelName(arg[1:]) + data.Type.String()
		data.ParamName = "cmd"
		data.ParamType = camelName(arg) + data.Type.String()
	case Job:
		data.CodeTemplate = commandTemplate
		data.TestTemplate = commandTestTemplate
		data.ConstructorName = camelName(arg) + data.Type.String()
		data.HandlerName = arg[0] + camelName(arg[1:]) + data.Type.String()
		data.ParamName = "job"
		data.ParamType = camelName(arg) + data.Type.String()
	case Query:
		data.CodeTemplate = requestTemplate
		data.TestTemplate = requestTestTemplate
		data.ConstructorName = camelName(arg) + data.Type.String()
		data.HandlerName = arg[0] + camelName(arg[1:]) + data.Type.String()
		data.ParamName = "query"
		data.ParamType = camelName(arg) + data.Type.String()
		data.ReturnType = camelName(arg) + "Response"
	default: // Request
		data.CodeTemplate = requestTemplate
		data.TestTemplate = requestTestTemplate
		data.ConstructorName = camelName(arg) + Request.String()
		data.HandlerName = arg[0] + camelName(arg[1:]) + Request.String()
		data.ParamName = "req"
		data.ParamType = camelName(arg) + Request.String()
		data.ReturnType = camelName(arg) + "Response"
	}

	templates := [][]byte{}

	codeBuf := bytes.Buffer{}
	testBuf := bytes.Buffer{}

	code, err := template.New("").Parse(data.CodeTemplate)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	err = code.Execute(&codeBuf, data)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	code, err = template.New("").Parse(data.TestTemplate)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	err = code.Execute(&testBuf, data)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	templates = append(templates, codeBuf.Bytes(), testBuf.Bytes())

	return templates, nil
}

func camelName(arg []string) string {
	name := ""

	for _, n := range arg {
		name += cases.Title(language.Und).String(n)
	}

	return name
}

func saveFiles(templates map[string][]byte) error {
	for path, templ := range templates {
		err := os.WriteFile(path, templ, 0o644) //nolint:gosec,gomnd // same permissions as default desktop behaviour
		if err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	return nil
}

//nolint:lll
const requestTemplate = `package {{ .PkgName }}

import (
	"context"

	"github.com/go-arrower/arrower/app"
)

func New{{- .ConstructorName -}}Handler() app.{{- .Type -}}[{{- .ParamType -}}, {{ .ReturnType -}}] {
	return &{{- .HandlerName -}}Handler{}
}

type {{ .HandlerName -}}Handler struct{}

type (
	{{ .ParamType }} struct{}
	{{ .ReturnType }} struct{}
)

func (h *{{- .HandlerName -}}Handler) H(ctx context.Context, {{ .ParamName }} {{ .ConstructorName -}}) ({{- .ReturnType -}}, error) {
	return {{ .ReturnType -}}{}, nil
}
`

const requestTestTemplate = `package {{ .PkgName }}_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	application "github.com/go-arrower/arrower/arrower/internal/generate/testdata"
)

func Test{{- .ConstructorName -}}Handler_H(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		handler := application.New{{- .ConstructorName -}}Handler()

		res, err := handler.H(context.Background(), application.{{- .ParamType -}}{})
		assert.NoError(t, err)
		assert.Empty(t, res)
	})
}
`

const commandTemplate = `package {{ .PkgName }}

import (
	"context"

	"github.com/go-arrower/arrower/app"
)

func New{{- .ConstructorName -}}Handler() app.{{- .Type -}}[{{- .ParamType -}}] {
	return &{{- .HandlerName -}}Handler{}
}

type {{ .HandlerName -}}Handler struct{}

type (
	{{ .ParamType }} struct{}
)

func (h *{{- .HandlerName -}}Handler) H(ctx context.Context, {{ .ParamName }} {{ .ConstructorName -}}) error {
	return nil
}
`

const commandTestTemplate = `package {{ .PkgName }}_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	application "github.com/go-arrower/arrower/arrower/internal/generate/testdata"
)

func Test{{- .ConstructorName -}}Handler_H(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		handler := application.New{{- .ConstructorName -}}Handler()

		err := handler.H(context.Background(), application.{{- .ParamType -}}{})
		assert.NoError(t, err)
	})
}
`

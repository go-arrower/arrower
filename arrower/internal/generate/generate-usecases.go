package generate

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/fatih/camelcase"
	"golang.org/x/mod/modfile"
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

type ParsedArgs struct {
	Context  string
	Args     []string
	CodeType CodeType
}

// ParseArgs takes the raw input from the developer and understands what usecase
// to generate.
func ParseArgs(calledFromPath string, args []string) (ParsedArgs, error) {
	argsLength := len(args)

	const numArgsForContextCall = 2
	if argsLength == 0 || argsLength > numArgsForContextCall {
		return ParsedArgs{}, ErrInvalidArguments
	}

	context := ""
	hasContext := argsLength == numArgsForContextCall

	if hasContext {
		context = strings.TrimSpace(args[0])

		_, err := os.Stat(path.Join(calledFromPath, "contexts", context, "internal/application"))
		if os.IsNotExist(err) { // folder exists
			return ParsedArgs{}, fmt.Errorf("%w: context does not exist", ErrInvalidArguments)
		}
	}

	camelCaseRE := regexp.MustCompile(`^[a-z]+(?:[A-Z][a-z]+)*$`)
	arg := strings.TrimSpace(args[argsLength-1])

	var parsed []string

	if camelCaseRE.MatchString(arg) { //nolint:gocritic // don't want to rewrite to switch/case
		parsed = camelcase.Split(arg)
	} else if strings.Contains(arg, "-") {
		parsed = strings.Split(arg, "-")
	} else {
		return ParsedArgs{}, fmt.Errorf("%w: could not parse name, use kebab-case or camelCase", ErrInvalidArguments)
	}

	for i, p := range parsed {
		parsed[i] = strings.ToLower(strings.TrimSpace(p))
	}

	parsed, cType := detectCodeType(parsed)

	return ParsedArgs{Args: parsed, Context: context, CodeType: cType}, nil
}

// detectCodeType extracts the intended type from the developer's input.
// sayHelloRequest is understood as a Request with the name sayHello.
func detectCodeType(parsed []string) ([]string, CodeType) {
	checkIfCodeType := parsed[len(parsed)-1]

	var cType CodeType

	switch checkIfCodeType {
	case "request":
		cType = Request
	case "command":
		cType = Command
	case "query":
		cType = Query
	case "job":
		cType = Job
	default:
		cType = Usecase
	}

	if cType != Unknown && cType != Usecase { // type found => don't return it
		return parsed[:len(parsed)-1], cType
	}

	return parsed, cType
}

func Generate(calledFromPath string, args []string, cType CodeType) ([]string, error) {
	pargs, err := ParseArgs(calledFromPath, args)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if cType == Unknown { // if no type is set, use the parsed one
		cType = pargs.CodeType
	}

	pkgPath, err := pkgPath(calledFromPath, pargs.Context)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	fileContent, err := renderFiles(pargs.Args, cType, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("could not render usecase: %w", err)
	}

	codeFile := strings.Join(pargs.Args, "-") + ".usecase.go"
	testFile := strings.Join(pargs.Args, "-") + ".usecase_test.go"

	applicationPath := detectApplicationPath(calledFromPath, pargs.Context)

	err = saveFiles(map[string][]byte{
		path.Join(applicationPath, codeFile): fileContent[0],
		path.Join(applicationPath, testFile): fileContent[1],
	})
	if err != nil {
		return nil, fmt.Errorf("could not save usecase: %w", err)
	}

	return []string{
		strings.TrimPrefix(path.Join(applicationPath, codeFile), calledFromPath+"/"),
		strings.TrimPrefix(path.Join(applicationPath, testFile), calledFromPath+"/"),
	}, nil
}

func detectApplicationPath(dir string, context string) string {
	searchDirs := []string{
		path.Join(dir, "/", "contexts", context, "internal/application"),
		path.Join(dir, "/", "shared/application"),
		path.Join(dir, "/", "application"),
	}

	for _, searchDir := range searchDirs {
		_, err := os.Stat(searchDir)
		if !os.IsNotExist(err) { // folder exists
			return searchDir
		}
	}

	return dir
}

func pkgPath(calledFromPath string, context string) (string, error) {
	b, err := os.ReadFile(path.Join(calledFromPath, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("could not read go.mod file: %w", err)
	}

	file, err := modfile.Parse("go.mod", b, nil)
	if err != nil {
		return "", fmt.Errorf("could not parse go.mod file: %w", err)
	}

	appPath := strings.TrimPrefix(path.Join(detectApplicationPath(calledFromPath, context)), path.Join(calledFromPath))

	return path.Join(file.Module.Mod.String(), appPath), nil
}

type renderData struct {
	CodeTemplate string
	TestTemplate string

	ParamName  string
	ParamType  string
	ReturnType string

	PkgPath         string
	PkgName         string
	Usecase         string // the name of the use case without any postfixes
	ConstructorName string // returns a struct of type HandlerName
	HandlerName     string // the struct that implements this usecase
	ErrMsg          string
	Type            CodeType // the usecase type
}

//nolint:funlen // long but straight forward to read
func renderFiles(arg []string, cType CodeType, pkgPath string) ([][]byte, error) {
	data := renderData{ //nolint:exhaustruct // not shared fields are set below
		PkgPath: pkgPath,
		PkgName: "application",
		Usecase: camelName(arg),
		Type:    cType,
		ErrMsg:  strings.Join(arg, " "),
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
	default: // Request, Usecase
		data.Type = Request
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

		err = exec.Command("go", "fmt", path).Run()
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
	"errors"

	"github.com/go-arrower/arrower/app"
)

var Err{{- .Usecase -}}Failed = errors.New("{{ .ErrMsg }} failed")

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

	"{{- .PkgPath -}}"
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
	"errors"

	"github.com/go-arrower/arrower/app"
)

var Err{{- .Usecase -}}Failed = errors.New("{{ .ErrMsg }} failed")

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

	"{{- .PkgPath -}}"
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

package hooks

import (
	"errors"
	"fmt"
	"go/build"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"github.com/traefik/yaegi/stdlib/unrestricted"
)

var ErrLoadHooksFailed = errors.New("loading hooks failed")

//nolint:gochecknoglobals // to allow the Register method to accept new hooks it is a global store.
var loadedHooks Hooks

// Load loads all hooks in the given directory.
func Load(dir string) (Hooks, error) {
	var interpreter *interp.Interpreter
	{
		interpreter = interp.New(interp.Options{GoPath: build.Default.GOPATH})

		err := interpreter.Use(stdlib.Symbols)
		if err != nil {
			return nil, fmt.Errorf("%w: could not load interpreter: %v", ErrLoadHooksFailed, err)
		}

		err = interpreter.Use(unrestricted.Symbols)
		if err != nil {
			return nil, fmt.Errorf("%w: could not load interpreter: %v", ErrLoadHooksFailed, err)
		}

		err = interpreter.Use(map[string]map[string]reflect.Value{
			"github.com/go-arrower/arrower/arrower/hooks/hooks": {
				"Register":  reflect.ValueOf(Register),
				"Hook":      reflect.ValueOf((*Hook)(nil)),
				"RunConfig": reflect.ValueOf((*RunConfig)(nil)),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("%w: could not load interpreter: %v", ErrLoadHooksFailed, err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("%w: could not read directory: %s: %v", ErrLoadHooksFailed, dir, err)
	}

	for _, e := range entries {
		hookFileName := e.Name()
		if !strings.HasSuffix(hookFileName, ".hook.go") {
			continue
		}

		_, err = interpreter.EvalPath(path.Join(dir, hookFileName))
		if err != nil {
			return nil, fmt.Errorf("%w: could not evaluate hook: %s: %v", ErrLoadHooksFailed, hookFileName, err)
		}
	}

	return loadedHooks, nil
}

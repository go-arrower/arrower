// Package app does.
package app

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

type Request[Req any, Res any] interface {
	H(ctx context.Context, req Req) (Res, error)
}

type Command[C any] interface {
	H(ctx context.Context, cmd C) error
}

type Query[Q any, Res any] interface {
	H(ctx context.Context, query Q) (Res, error)
}

type Job[J any] interface {
	H(ctx context.Context, job J) error
}

// commandName extracts a printable name from cmd in the format of: functionName.
//
// structName	 								=> strings.Split(fmt.Sprintf("%T", cmd), ".")[1]
// structname	 								=> strings.ToLower(strings.Split(fmt.Sprintf("%T", cmd), ".")[1])
// packageName.structName	 					=> fmt.Sprintf("%T", cmd)
// github.com/go-arrower/skeleton/.../package	=> fmt.Sprintln(reflect.TypeOf(cmd).PkgPath())
// structName is used, the other examples are for inspiration.
// The use case function can not be used, as it is anonymous / a closure returned by the use case constructor.
// Accessing the function name with runtime.Caller(4) will always lead to ".func1".
func commandName(cmd any) string {
	pkgPath := reflect.TypeOf(cmd).PkgPath()

	// example: github.com/go-arrower/skeleton/contexts/admin/internal/application_test
	// take string after /contexts/ and then take string before /internal/
	pkg0 := strings.Split(pkgPath, "/contexts/")

	hasContext := len(pkg0) == 2 //nolint:gomnd
	if hasContext {
		pkg1 := strings.Split(pkg0[1], "/internal/")
		if len(pkg1) == 2 { //nolint:gomnd
			context := pkg1[0]

			return fmt.Sprintf("%s.%T", context, cmd)
		}
	}

	// fallback: if the function is not called from a proper Context => packageName.structName
	return fmt.Sprintf("%T", cmd)
}

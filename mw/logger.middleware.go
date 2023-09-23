package mw

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
)

type DecoratorFunc[in, out any] interface {
	func(context.Context, in) (out, error)
}

type DecoratorFuncUnary[in any] interface {
	func(context.Context, in) error
}

// Logged wraps an application function / command with debug logs.
func Logged[in, out any, F DecoratorFunc[in, out]](logger *slog.Logger, next F) F { //nolint:ireturn,lll // valid use of generics
	return func(ctx context.Context, in in) (out, error) {
		cmdName := commandName(in)

		logger.DebugContext(ctx, "executing command",
			slog.String("command", cmdName),
		)

		result, err := next(ctx, in)

		if err == nil {
			logger.DebugContext(ctx, "command executed successfully",
				slog.String("command", cmdName))
		} else {
			logger.DebugContext(ctx, "failed to execute command",
				slog.String("command", cmdName),
				slog.String("error", err.Error()),
			)
		}

		return result, err
	}
}

// LoggedU is like Logged but for functions only returning errors, e.g. jobs.
func LoggedU[in any, F DecoratorFuncUnary[in]](logger *slog.Logger, next F) F { //nolint:ireturn,lll // valid use of generics
	return func(ctx context.Context, in in) error {
		cmdName := commandName(in)

		logger.DebugContext(ctx, "executing command",
			slog.String("command", cmdName),
		)

		err := next(ctx, in)

		if err == nil {
			logger.DebugContext(ctx, "command executed successfully",
				slog.String("command", cmdName))
		} else {
			logger.DebugContext(ctx, "failed to execute command",
				slog.String("command", cmdName),
				slog.String("error", err.Error()),
			)
		}

		return err
	}
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

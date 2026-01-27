package views

import (
	"context"
)

func DefaultBaseDataFunc() func(_ context.Context) (map[string]any, error) {
	return func(_ context.Context) (map[string]any, error) {
		return map[string]any{
			"Title": "{{ .Name }}",
		}, nil
	}
}

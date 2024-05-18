package application

import (
	"context"
	"errors"

	"github.com/go-arrower/arrower/app"
)

var ErrSomeTestFailed = errors.New("some test failed")

func NewSomeTestRequestHandler() app.Request[SomeTestRequest, SomeTestResponse] {
	return &someTestRequestHandler{}
}

type someTestRequestHandler struct{}

type (
	SomeTestRequest  struct{}
	SomeTestResponse struct{}
)

func (h *someTestRequestHandler) H(ctx context.Context, req SomeTestRequest) (SomeTestResponse, error) {
	return SomeTestResponse{}, nil
}

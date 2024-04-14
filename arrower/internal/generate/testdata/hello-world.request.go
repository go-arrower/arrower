package application

import (
	"context"
	"errors"

	"github.com/go-arrower/arrower/app"
)

var ErrHelloWorldFailed = errors.New("hello world failed")

func NewHelloWorldRequestHandler() app.Request[HelloWorldRequest, HelloWorldResponse] {
	return &helloWorldRequestHandler{}
}

type helloWorldRequestHandler struct{}

type (
	HelloWorldRequest  struct{}
	HelloWorldResponse struct{}
)

func (h *helloWorldRequestHandler) H(ctx context.Context, req HelloWorldRequest) (HelloWorldResponse, error) {
	return HelloWorldResponse{}, nil
}

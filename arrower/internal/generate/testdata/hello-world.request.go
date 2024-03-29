package application

import (
	"context"

	"github.com/go-arrower/arrower/app"
)

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

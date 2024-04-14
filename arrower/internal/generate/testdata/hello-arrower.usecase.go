package application

import (
	"context"
	"errors"

	"github.com/go-arrower/arrower/app"
)

var ErrHelloArrowerFailed = errors.New("hello arrower failed")

func NewHelloArrowerRequestHandler() app.Request[HelloArrowerRequest, HelloArrowerResponse] {
	return &helloArrowerRequestHandler{}
}

type helloArrowerRequestHandler struct{}

type (
	HelloArrowerRequest  struct{}
	HelloArrowerResponse struct{}
)

func (h *helloArrowerRequestHandler) H(ctx context.Context, req HelloArrowerRequest) (HelloArrowerResponse, error) {
	return HelloArrowerResponse{}, nil
}

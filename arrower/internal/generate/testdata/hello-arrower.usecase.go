package application

import (
	"context"

	"github.com/go-arrower/arrower/app"
)

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

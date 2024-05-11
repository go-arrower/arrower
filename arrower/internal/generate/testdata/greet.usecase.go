package application

import (
	"context"
	"errors"

	"github.com/go-arrower/arrower/app"
)

var ErrGreetFailed = errors.New("greet failed")

func NewGreetJobHandler() app.Job[GreetJob] {
	return &greetJobHandler{}
}

type greetJobHandler struct{}

type (
	GreetJob struct{}
)

func (h *greetJobHandler) H(ctx context.Context, job GreetJob) error {
	return nil
}

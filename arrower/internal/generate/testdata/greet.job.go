package application

import (
	"context"

	"github.com/go-arrower/arrower/app"
)

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

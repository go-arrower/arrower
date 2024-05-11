package application

import (
	"context"
	"errors"

	"github.com/go-arrower/arrower/app"
)

var ErrSayHelloFailed = errors.New("say hello failed")

func NewSayHelloCommandHandler() app.Command[SayHelloCommand] {
	return &sayHelloCommandHandler{}
}

type sayHelloCommandHandler struct{}

type (
	SayHelloCommand struct{}
)

func (h *sayHelloCommandHandler) H(ctx context.Context, cmd SayHelloCommand) error {
	return nil
}

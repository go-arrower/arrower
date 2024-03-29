package application

import (
	"context"

	"github.com/go-arrower/arrower/app"
)

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

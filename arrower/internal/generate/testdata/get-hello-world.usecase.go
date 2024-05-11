package application

import (
	"context"
	"errors"

	"github.com/go-arrower/arrower/app"
)

var ErrGetHelloWorldFailed = errors.New("get hello world failed")

func NewGetHelloWorldQueryHandler() app.Query[GetHelloWorldQuery, GetHelloWorldResponse] {
	return &getHelloWorldQueryHandler{}
}

type getHelloWorldQueryHandler struct{}

type (
	GetHelloWorldQuery    struct{}
	GetHelloWorldResponse struct{}
)

func (h *getHelloWorldQueryHandler) H(ctx context.Context, query GetHelloWorldQuery) (GetHelloWorldResponse, error) {
	return GetHelloWorldResponse{}, nil
}

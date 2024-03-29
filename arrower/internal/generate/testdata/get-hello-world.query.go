package application

import (
	"context"

	"github.com/go-arrower/arrower/app"
)

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

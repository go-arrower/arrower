package application

import (
	"context"

	"github.com/go-arrower/arrower/app"
)

func NewGetSomethingQueryHandler() app.Query[GetSomethingQuery, GetSomethingResponse] {
	return &getSomethingQueryHandler{}
}

type getSomethingQueryHandler struct{}

type (
	GetSomethingQuery    struct{}
	GetSomethingResponse struct{}
)

func (h *getSomethingQueryHandler) H(ctx context.Context, query GetSomethingQuery) (GetSomethingResponse, error) {
	return GetSomethingResponse{}, nil
}

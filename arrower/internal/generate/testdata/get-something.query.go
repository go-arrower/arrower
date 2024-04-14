package application

import (
	"context"
	"errors"

	"github.com/go-arrower/arrower/app"
)

var ErrGetSomethingFailed = errors.New("get something failed")

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

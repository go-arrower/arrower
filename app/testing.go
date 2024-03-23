package app

import (
	"context"
	"errors"
)

//
// This file contains convenience helpers you can use to easier test
// your calling code relying on this usecase pattern.
//

var ErrUseCaseFailed = errors.New("usecase failed")

func TestSuccessRequestHandler[Req any, Res any]() Request[Req, Res] {
	return &testSuccessRequestHandler[Req, Res]{}
}

type testSuccessRequestHandler[Req any, Res any] struct{}

func (h *testSuccessRequestHandler[Req, Res]) H(_ context.Context, _ Req) (Res, error) { //nolint:ireturn // valid use of generics
	var result Res

	return result, nil
}

func TestFailureRequestHandler[Req any, Res any]() Request[Req, Res] {
	return &testFailureRequestHandler[Req, Res]{}
}

type testFailureRequestHandler[Req any, Res any] struct{}

func (h *testFailureRequestHandler[Req, Res]) H(_ context.Context, _ Req) (Res, error) { //nolint:ireturn // valid use of generics
	var result Res

	return result, ErrUseCaseFailed
}

func TestSuccessCommandHandler[C any]() Command[C] {
	return &testSuccessCommandHandler[C]{}
}

type testSuccessCommandHandler[C any] struct{}

func (h *testSuccessCommandHandler[C]) H(_ context.Context, _ C) error {
	return nil
}

func TestFailureCommandHandler[C any]() Command[C] {
	return &testFailureCommandHandler[C]{}
}

type testFailureCommandHandler[C any] struct{}

func (h *testFailureCommandHandler[C]) H(_ context.Context, _ C) error {
	return ErrUseCaseFailed
}

func TestSuccessQueryHandler[Q any, Res any]() Query[Q, Res] {
	return &testSuccessQueryHandler[Q, Res]{}
}

type testSuccessQueryHandler[Q any, Res any] struct{}

func (h *testSuccessQueryHandler[Q, Res]) H(_ context.Context, _ Q) (Res, error) { //nolint:ireturn // valid use of generics
	var result Res

	return result, nil
}

func TestFailureQueryHandler[Q any, Res any]() Query[Q, Res] {
	return &testFailureQueryHandler[Q, Res]{}
}

type testFailureQueryHandler[Q any, Res any] struct{}

func (h *testFailureQueryHandler[Q, Res]) H(_ context.Context, _ Q) (Res, error) { //nolint:ireturn // valid use of generics
	var result Res

	return result, ErrUseCaseFailed
}

func TestSuccessJobHandler[J any]() Job[J] {
	return &testSuccessJobHandler[J]{}
}

type testSuccessJobHandler[J any] struct{}

func (h *testSuccessJobHandler[J]) H(_ context.Context, _ J) error {
	return nil
}

func TestFailureJobHandler[J any]() Job[J] {
	return &testFailureJobHandler[J]{}
}

type testFailureJobHandler[J any] struct{}

func (h *testFailureJobHandler[J]) H(_ context.Context, _ J) error {
	return ErrUseCaseFailed
}

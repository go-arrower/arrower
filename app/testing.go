package app

import (
	"context"
	"errors"
)

//
// This file contains convenience helpers you can use to easier test
// your calling code relying on this usecase pattern.
//

var errUseCaseFailed = errors.New("usecase failed")

func TestRequestHandler[Req any, Res any](handlerFunc dualHandlerFunc[Req, Res]) Request[Req, Res] {
	return &testDualHandler[Req, Res]{handler: handlerFunc}
}

func TestCommandHandler[C any](handlerFunc unaryHandlerFunc[C]) Command[C] {
	return &testUnaryHandler[C]{handler: handlerFunc}
}

func TestQueryHandler[Q any, Res any](handlerFunc dualHandlerFunc[Q, Res]) Query[Q, Res] {
	return &testDualHandler[Q, Res]{handler: handlerFunc}
}

func TestJobHandler[J any](handlerFunc unaryHandlerFunc[J]) Job[J] {
	return &testUnaryHandler[J]{handler: handlerFunc}
}

func TestSuccessRequestHandler[Req any, Res any]() Request[Req, Res] {
	return TestRequestHandler(func(_ context.Context, _ Req) (Res, error) {
		var result Res

		return result, nil
	})
}

func TestFailureRequestHandler[Req any, Res any]() Request[Req, Res] {
	return TestRequestHandler(func(_ context.Context, _ Req) (Res, error) {
		var result Res

		return result, errUseCaseFailed
	})
}

func TestSuccessCommandHandler[C any]() Command[C] {
	return TestCommandHandler(func(_ context.Context, _ C) error {
		return nil
	})
}

func TestFailureCommandHandler[C any]() Command[C] {
	return TestCommandHandler(func(_ context.Context, _ C) error {
		return errUseCaseFailed
	})
}

func TestSuccessQueryHandler[Q any, Res any]() Query[Q, Res] {
	return TestQueryHandler(func(_ context.Context, _ Q) (Res, error) {
		var result Res

		return result, nil
	})
}

func TestFailureQueryHandler[Q any, Res any]() Query[Q, Res] {
	return TestQueryHandler(func(_ context.Context, _ Q) (Res, error) {
		var result Res

		return result, errUseCaseFailed
	})
}

func TestSuccessJobHandler[J any]() Job[J] {
	return TestJobHandler(func(_ context.Context, _ J) error {
		return nil
	})
}

func TestFailureJobHandler[J any]() Job[J] {
	return TestJobHandler(func(_ context.Context, _ J) error {
		return errUseCaseFailed
	})
}

type dualHandlerFunc[Req any, Res any] func(_ context.Context, _ Req) (Res, error)

type testDualHandler[Req any, Res any] struct {
	handler dualHandlerFunc[Req, Res]
}

func (h *testDualHandler[Req, Res]) H(ctx context.Context, req Req) (Res, error) {
	return h.handler(ctx, req)
}

type unaryHandlerFunc[C any] func(_ context.Context, _ C) error

type testUnaryHandler[C any] struct {
	handler unaryHandlerFunc[C]
}

func (h *testUnaryHandler[C]) H(ctx context.Context, cmd C) error {
	return h.handler(ctx, cmd)
}

package console

import (
	"context"
	"errors"
	"fmt"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
)

var (
	errUnexpectedResult = errors.New("unexpected query result")
	errMissingEvent     = errors.New("expected event not returned")
)

type commandBus interface {
	Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error)
}

type queryBus interface {
	Handle(ctx context.Context, query cqs.Query) (any, error)
}

func ask[T any](ctx context.Context, bus queryBus, query cqs.Query) (T, error) {
	var zero T

	result, err := bus.Handle(ctx, query)
	if err != nil {
		return zero, err
	}

	typed, ok := result.(T)
	if !ok {
		return zero, fmt.Errorf("%w: %s returned %T", errUnexpectedResult, query.Name(), result)
	}

	return typed, nil
}

func dispatch[T event.Event](ctx context.Context, bus commandBus, cmd cqs.Command) (T, error) {
	var zero T

	events, err := bus.Handle(ctx, cmd)
	if err != nil {
		return zero, err
	}

	for _, recorded := range events {
		if typed, ok := recorded.(T); ok {
			return typed, nil
		}
	}

	return zero, fmt.Errorf("%w: %s returned %d events", errMissingEvent, cmd.Name(), len(events))
}

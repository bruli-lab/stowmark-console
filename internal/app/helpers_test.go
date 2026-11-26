package app_test

import (
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/stretchr/testify/require"
)

var errBackend = errors.New("backend unavailable")

type unknownMessage struct{}

func (unknownMessage) Name() string {
	return "unknown"
}

func singleEvent[T event.Event](t *testing.T, events []event.Event, name string) T {
	t.Helper()

	require.Len(t, events, 1)
	require.Equal(t, name, events[0].EventName())
	typed, ok := events[0].(T)
	require.True(t, ok)

	return typed
}

func requireInvalidCommand(t *testing.T, handler cqs.CommandHandler) {
	t.Helper()

	events, err := handler.Handle(t.Context(), unknownMessage{})

	require.ErrorAs(t, err, &cqs.InvalidCommandError{})
	require.Nil(t, events)
}

func requireInvalidQuery(t *testing.T, handler cqs.QueryHandler) {
	t.Helper()

	result, err := handler.Handle(t.Context(), unknownMessage{})

	require.ErrorAs(t, err, &cqs.InvalidQueryError{})
	require.Nil(t, result)
}

package store

import (
	"context"

	"github.com/bruli-lab/stowmark/pkg/stowmark"
)

type Session struct {
	handler *stowmark.Handler
}

func Open(ctx context.Context, repositoryPath string, privateKey *string) (*Session, error) {
	var options []stowmark.Option
	if privateKey != nil {
		options = append(options, stowmark.WithPrivateKey(*privateKey))
	}

	handler, err := stowmark.NewHandler(ctx, repositoryPath, options...)
	if err != nil {
		return nil, err
	}

	return &Session{handler: handler}, nil
}

func (s *Session) Close() error {
	return s.handler.Close()
}

package store

import (
	"context"
	"errors"

	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/progress"
	"github.com/bruli-lab/stowmark/pkg/stowmark"
)

var ErrNotConnected = errors.New("no repository is open")

type Session struct {
	handler           *stowmark.Handler
	missingPrivateKey bool
	progress          *progressReporter
}

type SessionOption func(*Session)

func WithProgress(listener progress.Listener) SessionOption {
	return func(s *Session) {
		s.progress = newProgressReporter(listener)
	}
}

func NewSession(options ...SessionOption) *Session {
	session := &Session{}
	for _, option := range options {
		option(session)
	}

	return session
}

func Open(ctx context.Context, repositoryURL string, privateKey *string, options ...SessionOption) (*Session, error) {
	session := NewSession(options...)
	if err := session.Connect(ctx, repositoryURL, privateKey); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Session) Connect(ctx context.Context, repositoryURL string, privateKey *string) error {
	var options []stowmark.Option
	if privateKey != nil {
		options = append(options, stowmark.WithPrivateKey(*privateKey))
	}
	if s.progress != nil {
		options = append(options, stowmark.WithProgress(s.progress))
	}

	handler, err := stowmark.NewHandler(ctx, repositoryURL, options...)
	if err != nil {
		return err
	}

	previous := s.handler
	s.handler = handler
	s.missingPrivateKey = false

	if previous != nil {
		return previous.Close()
	}

	return nil
}

func (s *Session) current() (*stowmark.Handler, error) {
	if s.handler == nil {
		return nil, ErrNotConnected
	}

	return s.handler, nil
}

func (s *Session) writable() error {
	if s.missingPrivateKey {
		return encryption.ErrPrivateKeyRequired
	}

	return nil
}

func (s *Session) Close() error {
	if s.handler == nil {
		return nil
	}

	err := s.handler.Close()
	s.handler = nil

	return err
}

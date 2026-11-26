package store

import (
	"context"
	"errors"

	"github.com/bruli-lab/stowmark/pkg/stowmark"
)

var ErrNotConnected = errors.New("no repository is open")

type Session struct {
	handler *stowmark.Handler
}

func NewSession() *Session {
	return &Session{}
}

func Open(ctx context.Context, repositoryURL string, privateKey *string) (*Session, error) {
	session := NewSession()
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

	handler, err := stowmark.NewHandler(ctx, repositoryURL, options...)
	if err != nil {
		return err
	}

	previous := s.handler
	s.handler = handler

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

func (s *Session) Close() error {
	if s.handler == nil {
		return nil
	}

	err := s.handler.Close()
	s.handler = nil

	return err
}

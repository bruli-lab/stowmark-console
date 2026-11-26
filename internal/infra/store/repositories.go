package store

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark/pkg/stowmark"
)

const configNotFoundMessage = "config file not found"

type Repositories struct {
	session *Session
}

func NewRepositories(session *Session) *Repositories {
	return &Repositories{session: session}
}

func (r *Repositories) Initialized(ctx context.Context) (bool, error) {
	handler, err := r.session.current()
	if err != nil {
		return false, err
	}

	_, err = handler.ListSnapshots(ctx)

	switch {
	case err == nil:
		return true, nil
	case isConfigNotFound(err):
		return false, nil
	default:
		return false, err
	}
}

func (r *Repositories) Initialize(ctx context.Context, options repository.InitOptions) error {
	handler, err := r.session.current()
	if err != nil {
		return err
	}

	return handler.InitWithOptions(ctx, stowmark.InitOptions{
		Compression: &stowmark.Compression{
			Type:  options.Compression(),
			Level: options.CompressionLevel(),
		},
		PublicKey: options.PublicKey(),
	})
}

func isConfigNotFound(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || strings.Contains(err.Error(), configNotFoundMessage)
}

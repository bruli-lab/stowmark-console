package store

import (
	"context"
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
	_, err := r.session.handler.ListSnapshots(ctx)

	switch {
	case err == nil:
		return true, nil
	case strings.Contains(err.Error(), configNotFoundMessage):
		return false, nil
	default:
		return false, err
	}
}

func (r *Repositories) Initialize(ctx context.Context, options repository.InitOptions) error {
	return r.session.handler.InitWithOptions(ctx, stowmark.InitOptions{
		Compression: &stowmark.Compression{
			Type:  options.Compression(),
			Level: options.CompressionLevel(),
		},
		PublicKey: options.PublicKey(),
	})
}

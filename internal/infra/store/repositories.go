package store

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
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

func (r *Repositories) Status(ctx context.Context) (repository.Status, error) {
	handler, err := r.session.current()
	if err != nil {
		return repository.StatusUninitialized, err
	}

	_, err = handler.ListSnapshots(ctx)
	err = translate(err)

	switch {
	case err == nil:
		return repository.StatusReady, nil
	case errors.Is(err, encryption.ErrPrivateKeyRequired):
		return repository.StatusKeyRequired, nil
	case isKeyProblem(err):
		return repository.StatusUninitialized, err
	case isConfigNotFound(err):
		return repository.StatusUninitialized, nil
	default:
		return repository.StatusUninitialized, err
	}
}

func (r *Repositories) Initialize(ctx context.Context, options repository.InitOptions) error {
	handler, err := r.session.current()
	if err != nil {
		return err
	}

	return translate(handler.InitWithOptions(ctx, stowmark.InitOptions{
		Compression: &stowmark.Compression{
			Type:  options.Compression(),
			Level: options.CompressionLevel(),
		},
		FormatVersion: options.FormatVersion(),
		PublicKey:     options.PublicKey(),
	}))
}

func (r *Repositories) Unlock(ctx context.Context, all bool) (repository.LockReport, error) {
	handler, err := r.session.current()
	if err != nil {
		return repository.LockReport{}, err
	}

	result, err := handler.Unlock(ctx, all)
	if err != nil {
		return repository.LockReport{}, translate(err)
	}

	return repository.LockReport{Removed: result.RemovedLocks, Active: result.ActiveLocks}, nil
}

func isConfigNotFound(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || strings.Contains(err.Error(), configNotFoundMessage)
}

func isKeyProblem(err error) bool {
	return errors.Is(err, encryption.ErrPrivateKeyUnreadable) ||
		errors.Is(err, encryption.ErrWrongPrivateKey) ||
		errors.Is(err, encryption.ErrNotEncrypted)
}

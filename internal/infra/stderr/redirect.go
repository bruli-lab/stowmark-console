package stderr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const descriptor = 2

type Redirection struct {
	original int
	target   *os.File
}

func RedirectTo(path string) (*Redirection, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	target, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log file %q: %w", path, err)
	}

	original, err := unix.Dup(descriptor)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("duplicate stderr: %w", err), target.Close())
	}

	if err := unix.Dup3(int(target.Fd()), descriptor, 0); err != nil {
		return nil, errors.Join(fmt.Errorf("redirect stderr: %w", err), unix.Close(original), target.Close())
	}

	return &Redirection{original: original, target: target}, nil
}

func (r *Redirection) Restore() error {
	return errors.Join(
		unix.Dup3(r.original, descriptor, 0),
		unix.Close(r.original),
		r.target.Close(),
	)
}

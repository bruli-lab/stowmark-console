package store

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark/pkg/stowmark"
)

const lockedMessage = "repository is locked"

type domainError struct {
	domain error
	cause  error
}

func (e domainError) Error() string {
	return e.cause.Error()
}

func (e domainError) Unwrap() []error {
	return []error{e.domain, e.cause}
}

var encryptionMessages = []struct {
	message string
	domain  error
}{
	{message: "private key is required to read the data", domain: encryption.ErrPrivateKeyRequired},
	{message: "decrypt symmetric key", domain: encryption.ErrWrongPrivateKey},
	{message: "decrypt metadata key", domain: encryption.ErrWrongPrivateKey},
	{message: "read private key", domain: encryption.ErrPrivateKeyUnreadable},
	{message: "parse private key", domain: encryption.ErrPrivateKeyUnreadable},
	{message: "validate private key", domain: encryption.ErrPrivateKeyUnreadable},
	{message: "invalid private key PEM", domain: encryption.ErrPrivateKeyUnreadable},
	{message: "private key is not RSA", domain: encryption.ErrPrivateKeyUnreadable},
	{message: "missing encryption config to encrypt", domain: encryption.ErrNotEncrypted},
	{message: "repository is not encrypted", domain: encryption.ErrNotEncrypted},
}

func translate(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, stowmark.ErrPrivateKeyRequired) {
		return fmt.Errorf("%w: %w", encryption.ErrPrivateKeyRequired, err)
	}

	if strings.Contains(err.Error(), lockedMessage) {
		return domainError{domain: repository.ErrLocked, cause: err}
	}

	for _, known := range encryptionMessages {
		if strings.Contains(err.Error(), known.message) {
			return fmt.Errorf("%w: %w", known.domain, err)
		}
	}

	return err
}

package console

import (
	"errors"
	"fmt"

	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

const environmentRepositoryName = "default"

type keyError struct {
	message string
	cause   error
}

func (e keyError) Error() string {
	return e.message
}

func (e keyError) Unwrap() error {
	return e.cause
}

func explainKeyError(profile repository.Profile, err error) error {
	name := fmt.Sprintf("%q", profile.Name())
	key := "(not set)"
	if profile.PrivateKey() != nil {
		key = *profile.PrivateKey()
	}

	var message string

	switch {
	case errors.Is(err, encryption.ErrPrivateKeyRequired):
		message = fmt.Sprintf("repository %s is encrypted and needs its private key: %s", name, privateKeyField(profile))
	case errors.Is(err, encryption.ErrWrongPrivateKey):
		message = fmt.Sprintf("the private key %s does not open repository %s: %s", key, name, privateKeyField(profile))
	case errors.Is(err, encryption.ErrPrivateKeyUnreadable):
		message = fmt.Sprintf("the private key of repository %s cannot be read (%v): %s", name, err, privateKeyField(profile))
	case errors.Is(err, encryption.ErrNotEncrypted):
		message = fmt.Sprintf("repository %s is not encrypted: remove its private key from the configuration", name)
	default:
		return err
	}

	return keyError{message: message, cause: err}
}

func privateKeyField(profile repository.Profile) string {
	if profile.Name() == environmentRepositoryName {
		return "set STOWMARK_PRIVATE_KEY (or private_key in repositories.yaml) to its private key file"
	}

	return fmt.Sprintf("set private_key for %q in repositories.yaml to its private key file", profile.Name())
}

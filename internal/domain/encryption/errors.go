package encryption

import "errors"

var (
	ErrPrivateKeyRequired   = errors.New("the repository is encrypted and needs its private key")
	ErrWrongPrivateKey      = errors.New("the private key does not decrypt the repository")
	ErrPrivateKeyUnreadable = errors.New("the private key cannot be read")
	ErrNotEncrypted         = errors.New("the repository is not encrypted")
	ErrKeyPathNotAbsolute   = errors.New("key paths must be absolute")
)

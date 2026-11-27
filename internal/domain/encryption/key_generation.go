package encryption

import (
	"errors"
	"path/filepath"

	"github.com/bruli-lab/go-core/event"
)

var ErrKeyFolderNotAbsolute = errors.New("the key folder must be an absolute path")

type KeyGeneration struct {
	event.BasicAggregateRoot
	folder string
}

func NewKeyGeneration(folder string) (*KeyGeneration, error) {
	if !filepath.IsAbs(folder) {
		return nil, ErrKeyFolderNotAbsolute
	}

	return &KeyGeneration{BasicAggregateRoot: event.NewBasicAggregateRoot(), folder: folder}, nil
}

func (g *KeyGeneration) Folder() string {
	return g.folder
}

func (g *KeyGeneration) Generated(keys KeyPair) error {
	if _, err := NewKeyPair(keys.PrivateKey(), keys.PublicKey()); err != nil {
		return err
	}

	g.Record(NewKeysGenerated(g.folder, keys))

	return nil
}

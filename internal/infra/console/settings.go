package console

import (
	"fmt"

	"github.com/bruli-lab/stowmark/pkg/stowmark"
)

type Settings struct {
	RepositoryPath string
	PrivateKey     *string
	Init           InitSettings
}

type InitSettings struct {
	PublicKey        *string
	Compression      string
	CompressionLevel *int
}

func (s InitSettings) options() stowmark.InitOptions {
	return stowmark.InitOptions{
		Compression: &stowmark.Compression{Type: s.compressionType(), Level: s.CompressionLevel},
		PublicKey:   s.PublicKey,
	}
}

func (s InitSettings) compressionType() string {
	if s.Compression == "" {
		return "none"
	}

	return s.Compression
}

func (s InitSettings) encryptionSummary() string {
	if s.PublicKey == nil {
		return "none (set STOWMARK_PUBLIC_KEY to enable it)"
	}

	return "public key " + *s.PublicKey
}

func (s InitSettings) compressionSummary() string {
	if s.CompressionLevel == nil {
		return s.compressionType()
	}

	return fmt.Sprintf("%s (level %d)", s.compressionType(), *s.CompressionLevel)
}

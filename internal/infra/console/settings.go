package console

import (
	"fmt"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

type Settings struct {
	RepositoryPath string
	Init           repository.InitOptions
}

func encryptionSummary(options repository.InitOptions) string {
	if !options.Encrypted() {
		return "none (set STOWMARK_PUBLIC_KEY to enable it)"
	}

	return "public key " + *options.PublicKey()
}

func compressionSummary(options repository.InitOptions) string {
	if options.CompressionLevel() == nil {
		return options.Compression()
	}

	return fmt.Sprintf("%s (level %d)", options.Compression(), *options.CompressionLevel())
}

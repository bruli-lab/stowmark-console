package console

import (
	"fmt"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

func encryptionSummary(options repository.InitOptions) string {
	if !options.Encrypted() {
		return "none (set a public key to enable it)"
	}

	return "public key " + *options.PublicKey()
}

func compressionSummary(options repository.InitOptions) string {
	if options.CompressionLevel() == nil {
		return options.Compression()
	}

	return fmt.Sprintf("%s (level %d)", options.Compression(), *options.CompressionLevel())
}

func formatSummary(options repository.InitOptions) string {
	return fmt.Sprintf("version %d", options.FormatVersion())
}

func formatRecommendation(options repository.InitOptions) string {
	if !options.Encrypted() || options.FormatVersion() >= repository.LatestFormatVersion {
		return ""
	}

	return "Format version 3 is recommended for encrypted repositories: it also encrypts\n" +
		"the manifests and streams the encryption. Set format_version: 3 to use it."
}

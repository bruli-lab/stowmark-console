package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/caarlos0/env/v11"
)

var ErrInvalidCompression = errors.New("invalid compression type")

var compressionTypes = []string{"none", "zstd", "lz4", "xz", "gzip"}

type Config struct {
	Repository       string  `env:"STOWMARK_REPOSITORY,required,notEmpty"`
	PrivateKey       *string `env:"STOWMARK_PRIVATE_KEY"`
	PublicKey        *string `env:"STOWMARK_PUBLIC_KEY"`
	Compression      string  `env:"STOWMARK_COMPRESSION" envDefault:"none"`
	CompressionLevel *int    `env:"STOWMARK_COMPRESSION_LEVEL"`
	LogFile          string  `env:"STOWMARK_CONSOLE_LOG_FILE"`
}

func New() (*Config, error) {
	return FromEnvironment(env.ToMap(os.Environ()))
}

func FromEnvironment(environment map[string]string) (*Config, error) {
	conf, err := env.ParseAsWithOptions[Config](env.Options{Environment: environment})
	if err != nil {
		return nil, err
	}

	if !slices.Contains(compressionTypes, conf.Compression) {
		return nil, fmt.Errorf("%w %q, expected one of %v", ErrInvalidCompression, conf.Compression, compressionTypes)
	}

	if conf.LogFile == "" {
		conf.LogFile, err = defaultLogFile()
		if err != nil {
			return nil, err
		}
	}

	return &conf, nil
}

func defaultLogFile() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve cache directory: %w", err)
	}

	return filepath.Join(cacheDir, "stowmark-console", "console.log"), nil
}

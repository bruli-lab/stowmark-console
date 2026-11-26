package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const RepositoryEnvVar = "STOWMARK_REPOSITORY"

type Config struct {
	repository     string
	repositoryType RepositoryType
	logFile        string
}

func (c Config) Repository() string {
	return c.repository
}

func (c Config) RepositoryType() RepositoryType {
	return c.repositoryType
}

func (c Config) LogFile() string {
	return c.logFile
}

func New() (*Config, error) {
	repoPath := os.Getenv(RepositoryEnvVar)
	repoType, err := ParseRepositoryType(repoPath)
	if err != nil {
		return nil, err
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("resolve cache directory: %w", err)
	}

	return &Config{
		repository:     repoPath,
		repositoryType: repoType,
		logFile:        filepath.Join(cacheDir, "stowmark-console", "console.log"),
	}, nil
}

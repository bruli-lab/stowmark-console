package config

import "os"

const RepositoryEnvVar = "STOWMARK_REPOSITORY"

type Config struct {
	repository     string
	repositoryType RepositoryType
}

func (c Config) Repository() string {
	return c.repository
}

func (c Config) RepositoryType() RepositoryType {
	return c.repositoryType
}

func New() (*Config, error) {
	repoPath := os.Getenv(RepositoryEnvVar)
	repoType, err := ParseRepositoryType(repoPath)
	if err != nil {
		return nil, err
	}
	return &Config{
		repository:     repoPath,
		repositoryType: repoType,
	}, nil
}

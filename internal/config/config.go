package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
	"github.com/caarlos0/env/v11"
	"go.yaml.in/yaml/v3"
)

const DefaultRepositoryName = "default"

var (
	ErrInvalidCompression = errors.New("invalid compression type")
	ErrNoRepositories     = errors.New("no repositories configured")
	ErrInvalidRepository  = errors.New("invalid repository")
	ErrInvalidFormat      = errors.New("invalid format version")
	ErrInvalidRetention   = errors.New("invalid retention")
)

const (
	defaultFormatVersion = 2
	latestFormatVersion  = 3
)

var compressionTypes = []string{"none", "zstd", "lz4", "xz", "gzip"}

type Config struct {
	LogFile          string
	RepositoriesFile string
	Repositories     []Repository
}

type Repository struct {
	Name             string            `yaml:"name"`
	URL              string            `yaml:"url"`
	PrivateKey       *string           `yaml:"private_key"`
	PublicKey        *string           `yaml:"public_key"`
	Compression      string            `yaml:"compression"`
	CompressionLevel *int              `yaml:"compression_level"`
	FormatVersion    int               `yaml:"format_version"`
	Retention        *Retention        `yaml:"retention"`
	Env              map[string]string `yaml:"env"`
}

type Retention struct {
	KeepLast    int           `yaml:"keep_last"`
	KeepDaily   int           `yaml:"keep_daily"`
	KeepWeekly  int           `yaml:"keep_weekly"`
	KeepMonthly int           `yaml:"keep_monthly"`
	KeepYearly  int           `yaml:"keep_yearly"`
	KeepWithin  string        `yaml:"keep_within"`
	Within      time.Duration `yaml:"-"`
}

type environmentConfig struct {
	Repository       string  `env:"STOWMARK_REPOSITORY"`
	PrivateKey       *string `env:"STOWMARK_PRIVATE_KEY"`
	PublicKey        *string `env:"STOWMARK_PUBLIC_KEY"`
	Compression      string  `env:"STOWMARK_COMPRESSION"`
	CompressionLevel *int    `env:"STOWMARK_COMPRESSION_LEVEL"`
	FormatVersion    int     `env:"STOWMARK_FORMAT_VERSION"`
	LogFile          string  `env:"STOWMARK_CONSOLE_LOG_FILE"`
	RepositoriesFile string  `env:"STOWMARK_CONSOLE_CONFIG"`
}

type repositoriesFile struct {
	Repositories []Repository `yaml:"repositories"`
}

func New() (*Config, error) {
	return FromEnvironment(env.ToMap(os.Environ()))
}

func FromEnvironment(environment map[string]string) (*Config, error) {
	values, err := env.ParseAsWithOptions[environmentConfig](env.Options{Environment: environment})
	if err != nil {
		return nil, err
	}

	conf := Config{LogFile: values.LogFile, RepositoriesFile: values.RepositoriesFile}

	if conf.LogFile == "" {
		if conf.LogFile, err = defaultPath(os.UserCacheDir, "console.log"); err != nil {
			return nil, err
		}
	}

	if conf.RepositoriesFile == "" {
		if conf.RepositoriesFile, err = defaultPath(os.UserConfigDir, "repositories.yaml"); err != nil {
			return nil, err
		}
	}

	if values.Repository != "" {
		conf.Repositories = append(conf.Repositories, Repository{
			Name:             DefaultRepositoryName,
			URL:              values.Repository,
			PrivateKey:       values.PrivateKey,
			PublicKey:        values.PublicKey,
			Compression:      values.Compression,
			CompressionLevel: values.CompressionLevel,
			FormatVersion:    values.FormatVersion,
		})
	}

	configured, err := readRepositories(conf.RepositoriesFile)
	if err != nil {
		return nil, err
	}

	conf.Repositories = append(conf.Repositories, configured...)

	if err := normalize(conf.Repositories); err != nil {
		return nil, err
	}

	if len(conf.Repositories) == 0 {
		return nil, fmt.Errorf("%w: set STOWMARK_REPOSITORY or create %s", ErrNoRepositories, conf.RepositoriesFile)
	}

	return &conf, nil
}

func defaultPath(baseDir func() (string, error), name string) (string, error) {
	base, err := baseDir()
	if err != nil {
		return "", fmt.Errorf("resolve %s directory: %w", name, err)
	}

	return filepath.Join(base, "stowmark-console", name), nil
}

func readRepositories(path string) ([]Repository, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)

	var file repositoriesFile
	if err := decoder.Decode(&file); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return file.Repositories, nil
}

func normalize(repositories []Repository) error {
	names := make(map[string]bool, len(repositories))

	for i := range repositories {
		repo := &repositories[i]

		if err := validate(repo, names); err != nil {
			return err
		}
		if err := normalizeRetention(repo); err != nil {
			return err
		}
		names[repo.Name] = true

		if repo.Compression == "" {
			repo.Compression = compressionTypes[0]
		}
		if repo.FormatVersion == 0 {
			repo.FormatVersion = defaultFormatVersion
		}
		if !strings.Contains(repo.URL, "://") {
			repo.URL = expandHome(repo.URL)
		}
		repo.PrivateKey = expandHomePointer(repo.PrivateKey)
		repo.PublicKey = expandHomePointer(repo.PublicKey)
	}

	return nil
}

func validate(repo *Repository, names map[string]bool) error {
	switch {
	case repo.Name == "":
		return fmt.Errorf("%w: every repository needs a name", ErrInvalidRepository)
	case names[repo.Name]:
		return fmt.Errorf("%w: the name %q is used more than once", ErrInvalidRepository, repo.Name)
	case repo.URL == "":
		return fmt.Errorf("%w: %q needs a url", ErrInvalidRepository, repo.Name)
	case repo.Compression != "" && !slices.Contains(compressionTypes, repo.Compression):
		return fmt.Errorf("%w %q in %q, expected one of %v", ErrInvalidCompression, repo.Compression, repo.Name, compressionTypes)
	case repo.FormatVersion < 0 || repo.FormatVersion > latestFormatVersion:
		return fmt.Errorf("%w %d in %q, expected 1, 2 or 3", ErrInvalidFormat, repo.FormatVersion, repo.Name)
	}

	return nil
}

func normalizeRetention(repo *Repository) error {
	if repo.Retention == nil {
		return nil
	}

	keep := repo.Retention
	if slices.ContainsFunc([]int{keep.KeepLast, keep.KeepDaily, keep.KeepWeekly, keep.KeepMonthly, keep.KeepYearly},
		func(count int) bool { return count < 0 }) {
		return fmt.Errorf("%w in %q: keep rules must not be negative", ErrInvalidRetention, repo.Name)
	}

	within, err := retention.ParseWithin(keep.KeepWithin)
	if err != nil {
		return fmt.Errorf("%w in %q: %w", ErrInvalidRetention, repo.Name, err)
	}
	keep.Within = within

	return nil
}

func expandHomePointer(path *string) *string {
	if path == nil {
		return nil
	}

	expanded := expandHome(*path)

	return &expanded
}

func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}

package repository

import (
	"errors"
	"fmt"
	"slices"
)

const (
	NoCompression        = "none"
	DefaultFormatVersion = 2
	LatestFormatVersion  = 3
	MinimumFormatVersion = 1
)

var (
	ErrInvalidCompression      = errors.New("invalid compression")
	ErrInvalidCompressionLevel = errors.New("invalid compression level")
	ErrInvalidFormatVersion    = errors.New("invalid format version")
)

type CompressionLevels struct {
	Minimum int
	Maximum int
	Default int
}

var compressionLevels = map[string]*CompressionLevels{
	NoCompression: nil,
	"zstd":        {Minimum: 1, Maximum: 22, Default: 3},
	"lz4":         {Minimum: 0, Maximum: 9, Default: 0},
	"gzip":        {Minimum: 1, Maximum: 9, Default: 6},
	"xz":          nil,
}

func Compressions() []string {
	return []string{NoCompression, "zstd", "lz4", "gzip", "xz"}
}

func LevelsOf(compression string) (CompressionLevels, bool) {
	levels := compressionLevels[compression]
	if levels == nil {
		return CompressionLevels{}, false
	}

	return *levels, true
}

type InitOptions struct {
	compression      string
	compressionLevel *int
	publicKey        *string
	formatVersion    int
}

func NewInitOptions(compression string, compressionLevel *int, publicKey *string) InitOptions {
	if compression == "" {
		compression = NoCompression
	}

	return InitOptions{
		compression:      compression,
		compressionLevel: compressionLevel,
		publicKey:        publicKey,
	}
}

func (o InitOptions) WithFormatVersion(version int) InitOptions {
	o.formatVersion = version
	return o
}

func (o InitOptions) WithPublicKey(path string) InitOptions {
	o.publicKey = &path
	return o
}

func (o InitOptions) Compression() string {
	if o.compression == "" {
		return NoCompression
	}

	return o.compression
}

func (o InitOptions) CompressionLevel() *int {
	return o.compressionLevel
}

func (o InitOptions) PublicKey() *string {
	return o.publicKey
}

func (o InitOptions) Encrypted() bool {
	return o.publicKey != nil
}

func (o InitOptions) FormatVersion() int {
	if o.formatVersion == 0 {
		return DefaultFormatVersion
	}

	return o.formatVersion
}

func (o InitOptions) Validate() error {
	if !slices.Contains(Compressions(), o.Compression()) {
		return fmt.Errorf("%w %q: use one of %v", ErrInvalidCompression, o.Compression(), Compressions())
	}

	if version := o.FormatVersion(); version < MinimumFormatVersion || version > LatestFormatVersion {
		return fmt.Errorf("%w %d: use %d to %d", ErrInvalidFormatVersion, version, MinimumFormatVersion, LatestFormatVersion)
	}

	if o.compressionLevel == nil {
		return nil
	}

	levels, ok := LevelsOf(o.Compression())
	if !ok {
		return fmt.Errorf("%w: %s has no levels", ErrInvalidCompressionLevel, o.Compression())
	}

	if level := *o.compressionLevel; level < levels.Minimum || level > levels.Maximum {
		return fmt.Errorf("%w %d: %s uses %d to %d", ErrInvalidCompressionLevel, level, o.Compression(), levels.Minimum, levels.Maximum)
	}

	return nil
}

func (o InitOptions) EncryptsMetadata() bool {
	return o.Encrypted() && o.FormatVersion() >= LatestFormatVersion
}

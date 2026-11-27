package repository

const (
	NoCompression        = "none"
	DefaultFormatVersion = 2
	LatestFormatVersion  = 3
)

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

func (o InitOptions) EncryptsMetadata() bool {
	return o.Encrypted() && o.FormatVersion() >= LatestFormatVersion
}

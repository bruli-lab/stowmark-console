package repository

const NoCompression = "none"

type InitOptions struct {
	compression      string
	compressionLevel *int
	publicKey        *string
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

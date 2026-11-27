package encryption

import "context"

type Keys interface {
	Generate(ctx context.Context, generation *KeyGeneration) (KeyPair, error)
	Rewrap(ctx context.Context, rewrap *Rewrap) error
	Rotate(ctx context.Context, rotation *Rotation) error
}

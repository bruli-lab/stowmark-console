package encryption

import "github.com/bruli-lab/go-core/event"

type Rotation struct {
	event.BasicAggregateRoot
	privateKey string
	publicKey  string
}

func NewRotation(privateKey, publicKey string) (*Rotation, error) {
	if err := validateKeyPaths(privateKey, publicKey); err != nil {
		return nil, err
	}

	return &Rotation{
		BasicAggregateRoot: event.NewBasicAggregateRoot(),
		privateKey:         privateKey,
		publicKey:          publicKey,
	}, nil
}

func (r *Rotation) PrivateKey() string {
	return r.privateKey
}

func (r *Rotation) PublicKey() string {
	return r.publicKey
}

func (r *Rotation) Rotated() {
	r.Record(NewKeyRotated(r.privateKey, r.publicKey))
}

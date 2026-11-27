package encryption_test

import (
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/stretchr/testify/require"
)

const (
	privateKey = "/keys/1234-private.pem"
	publicKey  = "/keys/1234-public.pem"
	newPublic  = "/keys/5678-public.pem"
)

func TestKeyPair(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		private, public string
		err             error
	}{
		"a pair without a private key":    {public: publicKey, err: encryption.ErrKeyPairIncomplete},
		"a pair without a public key":     {private: privateKey, err: encryption.ErrKeyPairIncomplete},
		"a pair with a relative key path": {private: "keys/private.pem", public: publicKey, err: encryption.ErrKeyPathNotAbsolute},
	}

	for name, tt := range tests {
		t.Run(`Given `+name+`
		When it is built
		Then it is rejected`, func(t *testing.T) {
			t.Parallel()

			_, err := encryption.NewKeyPair(tt.private, tt.public)

			require.ErrorIs(t, err, tt.err)
		})
	}

	t.Run(`Given absolute private and public keys
	When the pair is built
	Then it keeps both paths`, func(t *testing.T) {
		t.Parallel()

		keys, err := encryption.NewKeyPair(privateKey, publicKey)

		require.NoError(t, err)
		require.Equal(t, privateKey, keys.PrivateKey())
		require.Equal(t, publicKey, keys.PublicKey())
	})
}

func TestKeyGeneration(t *testing.T) {
	t.Parallel()

	t.Run(`Given a relative folder
	When a key generation is built
	Then it is rejected`, func(t *testing.T) {
		t.Parallel()

		_, err := encryption.NewKeyGeneration("keys")

		require.ErrorIs(t, err, encryption.ErrKeyFolderNotAbsolute)
	})

	t.Run(`Given a key generation
	When the keys are generated
	Then a keys generated event with the pair is recorded`, func(t *testing.T) {
		t.Parallel()
		generation, err := encryption.NewKeyGeneration("/keys")
		require.NoError(t, err)
		keys, err := encryption.NewKeyPair(privateKey, publicKey)
		require.NoError(t, err)

		require.NoError(t, generation.Generated(keys))

		events := generation.Events()
		require.Len(t, events, 1)
		generated, ok := events[0].(encryption.KeysGenerated)
		require.True(t, ok)
		require.Equal(t, encryption.KeysGeneratedEventName, generated.EventName())
		require.Equal(t, "/keys", generated.AggregateRootID())
		require.Equal(t, keys, generated.Keys)
	})

	t.Run(`Given a key generation
	When it is marked as generated with an empty pair
	Then it fails and records nothing`, func(t *testing.T) {
		t.Parallel()
		generation, err := encryption.NewKeyGeneration("/keys")
		require.NoError(t, err)

		require.ErrorIs(t, generation.Generated(encryption.KeyPair{}), encryption.ErrKeyPairIncomplete)
		require.Empty(t, generation.Events())
	})
}

func TestRewrap(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		oldPrivate, newPublic string
		err                   error
	}{
		"no old private key":        {newPublic: newPublic, err: encryption.ErrPrivateKeyPathRequired},
		"no new public key":         {oldPrivate: privateKey, err: encryption.ErrPublicKeyPathRequired},
		"a relative new public key": {oldPrivate: privateKey, newPublic: "new.pem", err: encryption.ErrKeyPathNotAbsolute},
	}

	for name, tt := range tests {
		t.Run(`Given `+name+`
		When a rewrap is built
		Then it is rejected`, func(t *testing.T) {
			t.Parallel()

			_, err := encryption.NewRewrap(tt.oldPrivate, tt.newPublic)

			require.ErrorIs(t, err, tt.err)
		})
	}

	t.Run(`Given the old private key and a new public key
	When the rewrap finishes
	Then a key rewrapped event is recorded`, func(t *testing.T) {
		t.Parallel()
		rewrap, err := encryption.NewRewrap(privateKey, newPublic)
		require.NoError(t, err)
		require.Equal(t, privateKey, rewrap.OldPrivateKey())
		require.Equal(t, newPublic, rewrap.NewPublicKey())

		rewrap.Rewrapped()

		events := rewrap.Events()
		require.Len(t, events, 1)
		rewrapped, ok := events[0].(encryption.KeyRewrapped)
		require.True(t, ok)
		require.Equal(t, encryption.KeyRewrappedEventName, rewrapped.EventName())
		require.Equal(t, privateKey, rewrapped.OldPrivateKey)
		require.Equal(t, newPublic, rewrapped.NewPublicKey)
	})
}

func TestRotation(t *testing.T) {
	t.Parallel()

	t.Run(`Given no private key
	When a rotation is built
	Then it is rejected`, func(t *testing.T) {
		t.Parallel()

		_, err := encryption.NewRotation("", publicKey)

		require.ErrorIs(t, err, encryption.ErrPrivateKeyPathRequired)
	})

	t.Run(`Given the private and public keys
	When the rotation finishes
	Then a key rotated event is recorded`, func(t *testing.T) {
		t.Parallel()
		rotation, err := encryption.NewRotation(privateKey, publicKey)
		require.NoError(t, err)
		require.Equal(t, privateKey, rotation.PrivateKey())
		require.Equal(t, publicKey, rotation.PublicKey())

		rotation.Rotated()

		events := rotation.Events()
		require.Len(t, events, 1)
		rotated, ok := events[0].(encryption.KeyRotated)
		require.True(t, ok)
		require.Equal(t, encryption.KeyRotatedEventName, rotated.EventName())
		require.Equal(t, publicKey, rotated.PublicKey)
	})
}

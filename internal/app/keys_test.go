package app_test

import (
	"context"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	"github.com/stretchr/testify/require"
)

const (
	privateKeyPath = "/keys/1234-private.pem"
	publicKeyPath  = "/keys/1234-public.pem"
	newPublicPath  = "/keys/5678-public.pem"
)

func TestGenerateKeys(t *testing.T) {
	t.Parallel()

	t.Run(`Given a command that is not a generate keys command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewGenerateKeys(&mocks.KeysMock{}))
	})

	t.Run(`Given a relative folder
	When the keys are generated
	Then the command is rejected before reaching the keys`, func(t *testing.T) {
		t.Parallel()
		keys := &mocks.KeysMock{}

		_, err := app.NewGenerateKeys(keys).Handle(t.Context(), app.GenerateKeysCommand{Folder: "keys"})

		require.ErrorIs(t, err, encryption.ErrKeyFolderNotAbsolute)
		require.Empty(t, keys.GenerateCalls())
	})

	t.Run(`Given a folder
	When the keys are generated
	Then a keys generated event with the new pair is returned`, func(t *testing.T) {
		t.Parallel()
		pair, err := encryption.NewKeyPair(privateKeyPath, publicKeyPath)
		require.NoError(t, err)
		keys := &mocks.KeysMock{
			GenerateFunc: func(context.Context, *encryption.KeyGeneration) (encryption.KeyPair, error) { return pair, nil },
		}

		events, err := app.NewGenerateKeys(keys).Handle(t.Context(), app.GenerateKeysCommand{Folder: "/keys"})

		require.NoError(t, err)
		generated := singleEvent[encryption.KeysGenerated](t, events, encryption.KeysGeneratedEventName)
		require.Equal(t, pair, generated.Keys)
		require.Equal(t, "/keys", keys.GenerateCalls()[0].Generation.Folder())
	})

	t.Run(`Given keys that cannot be generated
	When the keys are generated
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		keys := &mocks.KeysMock{
			GenerateFunc: func(context.Context, *encryption.KeyGeneration) (encryption.KeyPair, error) {
				return encryption.KeyPair{}, errBackend
			},
		}

		events, err := app.NewGenerateKeys(keys).Handle(t.Context(), app.GenerateKeysCommand{Folder: "/keys"})

		require.ErrorIs(t, err, errBackend)
		require.Nil(t, events)
	})
}

func TestRewrapKey(t *testing.T) {
	t.Parallel()

	t.Run(`Given a command that is not a rewrap key command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewRewrapKey(&mocks.KeysMock{}))
	})

	t.Run(`Given a relative new public key
	When the key is rewrapped
	Then the command is rejected before reaching the keys`, func(t *testing.T) {
		t.Parallel()
		keys := &mocks.KeysMock{}

		_, err := app.NewRewrapKey(keys).Handle(t.Context(), app.RewrapKeyCommand{
			OldPrivateKey: privateKeyPath, NewPublicKey: "public.pem",
		})

		require.ErrorIs(t, err, encryption.ErrKeyPathNotAbsolute)
		require.Empty(t, keys.RewrapCalls())
	})

	t.Run(`Given the old private key and a new public key
	When the key is rewrapped
	Then both reach the keys and a key rewrapped event is returned`, func(t *testing.T) {
		t.Parallel()
		keys := &mocks.KeysMock{RewrapFunc: func(context.Context, *encryption.Rewrap) error { return nil }}

		events, err := app.NewRewrapKey(keys).Handle(t.Context(), app.RewrapKeyCommand{
			OldPrivateKey: privateKeyPath, NewPublicKey: newPublicPath,
		})

		require.NoError(t, err)
		rewrapped := singleEvent[encryption.KeyRewrapped](t, events, encryption.KeyRewrappedEventName)
		require.Equal(t, newPublicPath, rewrapped.NewPublicKey)
		require.Equal(t, privateKeyPath, keys.RewrapCalls()[0].Rewrap.OldPrivateKey())
	})

	t.Run(`Given keys that fail to rewrap
	When the key is rewrapped
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		keys := &mocks.KeysMock{RewrapFunc: func(context.Context, *encryption.Rewrap) error { return errBackend }}

		events, err := app.NewRewrapKey(keys).Handle(t.Context(), app.RewrapKeyCommand{
			OldPrivateKey: privateKeyPath, NewPublicKey: newPublicPath,
		})

		require.ErrorIs(t, err, errBackend)
		require.Nil(t, events)
	})
}

func TestRotateKey(t *testing.T) {
	t.Parallel()

	t.Run(`Given a command that is not a rotate key command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewRotateKey(&mocks.KeysMock{}))
	})

	t.Run(`Given a rotation without public key
	When the key is rotated
	Then the command is rejected before reaching the keys`, func(t *testing.T) {
		t.Parallel()
		keys := &mocks.KeysMock{}

		_, err := app.NewRotateKey(keys).Handle(t.Context(), app.RotateKeyCommand{PrivateKey: privateKeyPath})

		require.ErrorIs(t, err, encryption.ErrPublicKeyPathRequired)
		require.Empty(t, keys.RotateCalls())
	})

	t.Run(`Given the private and public keys
	When the key is rotated
	Then they reach the keys and a key rotated event is returned`, func(t *testing.T) {
		t.Parallel()
		keys := &mocks.KeysMock{RotateFunc: func(context.Context, *encryption.Rotation) error { return nil }}

		events, err := app.NewRotateKey(keys).Handle(t.Context(), app.RotateKeyCommand{
			PrivateKey: privateKeyPath, PublicKey: publicKeyPath,
		})

		require.NoError(t, err)
		rotated := singleEvent[encryption.KeyRotated](t, events, encryption.KeyRotatedEventName)
		require.Equal(t, privateKeyPath, rotated.PrivateKey)
		require.Equal(t, publicKeyPath, keys.RotateCalls()[0].Rotation.PublicKey())
	})

	t.Run(`Given keys that fail to rotate
	When the key is rotated
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		keys := &mocks.KeysMock{RotateFunc: func(context.Context, *encryption.Rotation) error { return errBackend }}

		events, err := app.NewRotateKey(keys).Handle(t.Context(), app.RotateKeyCommand{
			PrivateKey: privateKeyPath, PublicKey: publicKeyPath,
		})

		require.ErrorIs(t, err, errBackend)
		require.Nil(t, events)
	})
}

package console

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/stretchr/testify/require"
)

func (h *harness) initEncrypted(t *testing.T, folder string) {
	t.Helper()
	h.pressAndRun(runes("e"))
	require.NotNil(t, h.prompt())
	h.clearInput()
	h.typeText(folder)
	h.pressAndRun(key(tea.KeyEnter))
}

func TestConsoleEncryptedInit(t *testing.T) {
	t.Parallel()

	t.Run(`Given a repository to initialize that has no public key
	When the user chooses to initialize it encrypted
	Then it asks for the folder of the new key pair, prefilled with the default one`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()

		h.pressAndRun(runes("e"))

		require.NotNil(t, h.prompt())
		require.Equal(t, defaultKeysFolder(), h.prompt().input.Value())
		require.Empty(t, h.keys.GenerateCalls())
	})

	t.Run(`Given the folder prompt of the encrypted init
	When the user goes back
	Then the initialization choice is shown again`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()
		h.pressAndRun(runes("e"))

		h.pressAndRun(key(tea.KeyEsc))

		require.NotNil(t, h.choice())
		require.Empty(t, h.repositories.InitializeCalls())
	})

	t.Run(`Given the folder prompt of the encrypted init
	When the user chooses a folder
	Then a key pair is generated there, the repository is initialized with format version 3 and the session uses the keys`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()
		folder := filepath.Join(t.TempDir(), "keys")

		h.initEncrypted(t, folder)

		require.Equal(t, folder, h.keys.GenerateCalls()[0].Generation.Folder())
		options := h.repositories.InitializeCalls()[0].Options
		require.Equal(t, repository.LatestFormatVersion, options.FormatVersion())
		require.Equal(t, "zstd", options.Compression())
		require.Equal(t, filepath.Join(folder, "1234-public.pem"), *options.PublicKey())
		require.True(t, h.model.current.Encrypted())
		require.Equal(t, filepath.Join(folder, "1234-private.pem"), *h.model.current.PrivateKey())
		require.Equal(t, fingerprint, h.model.current.Fingerprint())
		require.Len(t, h.snapshots.ListCalls(), 1)
		require.Contains(t, h.model.message, "initialized encrypted")
	})

	t.Run(`Given the default repository initialized encrypted
	When the result is shown
	Then it lists the environment variables to set`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()

		h.initEncrypted(t, filepath.Join(t.TempDir(), "keys"))

		view := h.model.dialog.view()
		require.Contains(t, view, "STOWMARK_PRIVATE_KEY=")
		require.Contains(t, view, "STOWMARK_FORMAT_VERSION=3")
		require.Contains(t, view, "Keep the private key safe")
	})

	t.Run(`Given a repository of the repositories file initialized encrypted
	When the result is shown
	Then it lists the lines to add to its entry`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false)
		h.profiles[0] = repository.NewProfile("vault", h.profiles[0].Location(), h.profiles[0].Init())
		h.started()

		h.initEncrypted(t, filepath.Join(t.TempDir(), "keys"))

		view := h.model.dialog.view()
		require.Contains(t, view, "private_key: ")
		require.Contains(t, view, "public_key: ")
		require.Contains(t, view, "format_version: 3")
	})

	t.Run(`Given a key pair that cannot be generated
	When the user initializes the repository encrypted
	Then the error is shown and nothing is initialized`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()
		h.keys.GenerateFunc = func(context.Context, *encryption.KeyGeneration) (encryption.KeyPair, error) {
			return encryption.KeyPair{}, errors.New("permission denied")
		}

		h.initEncrypted(t, filepath.Join(t.TempDir(), "keys"))

		require.ErrorContains(t, h.model.err, "permission denied")
		require.Empty(t, h.repositories.InitializeCalls())
		require.False(t, h.model.current.Encrypted())
	})

	t.Run(`Given a repository whose configuration has a public key
	When it has to be initialized
	Then the confirmation has no encrypted choice`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false)
		h.encrypted(t, 3, true)
		h.started()

		require.NotNil(t, h.confirmation())
		require.NotContains(t, h.view(), "Initialize encrypted")
	})
}

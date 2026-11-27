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

func (h *harness) initForm() *initForm {
	form, _ := h.model.dialog.(*initForm)
	return form
}

func (h *harness) focusInit(t *testing.T, field initField) {
	t.Helper()
	for range 6 {
		if h.initForm().focus == field {
			return
		}
		h.pressAndRun(key(tea.KeyTab))
	}
	require.Equal(t, field, h.initForm().focus)
}

func (h *harness) chooseInit(t *testing.T, field initField, value string) {
	t.Helper()
	h.focusInit(t, field)
	for range 6 {
		if containsValue(h.initForm(), field, value) {
			return
		}
		h.pressAndRun(key(tea.KeyRight))
	}
	require.True(t, containsValue(h.initForm(), field, value), "%s not reached", value)
}

func containsValue(form *initForm, field initField, value string) bool {
	switch field {
	case initCompression:
		return form.compressionName() == value
	case initEncryption:
		return form.encryptionName() == value
	case initLevel, initKeysFolder:
	}
	return false
}

func (h *harness) initEncrypted(t *testing.T, folder string) {
	t.Helper()
	h.chooseInit(t, initEncryption, encryptionNewKeys)
	h.focusInit(t, initKeysFolder)
	h.clearInput()
	h.typeText(folder)
	h.pressAndRun(key(tea.KeyEnter))
}

func TestConsoleInitForm(t *testing.T) {
	t.Parallel()

	t.Run(`Given the init form of a zstd repository
	When the user changes the compression to gzip with level 9
	Then the repository is initialized with them`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()
		h.chooseInit(t, initCompression, "gzip")
		h.focusInit(t, initLevel)
		h.typeText("9")

		h.pressAndRun(key(tea.KeyEnter))

		options := h.repositories.InitializeCalls()[0].Options
		require.Equal(t, "gzip", options.Compression())
		require.Equal(t, 9, *options.CompressionLevel())
		require.Equal(t, "gzip", h.model.current.Init().Compression())
	})

	t.Run(`Given the init form
	When the compression is xz or none
	Then there is no level field`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()

		h.chooseInit(t, initCompression, "xz")

		require.NotContains(t, h.initForm().visible(), initLevel)
		require.NotContains(t, h.view(), "empty for")
	})

	t.Run(`Given the init form with a level out of the range of the compression
	When the user submits it
	Then the form stays open with the error`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()
		h.focusInit(t, initLevel)
		h.typeText("30")

		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.initForm())
		require.ErrorIs(t, h.initForm().err, repository.ErrInvalidCompressionLevel)
		require.Empty(t, h.repositories.InitializeCalls())
	})

	t.Run(`Given the init form with a level that is not a number
	When the user submits it
	Then the form stays open with the error`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()
		h.focusInit(t, initLevel)
		h.typeText("fast")

		h.pressAndRun(key(tea.KeyEnter))

		require.ErrorIs(t, h.initForm().err, repository.ErrInvalidCompressionLevel)
		require.Contains(t, h.view(), "it must be a whole number")
		require.Empty(t, h.repositories.InitializeCalls())
	})

	t.Run(`Given the init form
	When the repository is initialized
	Then it uses the latest format version and the form does not offer another one`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()
		require.NotContains(t, h.view(), "Format")

		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, repository.LatestFormatVersion, h.repositories.InitializeCalls()[0].Options.FormatVersion())
	})
}

func TestConsoleEncryptedInit(t *testing.T) {
	t.Parallel()

	t.Run(`Given the init form of a repository without public key
	When the user chooses a new key pair
	Then it asks for the keys folder, prefilled with the default one`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()

		h.chooseInit(t, initEncryption, encryptionNewKeys)

		require.Contains(t, h.initForm().visible(), initKeysFolder)
		require.Equal(t, defaultKeysFolder(), h.initForm().keysFolder.Value())
		require.Empty(t, h.keys.GenerateCalls())
	})

	t.Run(`Given the init form with a new key pair
	When the user submits it with a folder
	Then a key pair is generated there, the repository is initialized with the chosen options and the session uses the keys`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()
		folder := filepath.Join(t.TempDir(), "keys")
		h.chooseInit(t, initCompression, "lz4")

		h.initEncrypted(t, folder)

		require.Equal(t, folder, h.keys.GenerateCalls()[0].Generation.Folder())
		options := h.repositories.InitializeCalls()[0].Options
		require.Equal(t, repository.LatestFormatVersion, options.FormatVersion())
		require.Equal(t, "lz4", options.Compression())
		require.Equal(t, filepath.Join(folder, "1234-public.pem"), *options.PublicKey())
		require.True(t, h.model.current.Encrypted())
		require.Equal(t, filepath.Join(folder, "1234-private.pem"), *h.model.current.PrivateKey())
		require.Equal(t, fingerprint, h.model.current.Fingerprint())
		require.Len(t, h.snapshots.ListCalls(), 1)
		require.Contains(t, h.model.message, "Repository default initialized encrypted")
	})

	t.Run(`Given the default repository initialized encrypted
	When the result is shown
	Then it lists the environment variables to set`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()

		h.initEncrypted(t, filepath.Join(t.TempDir(), "keys"))

		view := h.model.dialog.view()
		require.Contains(t, view, "STOWMARK_PRIVATE_KEY=")
		require.Contains(t, view, "STOWMARK_PUBLIC_KEY=")
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
	Then the encryption is the configured key and cannot be changed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false)
		keys := h.encrypted(t, 3, true)
		h.started()

		h.focusInit(t, initEncryption)
		h.pressAndRun(key(tea.KeyRight))
		require.Equal(t, encryptionConfigured, h.initForm().encryptionName())
		require.NotContains(t, h.view(), "◂")

		h.pressAndRun(key(tea.KeyEnter))

		require.Empty(t, h.keys.GenerateCalls())
		require.Equal(t, keys.publicKey, *h.repositories.InitializeCalls()[0].Options.PublicKey())
	})
}

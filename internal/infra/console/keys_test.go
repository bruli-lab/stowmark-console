package console

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/stretchr/testify/require"
)

const fingerprint = "SHA256:zrDYzwuHzX/QswVOFeE7LQBNvKhnrtMuqeR/X1V7n18"

type keyFiles struct {
	privateKey string
	publicKey  string
}

func writeKeyPair(t *testing.T, directory, id string) keyFiles {
	t.Helper()
	keys := keyFiles{
		privateKey: filepath.Join(directory, id+"-private.pem"),
		publicKey:  filepath.Join(directory, id+"-public.pem"),
	}
	require.NoError(t, os.WriteFile(keys.privateKey, []byte("-----BEGIN PRIVATE KEY-----\n"), 0o600))
	require.NoError(t, os.WriteFile(keys.publicKey, []byte("-----BEGIN PUBLIC KEY-----\n"), 0o600))

	return keys
}

func (h *harness) encrypted(t *testing.T, formatVersion int, withPrivateKey bool) keyFiles {
	t.Helper()
	keys := writeKeyPair(t, t.TempDir(), "1234")
	var privateKey *string
	if withPrivateKey {
		privateKey = &keys.privateKey
	}
	options := repository.NewInitOptions("zstd", nil, &keys.publicKey).WithFormatVersion(formatVersion)
	h.profiles[0] = repository.NewProfile("default", h.profiles[0].Location(), options).
		WithPrivateKey(privateKey).
		WithFingerprint(fingerprint)

	return keys
}

func (h *harness) command(name string) {
	h.pressAndRun(runes(":"))
	h.typeText(name)
	h.pressAndRun(key(tea.KeyEnter))
}

func (h *harness) wide() *harness {
	h.model.width = 260
	return h
}

func (h *harness) keyRequiredUntilKeyGiven() *harness {
	h.repositories.StatusFunc = func(context.Context) (repository.Status, error) {
		if len(h.catalog.UpdateCalls()) == 0 {
			return repository.StatusKeyRequired, nil
		}
		return repository.StatusReady, nil
	}
	return h
}

func (h *harness) info() *infoDialog {
	info, _ := h.model.dialog.(*infoDialog)
	return info
}

func (h *harness) replaceInput(value string) {
	h.clearInput()
	h.typeText(value)
}

func TestConsoleEncryptedRepository(t *testing.T) {
	t.Parallel()

	t.Run(`Given an encrypted repository with its private key
	When the console starts
	Then the header shows the lock and the top panel the private key and its fingerprint`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.encrypted(t, 3, true)

		h.started()

		view := h.view()
		require.Contains(t, view, lockBadge)
		require.Contains(t, view, "1234-private.pem")
		require.Contains(t, view, "SHA256:zrDYzwuHzX")
	})

	t.Run(`Given an encrypted repository
	When the console starts
	Then the top panel lists the key rotation commands`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.encrypted(t, 3, true)

		h.started()

		view := h.view()
		require.Contains(t, view, ":rewrap")
		require.Contains(t, view, ":rekey")
	})

	t.Run(`Given an encrypted repository configured without its private key
	When the console starts
	Then the top panel says that the private key is not set`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.encrypted(t, 2, false)

		h.started()

		require.Contains(t, h.view(), "not set")
	})

	t.Run(`Given an encrypted format version 3 repository without its private key
	When the console starts
	Then it explains which setting needs the key and asks for the private key file instead of offering to initialize it`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).keyRequiredUntilKeyGiven()

		h.started()

		require.NotNil(t, h.prompt())
		require.Equal(t, "Private key of default", h.prompt().title)
		require.Contains(t, h.view(), "STOWMARK_PRIVATE_KEY")
		require.Empty(t, h.snapshots.ListCalls())
		require.ErrorIs(t, h.model.err, encryption.ErrPrivateKeyRequired)
		require.ErrorContains(t, h.model.err, `repository "default" is encrypted and needs its private key`)
		require.ErrorContains(t, h.model.err, "STOWMARK_PRIVATE_KEY")
	})

	t.Run(`Given the private key prompt of a repository that needs its key
	When the user gives the private key file
	Then the session opens the repository with it, lists the snapshots and shows the setting to keep it`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).keyRequiredUntilKeyGiven()
		keys := writeKeyPair(t, t.TempDir(), "1234")
		h.started()
		h.clearInput()
		h.typeText(keys.privateKey)

		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, keys.privateKey, *h.catalog.UpdateCalls()[0].Profile.PrivateKey())
		require.Equal(t, keys.privateKey, *h.model.current.PrivateKey())
		require.NoError(t, h.model.err)
		require.Len(t, h.snapshots.ListCalls(), 1)
		require.NotNil(t, h.info())
		require.Contains(t, h.info().body, "STOWMARK_PRIVATE_KEY="+keys.privateKey)
	})

	t.Run(`Given the private key prompt of a repository of the repositories file
	When the user gives the private key file
	Then it shows the private_key line to add to its entry`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).keyRequiredUntilKeyGiven()
		h.profiles[0] = repository.NewProfile("vault", h.profiles[0].Location(), h.profiles[0].Init())
		keys := writeKeyPair(t, t.TempDir(), "1234")
		h.started()
		h.clearInput()
		h.typeText(keys.privateKey)

		h.pressAndRun(key(tea.KeyEnter))

		require.Contains(t, h.info().body, "private_key: "+keys.privateKey)
		require.NotContains(t, h.info().body, "STOWMARK_PRIVATE_KEY")
	})

	t.Run(`Given the private key prompt
	When the user gives a key that does not open the repository
	Then the error names the key and the prompt asks again`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.repositories.StatusFunc = func(context.Context) (repository.Status, error) {
			if len(h.catalog.UpdateCalls()) == 0 {
				return repository.StatusKeyRequired, nil
			}
			return repository.StatusUninitialized, encryption.ErrWrongPrivateKey
		}
		keys := writeKeyPair(t, t.TempDir(), "1234")
		h.started()
		h.clearInput()
		h.typeText(keys.privateKey)

		h.pressAndRun(key(tea.KeyEnter))

		require.ErrorIs(t, h.model.err, encryption.ErrWrongPrivateKey)
		require.ErrorContains(t, h.model.err, keys.privateKey)
		require.NotNil(t, h.prompt())
		require.Contains(t, h.prompt().err, "does not open repository")
		require.Empty(t, h.snapshots.ListCalls())
	})

	t.Run(`Given the private key prompt
	When the user cancels it
	Then the explanation of the setting to change stays`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).keyRequiredUntilKeyGiven()
		h.started()

		h.pressAndRun(key(tea.KeyEscape))

		require.Nil(t, h.model.dialog)
		require.ErrorIs(t, h.model.err, encryption.ErrPrivateKeyRequired)
		require.Empty(t, h.catalog.UpdateCalls())
	})

	t.Run(`Given several repositories and one that needs its private key
	When it is opened
	Then the repositories list is shown again with the explanation naming the private_key field`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", t.TempDir()).started()
		h.repositories.StatusFunc = func(context.Context) (repository.Status, error) {
			return repository.StatusKeyRequired, nil
		}
		h.pressAndRun(runes("j"))

		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.repositoriesScreen())
		require.ErrorContains(t, h.model.err, `set private_key for "nas" in repositories.yaml`)
	})

	t.Run(`Given an encrypted repository opened with another private key
	When the snapshots are loaded
	Then the error names the private key that does not open it`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		keys := h.encrypted(t, 3, true)
		h.snapshots.ListFunc = func(context.Context) ([]snapshot.Snapshot, error) {
			return nil, encryption.ErrWrongPrivateKey
		}

		h.started()

		require.ErrorIs(t, h.model.err, encryption.ErrWrongPrivateKey)
		require.ErrorContains(t, h.model.err, "the private key "+keys.privateKey+` does not open repository "default"`)
	})

	t.Run(`Given an encrypted and an unencrypted repository
	When the repositories are listed
	Then only the encrypted one has the lock`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", t.TempDir())
		h.encrypted(t, 3, true)

		h.started()

		rows := h.repositoriesScreen().table.Rows()
		require.Equal(t, lockBadge, rows[0][2])
		require.Empty(t, rows[1][2])
	})

	t.Run(`Given a repository configured with a public key that is not initialized
	When it is initialized
	Then it uses the configured key and the latest format version`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false)
		keys := h.encrypted(t, 3, true)
		h.started()

		h.pressAndRun(key(tea.KeyEnter))

		options := h.repositories.InitializeCalls()[0].Options
		require.Equal(t, keys.publicKey, *options.PublicKey())
		require.Equal(t, repository.LatestFormatVersion, options.FormatVersion())
	})
}

func TestConsoleKeyGeneration(t *testing.T) {
	t.Parallel()

	t.Run(`Given an open repository
	When the user generates a key pair in a folder
	Then the keys are generated there and the configuration lines are shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).wide().started()
		folder := filepath.Join(t.TempDir(), "keys")

		h.command("keygen")
		require.Equal(t, defaultKeysFolder(), h.prompt().input.Value())
		h.replaceInput(folder)
		require.Contains(t, h.view(), "The directory will be created")
		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, folder, h.keys.GenerateCalls()[0].Generation.Folder())
		require.NotNil(t, h.info())
		view := h.view()
		require.Contains(t, view, "public_key: "+filepath.Join(folder, "1234-public.pem"))
		require.Contains(t, view, "private_key: "+filepath.Join(folder, "1234-private.pem"))

		h.pressAndRun(key(tea.KeyEsc))
		require.Nil(t, h.model.dialog)
	})

	t.Run(`Given a file instead of a folder
	When the user generates a key pair there
	Then the form refuses it`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		file := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(file, nil, 0o600))

		h.command("keygen")
		h.replaceInput(file)
		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.prompt())
		require.Equal(t, errNotADirectory.Error(), h.prompt().err)
		require.Empty(t, h.keys.GenerateCalls())
	})

	t.Run(`Given several repositories and none open
	When the user asks for a key pair
	Then the console asks to choose a repository first`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", t.TempDir()).started()

		h.command("keygen")

		require.Nil(t, h.model.dialog)
		require.Equal(t, "Choose a repository first", h.model.message)
	})
}

func TestConsoleKeyRewrap(t *testing.T) {
	t.Parallel()

	t.Run(`Given an unencrypted repository
	When the user asks to rewrap its key
	Then the console says it is not encrypted`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		h.command("rewrap")

		require.Nil(t, h.model.dialog)
		require.Equal(t, "Repository default is not encrypted", h.model.message)
	})

	t.Run(`Given an encrypted repository and a new key pair
	When the user rewraps the key and confirms
	Then the keys reach the use case and the configuration change is shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).wide()
		current := h.encrypted(t, 3, true)
		next := writeKeyPair(t, filepath.Dir(current.privateKey), "5678")
		h.started()

		h.command("rewrap")
		require.Equal(t, current.privateKey, h.prompt().input.Value())
		h.pressAndRun(key(tea.KeyEnter))
		h.typeText(next.publicKey)
		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.confirmation())
		require.Contains(t, h.view(), "exclusive lock")
		h.pressAndRun(runes("y"))

		call := h.keys.RewrapCalls()[0].Rewrap
		require.Equal(t, current.privateKey, call.OldPrivateKey())
		require.Equal(t, next.publicKey, call.NewPublicKey())
		require.NotNil(t, h.info())
		require.Contains(t, h.view(), "public_key: "+next.publicKey)
		require.Contains(t, h.view(), "private_key: "+next.privateKey)
	})

	t.Run(`Given an encrypted repository whose key was already rewrapped
	When the user rewraps it with the old private key
	Then the error explains that the private key does not open it`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		current := h.encrypted(t, 3, true)
		h.keys.RewrapFunc = func(context.Context, *encryption.Rewrap) error { return encryption.ErrWrongPrivateKey }
		h.started()

		h.command("rewrap")
		h.pressAndRun(key(tea.KeyEnter))
		h.typeText(current.publicKey)
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("y"))

		require.Nil(t, h.model.dialog)
		require.ErrorIs(t, h.model.err, encryption.ErrWrongPrivateKey)
		require.ErrorContains(t, h.model.err, "failed to rewrap the key of default: the private key "+current.privateKey)
	})

	t.Run(`Given the rewrap form
	When the user enters a key file that does not exist
	Then the form refuses it`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.encrypted(t, 3, true)
		h.started()

		h.command("rewrap")
		h.replaceInput(filepath.Join(t.TempDir(), "missing.pem"))
		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, errFileNotFound.Error(), h.prompt().err)
	})

	t.Run(`Given the rewrap confirmation
	When the user rejects it
	Then nothing is rewrapped`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		current := h.encrypted(t, 3, true)
		next := writeKeyPair(t, filepath.Dir(current.privateKey), "5678")
		h.started()

		h.command("rewrap")
		h.pressAndRun(key(tea.KeyEnter))
		h.typeText(next.publicKey)
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("n"))

		require.Empty(t, h.keys.RewrapCalls())
		require.Equal(t, "Key rewrap cancelled", h.model.message)
	})
}

func TestConsoleKeyRotation(t *testing.T) {
	t.Parallel()

	t.Run(`Given an encrypted repository
	When the user rotates its key with the configured pair and types its name
	Then the key is rotated and no configuration change is needed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		current := h.encrypted(t, 3, true)
		h.started()

		h.command("rekey")
		h.pressAndRun(key(tea.KeyEnter))
		require.Equal(t, current.publicKey, h.prompt().input.Value())
		h.pressAndRun(key(tea.KeyEnter))
		h.typeText("other")
		h.pressAndRun(key(tea.KeyEnter))
		require.Contains(t, h.prompt().err, "type default")
		h.replaceInput("default")
		h.pressAndRun(key(tea.KeyEnter))

		call := h.keys.RotateCalls()[0].Rotation
		require.Equal(t, current.privateKey, call.PrivateKey())
		require.Equal(t, current.publicKey, call.PublicKey())
		require.Nil(t, h.model.dialog)
		require.Equal(t, "Key of default rotated: the repository uses a new key generation", h.model.message)
	})

	t.Run(`Given an encrypted repository configured with its private key only
	When the user rotates its key
	Then the public key of the same pair is proposed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		current := h.encrypted(t, 3, true)
		privateKey := current.privateKey
		h.profiles[0] = repository.NewProfile("default", h.profiles[0].Location(), repository.NewInitOptions("", nil, nil)).
			WithPrivateKey(&privateKey)
		h.started()

		h.command("rekey")
		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, current.publicKey, h.prompt().input.Value())
	})

	t.Run(`Given an encrypted repository
	When the user rotates its key to another pair
	Then the configuration change is shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).wide()
		current := h.encrypted(t, 3, true)
		next := writeKeyPair(t, filepath.Dir(current.privateKey), "5678")
		h.started()

		h.command("rekey")
		h.pressAndRun(key(tea.KeyEnter))
		h.replaceInput(next.publicKey)
		h.pressAndRun(key(tea.KeyEnter))
		h.typeText("default")
		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.info())
		require.Contains(t, h.view(), "private_key: "+next.privateKey)
	})

	t.Run(`Given a running key rotation
	When the user presses Esc
	Then it is cancelled`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.encrypted(t, 3, true)
		h.keys.RotateFunc = func(ctx context.Context, _ *encryption.Rotation) error { return ctx.Err() }
		h.started()
		h.command("rekey")
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(key(tea.KeyEnter))
		h.typeText("default")

		operation := h.start(key(tea.KeyEnter))
		require.True(t, h.model.loading)
		h.pressAndRun(key(tea.KeyEsc))
		h.run(operation)

		require.False(t, h.model.loading)
		require.Equal(t, "Cancelled: rotate the key of default", h.model.message)
	})
}

func TestKeyFileCompletion(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	keys := writeKeyPair(t, directory, "1234")
	require.NoError(t, os.Mkdir(filepath.Join(directory, "old"), 0o700))
	notes := filepath.Join(directory, "notes.txt")
	require.NoError(t, os.WriteFile(notes, []byte("hello"), 0o600))

	t.Run(`Given a folder with keys, other files and folders
	When key files are suggested
	Then only the PEM files and the folders are offered`, func(t *testing.T) {
		t.Parallel()

		suggestions := keyFileSuggestions(directory + "/")

		require.ElementsMatch(t, []string{keys.privateKey, keys.publicKey, filepath.Join(directory, "old") + "/"}, suggestions)
	})

	t.Run(`Given a file that is not a PEM key
	When it is described as a key file
	Then it is refused`, func(t *testing.T) {
		t.Parallel()

		_, err := describeKeyFile(notes)

		require.ErrorIs(t, err, errNotAPEMKey)
	})

	t.Run(`Given a private key whose public key is next to it
	When its pair is looked for
	Then the public key is found, and nothing for a key named differently`, func(t *testing.T) {
		t.Parallel()

		require.Equal(t, keys.publicKey, pairedKey(keys.privateKey, privateKeySuffix, publicKeySuffix))
		require.Empty(t, pairedKey(notes, privateKeySuffix, publicKeySuffix))
	})
}

func TestExplainKeyError(t *testing.T) {
	t.Parallel()

	privateKey := "/keys/1234-private.pem"
	profile := repository.NewProfile("nas", "/backups", repository.NewInitOptions("", nil, nil)).WithPrivateKey(&privateKey)

	tests := map[error]string{
		encryption.ErrPrivateKeyUnreadable: `the private key of repository "nas" cannot be read`,
		encryption.ErrNotEncrypted:         `repository "nas" is not encrypted`,
		errUnreachable:                     "unreachable",
	}

	for cause, expected := range tests {
		t.Run(`Given the error "`+cause.Error()+`"
		When it is explained
		Then the message names the repository and keeps the cause`, func(t *testing.T) {
			t.Parallel()

			explained := explainKeyError(profile, cause)

			require.ErrorIs(t, explained, cause)
			require.Contains(t, explained.Error(), expected)
		})
	}
}

func TestConsoleKeyRewrapToConfiguration(t *testing.T) {
	t.Parallel()

	t.Run(`Given an encrypted repository rewrapped earlier to another key
	When the user rewraps it back to the configured key pair
	Then no configuration change is asked`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		current := h.encrypted(t, 3, true)
		other := writeKeyPair(t, filepath.Dir(current.privateKey), "5678")
		h.started()

		h.command("rewrap")
		h.replaceInput(other.privateKey)
		h.pressAndRun(key(tea.KeyEnter))
		h.typeText(current.publicKey)
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("y"))

		require.Equal(t, other.privateKey, h.keys.RewrapCalls()[0].Rewrap.OldPrivateKey())
		require.Nil(t, h.model.dialog)
		require.Equal(t, "Key of default rewrapped to the configured key pair", h.model.message)
	})
}

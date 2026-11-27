package store_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
	"github.com/stretchr/testify/require"
)

type encryptedRepository struct {
	location   string
	source     string
	snapshotID string
	keys       encryption.KeyPair
}

func newEncryptedRepository(t *testing.T, formatVersion int) encryptedRepository {
	t.Helper()
	setup := encryptedSetup(t, formatVersion)
	keys, err := encryption.NewKeyPair(*setup.privateKey, *setup.publicKey)
	require.NoError(t, err)
	location := filepath.Join(t.TempDir(), "repository")
	source := writeSource(t)

	session, err := store.Open(t.Context(), location, setup.privateKey)
	require.NoError(t, err)
	defer func() { require.NoError(t, session.Close()) }()

	require.NoError(t, store.NewRepositories(session).Initialize(t.Context(), setup.initOptions()))
	created, err := store.NewSnapshots(session).Create(t.Context(), creation(t, source))
	require.NoError(t, err)

	return encryptedRepository{location: location, source: source, snapshotID: created.ID(), keys: keys}
}

func openSession(t *testing.T, location string, privateKey *string) *store.Session {
	t.Helper()
	session, err := store.Open(t.Context(), location, privateKey)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, session.Close()) })

	return session
}

func configFingerprint(t *testing.T, location string) (fingerprint string, generation int) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(location, "config.json"))
	require.NoError(t, err)

	var conf struct {
		Encryption struct {
			Fingerprint string `json:"public_key_fingerprint"`
			Generation  int    `json:"generation"`
		} `json:"encryption"`
	}
	require.NoError(t, json.Unmarshal(content, &conf))

	return conf.Encryption.Fingerprint, conf.Encryption.Generation
}

func TestStoreKeyProblems(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")
	v2 := newEncryptedRepository(t, 2)
	v3 := newEncryptedRepository(t, 3)
	other := generateKeys(t).PrivateKey()
	missing := filepath.Join(t.TempDir(), "missing-private.pem")

	t.Run(`Given a repository encrypted with format version 3
	When its status is checked without a private key
	Then it is initialized but needs the key, and reading it says so`, func(t *testing.T) {
		session := openSession(t, v3.location, nil)

		status, err := store.NewRepositories(session).Status(t.Context())
		require.NoError(t, err)
		require.Equal(t, repository.StatusKeyRequired, status)

		_, err = store.NewSnapshots(session).List(t.Context())
		require.ErrorIs(t, err, encryption.ErrPrivateKeyRequired)
	})

	for name, location := range map[string]string{"2": v2.location, "3": v3.location} {
		t.Run(`Given a repository encrypted with format version `+name+`
		When its status is checked with another private key
		Then a wrong private key error is returned`, func(t *testing.T) {
			_, err := store.NewRepositories(openSession(t, location, &other)).Status(t.Context())

			require.ErrorIs(t, err, encryption.ErrWrongPrivateKey)
		})

		t.Run(`Given a repository encrypted with format version `+name+`
		When its status is checked with a private key file that does not exist
		Then an unreadable private key error is returned instead of a missing repository`, func(t *testing.T) {
			_, err := store.NewRepositories(openSession(t, location, &missing)).Status(t.Context())

			require.ErrorIs(t, err, encryption.ErrPrivateKeyUnreadable)
		})
	}

	t.Run(`Given an unencrypted repository
	When its status is checked with a private key
	Then a not encrypted error is returned`, func(t *testing.T) {
		location := filepath.Join(t.TempDir(), "plain")
		session := openSession(t, location, nil)
		require.NoError(t, store.NewRepositories(session).Initialize(t.Context(), repository.NewInitOptions("", nil, nil)))

		_, err := store.NewRepositories(openSession(t, location, &other)).Status(t.Context())

		require.ErrorIs(t, err, encryption.ErrNotEncrypted)
	})
}

func TestStoreCatalogWithEncryptedRepository(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")
	v2 := newEncryptedRepository(t, 2)
	publicKey := v2.keys.PublicKey()
	privateKey := v2.keys.PrivateKey()
	session := store.NewSession()
	t.Cleanup(func() { require.NoError(t, session.Close()) })
	options := repository.NewInitOptions("", nil, &publicKey)
	catalog := store.NewCatalog(session, []store.Entry{
		{Profile: repository.NewProfile("public-only", v2.location, options)},
		{Profile: repository.NewProfile("private-only", v2.location, repository.NewInitOptions("", nil, nil)).WithPrivateKey(&privateKey)},
	})
	expected, _ := configFingerprint(t, v2.location)

	t.Run(`Given an encrypted repository configured with its public key only
	When it is opened
	Then the profile carries the fingerprint Stowmark stored and creating a snapshot is refused`, func(t *testing.T) {
		profile, err := catalog.Open(t.Context(), "public-only")
		require.NoError(t, err)
		require.Equal(t, expected, profile.Fingerprint())

		_, err = store.NewSnapshots(session).Create(t.Context(), creation(t, v2.source))

		require.ErrorIs(t, err, encryption.ErrPrivateKeyRequired)
	})

	t.Run(`Given an encrypted repository configured with its private key only
	When it is opened
	Then the fingerprint is derived from the private key and snapshots can be created`, func(t *testing.T) {
		profile, err := catalog.Open(t.Context(), "private-only")
		require.NoError(t, err)
		require.Equal(t, expected, profile.Fingerprint())

		_, err = store.NewSnapshots(session).Create(t.Context(), creation(t, v2.source))

		require.NoError(t, err)
	})
}

func TestStoreKeys(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")

	t.Run(`Given a folder
	When a key pair is generated
	Then the private and public key files exist inside it`, func(t *testing.T) {
		keys := generateKeys(t)

		require.FileExists(t, keys.PrivateKey())
		require.FileExists(t, keys.PublicKey())
		require.Equal(t, filepath.Dir(keys.PrivateKey()), filepath.Dir(keys.PublicKey()))
	})

	t.Run(`Given an encrypted repository and a new key pair
	When the key is rewrapped to the new public key
	Then the old private key no longer opens it and the new one reads its files`, func(t *testing.T) {
		repo := newEncryptedRepository(t, 3)
		oldPrivate := repo.keys.PrivateKey()
		next := generateKeys(t)
		newPrivate := next.PrivateKey()
		rewrap, err := encryption.NewRewrap(oldPrivate, next.PublicKey())
		require.NoError(t, err)

		require.NoError(t, store.NewKeys(openSession(t, repo.location, &oldPrivate)).Rewrap(t.Context(), rewrap))

		_, err = store.NewSnapshots(openSession(t, repo.location, &oldPrivate)).List(t.Context())
		require.ErrorIs(t, err, encryption.ErrWrongPrivateKey)
		content, err := store.NewSnapshots(openSession(t, repo.location, &newPrivate)).
			FileContent(t.Context(), repo.snapshotID, filepath.Join(repo.source, "README.txt"), 100)
		require.NoError(t, err)
		require.Equal(t, "hello stowmark", string(content))
	})

	t.Run(`Given an encrypted repository
	When the key is rotated with the same key pair
	Then a new key generation is stored and its files are still readable`, func(t *testing.T) {
		repo := newEncryptedRepository(t, 3)
		privateKey := repo.keys.PrivateKey()
		rotation, err := encryption.NewRotation(privateKey, repo.keys.PublicKey())
		require.NoError(t, err)
		session := openSession(t, repo.location, &privateKey)

		require.NoError(t, store.NewKeys(session).Rotate(t.Context(), rotation))

		_, generation := configFingerprint(t, repo.location)
		require.Equal(t, 2, generation)
		content, err := store.NewSnapshots(openSession(t, repo.location, &privateKey)).
			FileContent(t.Context(), repo.snapshotID, filepath.Join(repo.source, "notes", "todo.md"), 100)
		require.NoError(t, err)
		require.Equal(t, "- verify\n- restore\n", string(content))
	})

	t.Run(`Given an unencrypted repository
	When its key is rewrapped
	Then a not encrypted error is returned`, func(t *testing.T) {
		location := filepath.Join(t.TempDir(), "plain")
		session := openSession(t, location, nil)
		require.NoError(t, store.NewRepositories(session).Initialize(t.Context(), repository.NewInitOptions("", nil, nil)))
		keys := generateKeys(t)
		rewrap, err := encryption.NewRewrap(keys.PrivateKey(), keys.PublicKey())
		require.NoError(t, err)

		require.ErrorIs(t, store.NewKeys(session).Rewrap(t.Context(), rewrap), encryption.ErrNotEncrypted)
	})

	t.Run(`Given a session without an open repository
	When keys are generated, rewrapped or rotated
	Then a not connected error is returned`, func(t *testing.T) {
		keys := store.NewKeys(store.NewSession())
		generation, err := encryption.NewKeyGeneration(t.TempDir())
		require.NoError(t, err)
		rotation, err := encryption.NewRotation("/keys/private.pem", "/keys/public.pem")
		require.NoError(t, err)
		rewrap, err := encryption.NewRewrap("/keys/private.pem", "/keys/public.pem")
		require.NoError(t, err)

		_, err = keys.Generate(t.Context(), generation)
		require.ErrorIs(t, err, store.ErrNotConnected)
		require.ErrorIs(t, keys.Rotate(t.Context(), rotation), store.ErrNotConnected)
		require.ErrorIs(t, keys.Rewrap(t.Context(), rewrap), store.ErrNotConnected)
	})
}

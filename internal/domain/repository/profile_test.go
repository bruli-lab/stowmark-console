package repository_test

import (
	"testing"
	"time"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/stretchr/testify/require"
)

func TestProfileKind(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"/home/user/backups":                    "local",
		"relative/backups":                      "local",
		"ssh://stowmark@localhost:12222/backup": "ssh",
		"S3://bucket/backups":                   "s3",
		"webdavs://dav.example.com/backups":     "webdavs",
		"gcs://bucket/backups":                  "gcs",
	}

	for location, expected := range tests {
		t.Run(`Given a profile located at "`+location+`"
		When its kind is asked
		Then it is "`+expected+`"`, func(t *testing.T) {
			t.Parallel()

			profile := repository.NewProfile("name", location, repository.NewInitOptions("", nil, nil))

			require.Equal(t, expected, profile.Kind())
			require.Equal(t, expected == "local", profile.Local())
		})
	}
}

func TestProfileEncryption(t *testing.T) {
	t.Parallel()

	privateKey := "/keys/private.pem"
	publicKey := "/keys/public.pem"

	tests := map[string]struct {
		privateKey, publicKey *string
		encrypted, missingKey bool
	}{
		"no keys":                   {},
		"only a private key":        {privateKey: &privateKey, encrypted: true},
		"only a public key":         {publicKey: &publicKey, encrypted: true, missingKey: true},
		"a private and public keys": {privateKey: &privateKey, publicKey: &publicKey, encrypted: true},
	}

	for name, tt := range tests {
		t.Run(`Given a profile with `+name+`
		When its encryption is asked
		Then it reports whether it is encrypted and whether the private key is missing`, func(t *testing.T) {
			t.Parallel()

			profile := repository.NewProfile("name", "/backups", repository.NewInitOptions("", nil, tt.publicKey)).
				WithPrivateKey(tt.privateKey).
				WithFingerprint("SHA256:abc")

			require.Equal(t, tt.encrypted, profile.Encrypted())
			require.Equal(t, tt.missingKey, profile.MissingPrivateKey())
			require.Equal(t, tt.privateKey, profile.PrivateKey())
			require.Equal(t, "SHA256:abc", profile.Fingerprint())
		})
	}
}

func TestInitOptionsFormatVersion(t *testing.T) {
	t.Parallel()

	publicKey := "/keys/public.pem"

	t.Run(`Given init options without a format version
	When the format version is asked
	Then it is the Stowmark default`, func(t *testing.T) {
		t.Parallel()

		options := repository.NewInitOptions("", nil, &publicKey)

		require.Equal(t, repository.DefaultFormatVersion, options.FormatVersion())
		require.False(t, options.EncryptsMetadata())
	})

	t.Run(`Given encrypted init options with format version 3
	When they are asked whether metadata is encrypted
	Then they are`, func(t *testing.T) {
		t.Parallel()

		options := repository.NewInitOptions("", nil, &publicKey).WithFormatVersion(3)

		require.Equal(t, 3, options.FormatVersion())
		require.True(t, options.EncryptsMetadata())
	})

	t.Run(`Given unencrypted init options with format version 3
	When they are asked whether metadata is encrypted
	Then they are not`, func(t *testing.T) {
		t.Parallel()

		require.False(t, repository.NewInitOptions("", nil, nil).WithFormatVersion(3).EncryptsMetadata())
	})
}

func TestProfileRetention(t *testing.T) {
	t.Parallel()

	t.Run(`Given a profile with retention rules
	When its retention is asked
	Then it returns the rules`, func(t *testing.T) {
		t.Parallel()
		rules := retention.Rules{Last: 7, Within: time.Hour}

		profile := repository.NewProfile("nas", "/backups", repository.NewInitOptions("", nil, nil)).WithRetention(&rules)

		configured, ok := profile.Retention()
		require.True(t, ok)
		require.Equal(t, rules, configured)
	})

	t.Run(`Given an unencrypted profile
	When it gets a key pair
	Then it is encrypted, keeps its other init options and has the private key`, func(t *testing.T) {
		t.Parallel()
		options := repository.NewInitOptions("zstd", nil, nil).WithFormatVersion(repository.LatestFormatVersion)
		base := repository.NewProfile("nas", "/backups", repository.NewInitOptions("", nil, nil)).WithInit(options)

		profile := base.WithKeys("/keys/a-public.pem", "/keys/a-private.pem")

		require.True(t, profile.Encrypted())
		require.False(t, profile.MissingPrivateKey())
		require.True(t, profile.Init().EncryptsMetadata())
		require.Equal(t, "zstd", profile.Init().Compression())
		require.Equal(t, "/keys/a-private.pem", *profile.PrivateKey())
		require.False(t, base.Encrypted())
	})

	t.Run(`Given a profile with retention rules and exclusions
	When both are asked
	Then each keeps its value`, func(t *testing.T) {
		t.Parallel()
		rules := retention.Rules{Last: 3}
		exclusions := snapshot.NewExclusions([]string{"*.tmp"}, true)
		base := repository.NewProfile("nas", "/backups", repository.NewInitOptions("", nil, nil))

		profile := base.WithExclusions(exclusions).WithRetention(&rules)

		configured, ok := profile.Retention()
		require.True(t, ok)
		require.Equal(t, rules, configured)
		require.Equal(t, exclusions, profile.Exclusions())
		require.True(t, base.Exclusions().Empty())
	})

	t.Run(`Given a profile without retention rules
	When its retention is asked
	Then it reports that there are none`, func(t *testing.T) {
		t.Parallel()

		_, ok := repository.NewProfile("nas", "/backups", repository.NewInitOptions("", nil, nil)).Retention()

		require.False(t, ok)
	})
}

package repository_test

import (
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
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

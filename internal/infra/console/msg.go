package console

import (
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type repositoryStatusMsg struct {
	initialized bool
	err         error
}

type repositoryInitializedMsg struct {
	event repository.Initialized
	err   error
}

type snapshotsLoadedMsg struct {
	snapshots []snapshot.Snapshot
	selectID  string
	err       error
}

type manifestLoadedMsg struct {
	manifest snapshot.Manifest
	err      error
}

type fileContentLoadedMsg struct {
	name      string
	content   []byte
	truncated bool
	err       error
}

type snapshotCreatedMsg struct {
	event snapshot.Created
	err   error
}

type snapshotVerifiedMsg struct {
	snapshotID string
	event      snapshot.Verified
	err        error
}

type snapshotRestoredMsg struct {
	snapshotID string
	event      snapshot.Restored
	err        error
}

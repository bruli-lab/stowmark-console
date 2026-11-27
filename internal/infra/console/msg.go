package console

import (
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type repositoriesLoadedMsg struct {
	profiles []repository.Profile
	err      error
}

type repositoryOpenedMsg struct {
	name  string
	event repository.Opened
	err   error
}

type repositoryStatusMsg struct {
	status repository.Status
	err    error
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
	request   filePreviewRequested
	content   previewContent
	truncated bool
	err       error
}

type fileOpenedMsg struct {
	name string
	err  error
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

type keysGeneratedMsg struct {
	event encryption.KeysGenerated
	err   error
}

type keyRewrappedMsg struct {
	request rewrapRequest
	event   encryption.KeyRewrapped
	err     error
}

type keyRotatedMsg struct {
	request rotationRequest
	event   encryption.KeyRotated
	err     error
}

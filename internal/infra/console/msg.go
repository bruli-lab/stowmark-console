package console

import (
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

type repositoryLoadedMsg struct {
	entries []repository.Entry
	err     error
	path    string
}

type filePreviewLoadedMsg struct {
	path    string
	content string
	err     error
}

type snapshotVerifiedMsg struct {
	snapshotID string
	err        error
}

type snapshotRestoredMsg struct {
	snapshotID  string
	destination string
	err         error
}

type snapshotCreatedMsg struct {
	snapshotID string
	fileCount  int
	totalSize  int64
	err        error
}

type initRepositoryMsg struct {
	err error
}

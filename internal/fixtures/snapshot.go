package fixtures

import (
	"time"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type SnapshotBuilder struct {
	id        string
	createdAt time.Time
	fileCount int
	totalSize int64
	source    string
}

func Snapshot() *SnapshotBuilder {
	return &SnapshotBuilder{
		id:        "20261126T160034Z-f482b24d",
		createdAt: time.Date(2026, 11, 26, 16, 0, 34, 0, time.UTC),
		fileCount: 3,
		totalSize: 2048,
		source:    "/home/user/documents",
	}
}

func (b *SnapshotBuilder) WithID(id string) *SnapshotBuilder {
	b.id = id
	return b
}

func (b *SnapshotBuilder) WithCreatedAt(createdAt time.Time) *SnapshotBuilder {
	b.createdAt = createdAt
	return b
}

func (b *SnapshotBuilder) WithSource(source string) *SnapshotBuilder {
	b.source = source
	return b
}

func (b *SnapshotBuilder) Build() snapshot.Snapshot {
	return snapshot.New(b.id, b.createdAt, b.fileCount, b.totalSize, b.source)
}

type ManifestBuilder struct {
	snapshotID string
	source     string
	files      []snapshot.File
}

func Manifest() *ManifestBuilder {
	return &ManifestBuilder{
		snapshotID: "20261126T160034Z-f482b24d",
		source:     "/home/user/documents",
		files: []snapshot.File{
			snapshot.NewFile("/home/user/documents/README.txt", 100),
			snapshot.NewFile("/home/user/documents/notes/todo.md", 200),
			snapshot.NewFile("/home/user/documents/notes/2026/diary.txt", 300),
		},
	}
}

func (b *ManifestBuilder) WithSnapshotID(snapshotID string) *ManifestBuilder {
	b.snapshotID = snapshotID
	return b
}

func (b *ManifestBuilder) WithSource(source string) *ManifestBuilder {
	b.source = source
	return b
}

func (b *ManifestBuilder) WithFiles(files ...snapshot.File) *ManifestBuilder {
	b.files = files
	return b
}

func (b *ManifestBuilder) Build() snapshot.Manifest {
	return snapshot.NewManifest(b.snapshotID, b.source, b.files)
}

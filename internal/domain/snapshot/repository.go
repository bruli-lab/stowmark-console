package snapshot

import "context"

type Repository interface {
	List(ctx context.Context) ([]Snapshot, error)
	Manifest(ctx context.Context, snapshotID string) (Manifest, error)
	Create(ctx context.Context, source string) (Snapshot, error)
	Verify(ctx context.Context, snapshotID string) ([]Failure, error)
	Restore(ctx context.Context, snapshotID, destination string) ([]Failure, error)
	FileContent(ctx context.Context, snapshotID, path string, maxSize int64) ([]byte, error)
	ExtractFile(ctx context.Context, extraction *FileExtraction) (string, error)
}

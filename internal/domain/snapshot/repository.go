package snapshot

import "context"

type Repository interface {
	List(ctx context.Context) ([]Snapshot, error)
	Manifest(ctx context.Context, snapshotID string) (Manifest, error)
	Create(ctx context.Context, creation *Creation) (Snapshot, error)
	Verify(ctx context.Context, verification *Verification) ([]Failure, error)
	Restore(ctx context.Context, restoration *Restoration) ([]Failure, error)
	FileContent(ctx context.Context, snapshotID, path string, maxSize int64) ([]byte, error)
	ExtractFile(ctx context.Context, extraction *FileExtraction) (string, error)
}

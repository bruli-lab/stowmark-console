package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/bruli-lab/stowmark/pkg/stowmark"
)

var ErrRestoredFileNotFound = errors.New("restored file not found")

type Snapshots struct {
	session *Session
}

func NewSnapshots(session *Session) *Snapshots {
	return &Snapshots{session: session}
}

func (s *Snapshots) List(ctx context.Context) ([]snapshot.Snapshot, error) {
	handler, err := s.session.current()
	if err != nil {
		return nil, err
	}

	summaries, err := handler.ListSnapshots(ctx)
	if err != nil {
		return nil, err
	}

	snapshots := make([]snapshot.Snapshot, len(summaries))
	for i := range summaries {
		summary := &summaries[i]
		snapshots[i] = snapshot.New(summary.ID, summary.CreatedAt, summary.FileCount, summary.TotalSize, summary.Source)
	}

	return snapshots, nil
}

func (s *Snapshots) Manifest(ctx context.Context, snapshotID string) (snapshot.Manifest, error) {
	handler, err := s.session.current()
	if err != nil {
		return snapshot.Manifest{}, err
	}

	manifest, err := handler.GetManifest(ctx, snapshotID)
	if err != nil {
		return snapshot.Manifest{}, err
	}

	files := make([]snapshot.File, len(manifest.Files))
	for i := range manifest.Files {
		files[i] = snapshot.NewFile(manifest.Files[i].Path, manifest.Files[i].Size)
	}

	return snapshot.NewManifest(manifest.ID, manifest.Source, files), nil
}

func (s *Snapshots) Create(ctx context.Context, creation *snapshot.Creation) (snapshot.Snapshot, error) {
	handler, err := s.session.current()
	if err != nil {
		return snapshot.Snapshot{}, err
	}

	result, err := handler.CreateSnapshot(ctx, creation.Source(), nil)
	if err != nil {
		return snapshot.Snapshot{}, err
	}

	manifest, err := handler.GetManifest(ctx, result.ID)
	if err != nil {
		return snapshot.Snapshot{}, err
	}

	var totalSize int64
	for i := range manifest.Files {
		totalSize += manifest.Files[i].Size
	}

	return snapshot.New(result.ID, manifest.CreatedAt, len(manifest.Files), totalSize, manifest.Source), nil
}

func (s *Snapshots) Verify(ctx context.Context, verification *snapshot.Verification) ([]snapshot.Failure, error) {
	handler, err := s.session.current()
	if err != nil {
		return nil, err
	}

	result, err := handler.VerifySnapshot(ctx, verification.SnapshotID(), nil)
	if err != nil {
		return nil, err
	}

	return failures(result), nil
}

func (s *Snapshots) Restore(ctx context.Context, restoration *snapshot.Restoration) ([]snapshot.Failure, error) {
	handler, err := s.session.current()
	if err != nil {
		return nil, err
	}

	destination := restoration.Destination()

	result, err := handler.RestoreSnapshot(ctx, restoration.SnapshotID(), &destination, nil)
	if err != nil {
		return nil, err
	}

	return failures(result), nil
}

func (s *Snapshots) FileContent(ctx context.Context, snapshotID, path string, maxSize int64) ([]byte, error) {
	handler, err := s.session.current()
	if err != nil {
		return nil, err
	}

	directory, err := os.MkdirTemp("", "stowmark-console-preview-*")
	if err != nil {
		return nil, fmt.Errorf("create preview directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(directory) }()

	restored, err := restoreSingleFile(ctx, handler, snapshotID, path, directory)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(restored)
	if err != nil {
		return nil, fmt.Errorf("open restored file: %w", err)
	}
	defer func() { _ = file.Close() }()

	return io.ReadAll(io.LimitReader(file, maxSize))
}

func (s *Snapshots) ExtractFile(ctx context.Context, extraction *snapshot.FileExtraction) (string, error) {
	handler, err := s.session.current()
	if err != nil {
		return "", err
	}

	target, err := os.MkdirTemp(extraction.Directory(), "extract-*")
	if err != nil {
		return "", fmt.Errorf("create extraction directory: %w", err)
	}

	return restoreSingleFile(ctx, handler, extraction.SnapshotID(), extraction.Path(), target)
}

func restoreSingleFile(ctx context.Context, handler *stowmark.Handler, snapshotID, path, directory string) (string, error) {
	if err := handler.RestoreFile(ctx, snapshotID, path, &directory, nil); err != nil {
		return "", err
	}

	return findSingleFile(directory)
}

func findSingleFile(directory string) (string, error) {
	var found string

	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("find restored file: %w", err)
	}

	if found == "" {
		return "", ErrRestoredFileNotFound
	}

	return found, nil
}

func failures(result *stowmark.Result) []snapshot.Failure {
	if result.IsSuccess {
		return nil
	}

	failed := make([]snapshot.Failure, len(result.Failed))
	for i, failure := range result.Failed {
		failed[i] = snapshot.Failure{Path: failure.Path, Reason: failure.Reason}
	}

	return failed
}

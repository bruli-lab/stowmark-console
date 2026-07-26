package repository

import "context"

type FolderRepository interface {
	ReadEntries(ctx context.Context, path string) ([]Entry, error)
	IsTextFile(ctx context.Context, filePath string) (bool, error)
}

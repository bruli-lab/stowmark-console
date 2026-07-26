package repository

import "context"

type Read struct {
	repo FolderRepository
}

func (r *Read) ReadEntries(ctx context.Context, path string) ([]Entry, error) {
	return r.repo.ReadEntries(ctx, path)
}

func NewRead(repo FolderRepository) *Read {
	return &Read{repo: repo}
}

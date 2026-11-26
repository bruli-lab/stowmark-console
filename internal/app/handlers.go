package app

import (
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type Handlers struct {
	ListRepositories *ListRepositories
	OpenRepository   *OpenRepository
	RepositoryStatus *RepositoryStatus
	InitRepository   *InitRepository
	ListSnapshots    *ListSnapshots
	GetManifest      *GetManifest
	GetFileContent   *GetFileContent
	CreateSnapshot   *CreateSnapshot
	VerifySnapshot   *VerifySnapshot
	RestoreSnapshot  *RestoreSnapshot
}

func NewHandlers(catalog repository.Catalog, repositories repository.Repository, snapshots snapshot.Repository) Handlers {
	return Handlers{
		ListRepositories: NewListRepositories(catalog),
		OpenRepository:   NewOpenRepository(catalog),
		RepositoryStatus: NewRepositoryStatus(repositories),
		InitRepository:   NewInitRepository(repositories),
		ListSnapshots:    NewListSnapshots(snapshots),
		GetManifest:      NewGetManifest(snapshots),
		GetFileContent:   NewGetFileContent(snapshots),
		CreateSnapshot:   NewCreateSnapshot(snapshots),
		VerifySnapshot:   NewVerifySnapshot(snapshots),
		RestoreSnapshot:  NewRestoreSnapshot(snapshots),
	}
}

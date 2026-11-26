package app

import (
	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type Buses struct {
	Commands cqs.CommandBus
	Queries  cqs.QueryBus
}

func NewBuses(catalog repository.Catalog, repositories repository.Repository, snapshots snapshot.Repository) Buses {
	commands := cqs.NewCommandBus()
	commands.Subscribe(OpenRepositoryCommandName, NewOpenRepository(catalog))
	commands.Subscribe(InitRepositoryCommandName, NewInitRepository(repositories))
	commands.Subscribe(CreateSnapshotCommandName, NewCreateSnapshot(snapshots))
	commands.Subscribe(VerifySnapshotCommandName, NewVerifySnapshot(snapshots))
	commands.Subscribe(RestoreSnapshotCommandName, NewRestoreSnapshot(snapshots))
	commands.Subscribe(ExtractFileCommandName, NewExtractFile(snapshots))

	queries := cqs.NewQueryBus()
	queries.Subscribe(ListRepositoriesQueryName, NewListRepositories(catalog))
	queries.Subscribe(RepositoryStatusQueryName, NewRepositoryStatus(repositories))
	queries.Subscribe(ListSnapshotsQueryName, NewListSnapshots(snapshots))
	queries.Subscribe(GetManifestQueryName, NewGetManifest(snapshots))
	queries.Subscribe(GetFileContentQueryName, NewGetFileContent(snapshots))

	return Buses{Commands: commands, Queries: queries}
}

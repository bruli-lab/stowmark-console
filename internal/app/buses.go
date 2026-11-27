package app

import (
	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type Buses struct {
	Commands cqs.CommandBus
	Queries  cqs.QueryBus
}

type Ports struct {
	Catalog      repository.Catalog
	Repositories repository.Repository
	Snapshots    snapshot.Repository
	Keys         encryption.Keys
}

func NewBuses(ports Ports) Buses {
	commands := cqs.NewCommandBus()
	commands.Subscribe(OpenRepositoryCommandName, NewOpenRepository(ports.Catalog))
	commands.Subscribe(InitRepositoryCommandName, NewInitRepository(ports.Repositories))
	commands.Subscribe(CreateSnapshotCommandName, NewCreateSnapshot(ports.Snapshots))
	commands.Subscribe(VerifySnapshotCommandName, NewVerifySnapshot(ports.Snapshots))
	commands.Subscribe(RestoreSnapshotCommandName, NewRestoreSnapshot(ports.Snapshots))
	commands.Subscribe(ExtractFileCommandName, NewExtractFile(ports.Snapshots))
	commands.Subscribe(GenerateKeysCommandName, NewGenerateKeys(ports.Keys))
	commands.Subscribe(RewrapKeyCommandName, NewRewrapKey(ports.Keys))
	commands.Subscribe(RotateKeyCommandName, NewRotateKey(ports.Keys))

	queries := cqs.NewQueryBus()
	queries.Subscribe(ListRepositoriesQueryName, NewListRepositories(ports.Catalog))
	queries.Subscribe(RepositoryStatusQueryName, NewRepositoryStatus(ports.Repositories))
	queries.Subscribe(ListSnapshotsQueryName, NewListSnapshots(ports.Snapshots))
	queries.Subscribe(GetManifestQueryName, NewGetManifest(ports.Snapshots))
	queries.Subscribe(GetFileContentQueryName, NewGetFileContent(ports.Snapshots))

	return Buses{Commands: commands, Queries: queries}
}

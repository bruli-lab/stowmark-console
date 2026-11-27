package app

import (
	"context"
	"slices"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

const InitEncryptedRepositoryCommandName = "init-encrypted-repository"

type InitEncryptedRepositoryCommand struct {
	Profile    repository.Profile
	KeysFolder string
}

func (InitEncryptedRepositoryCommand) Name() string {
	return InitEncryptedRepositoryCommandName
}

type InitEncryptedRepository struct {
	catalog    repository.Catalog
	repository repository.Repository
	keys       encryption.Keys
}

func NewInitEncryptedRepository(catalog repository.Catalog, repo repository.Repository, keys encryption.Keys) InitEncryptedRepository {
	return InitEncryptedRepository{catalog: catalog, repository: repo, keys: keys}
}

func (h InitEncryptedRepository) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(InitEncryptedRepositoryCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(InitEncryptedRepositoryCommandName, command.Name())
	}

	if cmd.Profile.Name() == "" {
		return nil, ErrRepositoryNameRequired
	}

	generation, err := encryption.NewKeyGeneration(cmd.KeysFolder)
	if err != nil {
		return nil, err
	}

	keys, err := h.keys.Generate(ctx, generation)
	if err != nil {
		return nil, err
	}

	if err := generation.Generated(keys); err != nil {
		return nil, err
	}

	opened, err := h.catalog.Update(ctx, cmd.Profile.WithKeys(keys.PublicKey(), keys.PrivateKey()))
	if err != nil {
		return nil, err
	}

	if err := h.repository.Initialize(ctx, opened.Init()); err != nil {
		return nil, err
	}

	return slices.Concat(
		generation.Events(),
		[]event.Event{repository.NewOpened(opened), repository.NewInitialized(opened.Init())},
	), nil
}

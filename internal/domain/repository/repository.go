package repository

import "context"

type Repository interface {
	Status(ctx context.Context) (Status, error)
	Initialize(ctx context.Context, options InitOptions) error
	Unlock(ctx context.Context, all bool) (LockReport, error)
}

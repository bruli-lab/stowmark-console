package repository

import "context"

type Repository interface {
	Initialized(ctx context.Context) (bool, error)
	Initialize(ctx context.Context, options InitOptions) error
}

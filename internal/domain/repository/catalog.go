package repository

import (
	"context"
	"errors"
)

var ErrUnknownProfile = errors.New("unknown repository")

type Catalog interface {
	Profiles() []Profile
	Open(ctx context.Context, name string) (Profile, error)
}

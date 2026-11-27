package repository

import "errors"

var ErrLocked = errors.New("repository is locked")

type LockReport struct {
	Removed int
	Active  int
}

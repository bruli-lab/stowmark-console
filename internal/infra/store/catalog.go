package store

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

type Entry struct {
	Profile     repository.Profile
	PrivateKey  *string
	Environment map[string]string
}

type Catalog struct {
	session  *Session
	entries  []Entry
	current  *Entry
	original map[string]*string
}

func NewCatalog(session *Session, entries []Entry) *Catalog {
	return &Catalog{session: session, entries: entries}
}

func (c *Catalog) Profiles() []repository.Profile {
	profiles := make([]repository.Profile, len(c.entries))
	for i := range c.entries {
		profiles[i] = c.entries[i].Profile
	}

	return profiles
}

func (c *Catalog) Open(ctx context.Context, name string) (repository.Profile, error) {
	entry, found := c.find(name)
	if !found {
		return repository.Profile{}, fmt.Errorf("%w %q", repository.ErrUnknownProfile, name)
	}

	if err := c.applyEnvironment(entry.Environment); err != nil {
		return repository.Profile{}, err
	}

	if err := c.session.Connect(ctx, entry.Profile.Location(), entry.PrivateKey); err != nil {
		return repository.Profile{}, errors.Join(err, c.applyEnvironment(c.currentEnvironment()))
	}

	c.current = entry

	return entry.Profile, nil
}

func (c *Catalog) find(name string) (*Entry, bool) {
	for i := range c.entries {
		if c.entries[i].Profile.Name() == name {
			return &c.entries[i], true
		}
	}

	return nil, false
}

func (c *Catalog) currentEnvironment() map[string]string {
	if c.current == nil {
		return nil
	}

	return c.current.Environment
}

func (c *Catalog) applyEnvironment(environment map[string]string) error {
	if err := c.restoreEnvironment(); err != nil {
		return err
	}

	c.original = make(map[string]*string, len(environment))

	for key, value := range environment {
		if previous, ok := os.LookupEnv(key); ok {
			c.original[key] = &previous
		} else {
			c.original[key] = nil
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}

	return nil
}

func (c *Catalog) restoreEnvironment() error {
	var errs []error

	for key, value := range c.original {
		if value == nil {
			errs = append(errs, os.Unsetenv(key))
			continue
		}
		errs = append(errs, os.Setenv(key, *value))
	}

	c.original = nil

	return errors.Join(errs...)
}

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

	profile := entry.Profile
	if err := c.session.Connect(ctx, profile.Location(), profile.PrivateKey()); err != nil {
		return repository.Profile{}, errors.Join(err, c.applyEnvironment(c.currentEnvironment()))
	}

	c.session.missingPrivateKey = profile.MissingPrivateKey()
	c.current = entry

	return profile.WithFingerprint(keyFingerprint(profile.Init().PublicKey(), profile.PrivateKey())), nil
}

func (c *Catalog) Update(ctx context.Context, profile repository.Profile) (repository.Profile, error) {
	entry, found := c.find(profile.Name())
	if !found {
		return repository.Profile{}, fmt.Errorf("%w %q", repository.ErrUnknownProfile, profile.Name())
	}

	previous := entry.Profile
	entry.Profile = profile

	opened, err := c.Open(ctx, profile.Name())
	if err != nil {
		entry.Profile = previous
		return repository.Profile{}, err
	}

	return opened, nil
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

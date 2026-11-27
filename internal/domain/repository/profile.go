package repository

import (
	"strings"

	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const localKind = "local"

type Profile struct {
	name        string
	location    string
	init        *InitOptions
	privateKey  *string
	fingerprint string
	defaults    *defaults
}

type defaults struct {
	retention  *retention.Rules
	exclusions snapshot.Exclusions
}

func (p Profile) withDefaults(change func(*defaults)) Profile {
	updated := defaults{}
	if p.defaults != nil {
		updated = *p.defaults
	}
	change(&updated)
	p.defaults = &updated

	return p
}

func NewProfile(name, location string, init InitOptions) Profile {
	return Profile{name: name, location: location, init: &init}
}

func (p Profile) WithKeys(publicKey, privateKey string) Profile {
	init := p.Init().WithPublicKey(publicKey)
	p.init = &init
	p.privateKey = &privateKey
	return p
}

func (p Profile) WithInit(options InitOptions) Profile {
	p.init = &options
	return p
}

func (p Profile) WithPrivateKey(path *string) Profile {
	p.privateKey = path
	return p
}

func (p Profile) WithFingerprint(fingerprint string) Profile {
	p.fingerprint = fingerprint
	return p
}

func (p Profile) WithRetention(rules *retention.Rules) Profile {
	return p.withDefaults(func(d *defaults) { d.retention = rules })
}

func (p Profile) Retention() (retention.Rules, bool) {
	if p.defaults == nil || p.defaults.retention == nil {
		return retention.Rules{}, false
	}

	return *p.defaults.retention, true
}

func (p Profile) WithExclusions(exclusions snapshot.Exclusions) Profile {
	return p.withDefaults(func(d *defaults) { d.exclusions = exclusions })
}

func (p Profile) Exclusions() snapshot.Exclusions {
	if p.defaults == nil {
		return snapshot.Exclusions{}
	}

	return p.defaults.exclusions
}

func (p Profile) Name() string {
	return p.name
}

func (p Profile) Location() string {
	return p.location
}

func (p Profile) Init() InitOptions {
	if p.init == nil {
		return NewInitOptions("", nil, nil)
	}

	return *p.init
}

func (p Profile) PrivateKey() *string {
	return p.privateKey
}

func (p Profile) Fingerprint() string {
	return p.fingerprint
}

func (p Profile) Encrypted() bool {
	return p.privateKey != nil || p.Init().Encrypted()
}

func (p Profile) MissingPrivateKey() bool {
	return p.privateKey == nil && p.Init().Encrypted()
}

func (p Profile) Kind() string {
	scheme, _, found := strings.Cut(p.location, "://")
	if !found || scheme == "" {
		return localKind
	}

	return strings.ToLower(scheme)
}

func (p Profile) Local() bool {
	return p.Kind() == localKind
}

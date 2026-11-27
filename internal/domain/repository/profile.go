package repository

import (
	"strings"

	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
)

const localKind = "local"

type Profile struct {
	name        string
	location    string
	init        *InitOptions
	privateKey  *string
	fingerprint string
	retention   *retention.Rules
}

func NewProfile(name, location string, init InitOptions) Profile {
	return Profile{name: name, location: location, init: &init}
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
	p.retention = rules
	return p
}

func (p Profile) Retention() (retention.Rules, bool) {
	if p.retention == nil {
		return retention.Rules{}, false
	}

	return *p.retention, true
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

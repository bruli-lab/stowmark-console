package repository

import "strings"

const localKind = "local"

type Profile struct {
	name     string
	location string
	init     InitOptions
}

func NewProfile(name, location string, init InitOptions) Profile {
	return Profile{name: name, location: location, init: init}
}

func (p Profile) Name() string {
	return p.name
}

func (p Profile) Location() string {
	return p.location
}

func (p Profile) Init() InitOptions {
	return p.init
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

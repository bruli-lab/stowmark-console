package repository

import (
	"github.com/bruli-lab/go-core/event"
	"github.com/google/uuid"
)

const (
	InitializedEventName = "repository.initialized"
	OpenedEventName      = "repository.opened"
)

type Initialized struct {
	event.BasicEvent
	Options InitOptions
}

func NewInitialized(options InitOptions) Initialized {
	return Initialized{
		BasicEvent: event.NewBasicEvent(InitializedEventName, uuid.New(), ""),
		Options:    options,
	}
}

type Opened struct {
	event.BasicEvent
	Profile Profile
}

func NewOpened(profile Profile) Opened {
	return Opened{
		BasicEvent: event.NewBasicEvent(OpenedEventName, uuid.New(), profile.Name()),
		Profile:    profile,
	}
}

package encryption

import (
	"github.com/bruli-lab/go-core/event"
	"github.com/google/uuid"
)

const (
	KeysGeneratedEventName = "encryption.keys_generated"
	KeyRewrappedEventName  = "encryption.key_rewrapped"
	KeyRotatedEventName    = "encryption.key_rotated"
)

type KeysGenerated struct {
	event.BasicEvent
	Keys KeyPair
}

func NewKeysGenerated(folder string, keys KeyPair) KeysGenerated {
	return KeysGenerated{
		BasicEvent: event.NewBasicEvent(KeysGeneratedEventName, uuid.New(), folder),
		Keys:       keys,
	}
}

type KeyRewrapped struct {
	event.BasicEvent
	OldPrivateKey string
	NewPublicKey  string
}

func NewKeyRewrapped(oldPrivateKey, newPublicKey string) KeyRewrapped {
	return KeyRewrapped{
		BasicEvent:    event.NewBasicEvent(KeyRewrappedEventName, uuid.New(), newPublicKey),
		OldPrivateKey: oldPrivateKey,
		NewPublicKey:  newPublicKey,
	}
}

type KeyRotated struct {
	event.BasicEvent
	PrivateKey string
	PublicKey  string
}

func NewKeyRotated(privateKey, publicKey string) KeyRotated {
	return KeyRotated{
		BasicEvent: event.NewBasicEvent(KeyRotatedEventName, uuid.New(), publicKey),
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
}

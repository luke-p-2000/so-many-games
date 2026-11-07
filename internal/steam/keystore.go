package steam

import (
	"errors"

	"github.com/zalando/go-keyring"
)

// KeyStore holds the Steam Web API key. The only real implementation is the Windows
// Credential Manager; the key must never reach the database, exports or logs (S2).
type KeyStore interface {
	Get() (string, error) // ErrNoKey when nothing is stored
	Set(key string) error
	Delete() error
}

// ErrNoKey means no API key has been saved yet.
var ErrNoKey = errors.New("no Steam API key saved")

const (
	keyringService = "SoManyGames"
	keyringUser    = "steam-web-api-key"
)

// CredentialManager stores the key in the Windows Credential Manager via go-keyring.
type CredentialManager struct{}

func (CredentialManager) Get() (string, error) {
	k, err := keyring.Get(keyringService, keyringUser)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNoKey
	}
	return k, err
}

func (CredentialManager) Set(key string) error {
	return keyring.Set(keyringService, keyringUser, key)
}

func (CredentialManager) Delete() error {
	err := keyring.Delete(keyringService, keyringUser)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

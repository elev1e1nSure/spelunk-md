package config

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "spelunk-md"
	accountName = "openrouter-api-key"

	DefaultModel = "deepseek/deepseek-v4-flash"
)

var ErrNoAPIKey = errors.New("api key not set — run: spelunk-md --api-key YOUR_KEY")

// SetAPIKey saves the key to the system keyring.
func SetAPIKey(key string) error {
	if err := keyring.Set(serviceName, accountName, key); err != nil {
		return fmt.Errorf("failed to save api key: %w", err)
	}
	return nil
}

// DeleteAPIKey removes the key from the system keyring.
func DeleteAPIKey() error {
	err := keyring.Delete(serviceName, accountName)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("failed to delete api key: %w", err)
	}
	return nil
}

// GetAPIKey retrieves the key from the system keyring.
func GetAPIKey() (string, error) {
	key, err := keyring.Get(serviceName, accountName)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNoAPIKey
		}
		return "", fmt.Errorf("keyring error: %w", err)
	}
	if key == "" {
		return "", ErrNoAPIKey
	}
	return key, nil
}

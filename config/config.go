package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "spelunk-md"
	accountName = "openrouter-api-key"

	DefaultModel = "deepseek/deepseek-v4-flash"

	// EnvAPIKey is checked as a fallback when the keyring has no stored key.
	EnvAPIKey = "OPENROUTER_API_KEY"
)

var ErrNoAPIKey = errors.New(
	"api key not set — run: spelunk-md --api-key YOUR_KEY  (or set OPENROUTER_API_KEY env var)",
)

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

// GetAPIKey retrieves the key from the system keyring, falling back to the
// OPENROUTER_API_KEY environment variable if the keyring has no entry.
// Keys sourced from env are used as-is and are never persisted to the keyring.
func GetAPIKey() (string, error) {
	key, err := keyring.Get(serviceName, accountName)
	if err == nil && key != "" {
		return key, nil
	}

	if !errors.Is(err, keyring.ErrNotFound) && err != nil {
		// Real keyring error (not just "not found") — warn but continue to env fallback.
		fmt.Fprintf(os.Stderr, "warning: keyring error: %v — falling back to %s\n", err, EnvAPIKey)
	}

	if envKey := os.Getenv(EnvAPIKey); envKey != "" {
		return envKey, nil
	}

	return "", ErrNoAPIKey
}

package storage

import (
	"fmt"

	"github.com/appetiteclub/apt"
)

// FromProperties builds the storage backend from apt.Config.
// NOTE: once aqm ships a storage binding primitive this can move there.
func FromProperties(config *apt.Config) (MediaStorage, error) {
	if config == nil {
		return nil, fmt.Errorf("storage: properties required")
	}

	backend, _ := config.GetString("storage.backend")
	switch backend {
	case "", "local":
		directory, _ := config.GetString("storage.local.directory")
		local, err := NewLocalBackend(directory)
		if err != nil {
			return nil, fmt.Errorf("storage: local backend: %w", err)
		}
		return local, nil
	case "noop":
		return NewNoopBackend(), nil
	default:
		return nil, fmt.Errorf("storage: unsupported backend %q", backend)
	}
}

package media

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/appetiteclub/appetite/services/media/internal/storage"
)

type FileStore struct {
	backend storage.MediaStorage
}

func NewFileStore(backend storage.MediaStorage) *FileStore {
	if backend == nil {
		backend, _ = storage.NewLocalBackend("./data/media")
	}
	return &FileStore{backend: backend}
}

func (fs *FileStore) Save(ctx context.Context, name string, data io.Reader, mime string) (string, error) {
	if fs.backend == nil {
		return "", fmt.Errorf("media filestore: backend not configured")
	}
	if name == "" {
		name = storage.GenerateName()
	}
	return fs.backend.Save(ctx, filepath.Base(name), data, mime)
}

func (fs *FileStore) Delete(ctx context.Context, path string) error {
	if fs.backend == nil {
		return nil
	}
	return fs.backend.Delete(ctx, path)
}

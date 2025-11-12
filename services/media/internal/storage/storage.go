package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

type MediaStorage interface {
	Save(ctx context.Context, id string, data io.Reader, mime string) (string, error)
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
	URL(ctx context.Context, path string) (string, error)
}

type LocalBackend struct {
	baseDir string
}

func NewLocalBackend(baseDir string) (*LocalBackend, error) {
	if baseDir == "" {
		baseDir = "./data/media"
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	return &LocalBackend{baseDir: baseDir}, nil
}

func (l *LocalBackend) Save(ctx context.Context, id string, data io.Reader, _ string) (string, error) {
	if id == "" {
		return "", ErrInvalidID
	}
	destPath := filepath.Join(l.baseDir, id)
	file, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if data != nil {
		if _, err := io.Copy(file, data); err != nil {
			return "", err
		}
	}
	return destPath, nil
}

func (l *LocalBackend) Delete(ctx context.Context, path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (l *LocalBackend) Exists(ctx context.Context, path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (l *LocalBackend) URL(ctx context.Context, path string) (string, error) {
	return path, nil
}

type NoopBackend struct{}

func NewNoopBackend() *NoopBackend {
	return &NoopBackend{}
}

func (n *NoopBackend) Save(ctx context.Context, id string, data io.Reader, mime string) (string, error) {
	return id, nil
}

func (n *NoopBackend) Delete(ctx context.Context, path string) error {
	return nil
}

func (n *NoopBackend) Exists(ctx context.Context, path string) (bool, error) {
	return true, nil
}

func (n *NoopBackend) URL(ctx context.Context, path string) (string, error) {
	return path, nil
}

var ErrInvalidID = errors.New("storage: invalid id")

func GenerateName() string {
	return time.Now().UTC().Format("20060102_150405.000000000")
}

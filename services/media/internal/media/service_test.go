package media

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/google/uuid"

	"github.com/appetiteclub/appetite/services/media/internal/dictionary"
	"github.com/appetiteclub/appetite/services/media/internal/storage"
)

type stubDictionary struct {
	categories []uuid.UUID
	tags       [][]uuid.UUID
	failCat    bool
	failTags   bool
}

func (s *stubDictionary) EnsureCategory(ctx context.Context, id uuid.UUID) error {
	if s.failCat {
		return io.EOF
	}
	s.categories = append(s.categories, id)
	return nil
}

func (s *stubDictionary) EnsureTags(ctx context.Context, ids []uuid.UUID) error {
	if s.failTags {
		return io.EOF
	}
	copied := append([]uuid.UUID(nil), ids...)
	s.tags = append(s.tags, copied)
	return nil
}

type stubStorage struct {
	saves []struct {
		id   string
		mime string
		size int
	}
	deletes []string
	path    string
}

func (s *stubStorage) Save(ctx context.Context, id string, data io.Reader, mime string) (string, error) {
	buf := &bytes.Buffer{}
	if data != nil {
		io.Copy(buf, data)
	}
	s.saves = append(s.saves, struct {
		id   string
		mime string
		size int
	}{id: id, mime: mime, size: buf.Len()})
	if s.path != "" {
		return s.path, nil
	}
	return "stored/" + id, nil
}

func (s *stubStorage) Delete(ctx context.Context, path string) error {
	s.deletes = append(s.deletes, path)
	return nil
}

func (s *stubStorage) Exists(context.Context, string) (bool, error) { return true, nil }

func (s *stubStorage) URL(ctx context.Context, path string) (string, error) { return path, nil }

func TestServiceLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	dict := &stubDictionary{}
	store := &stubStorage{path: "stored/custom-path"}

	svc := NewService(repo, store, dict, ServiceOptions{
		EnableCropping:    true,
		EnableCompression: true,
		Variants: []VariantDefinition{
			{Name: "thumb", Width: 150, Height: 150},
		},
	})

	resourceID := uuid.New()
	categoryID := uuid.New()
	tagID := uuid.New()

	media, err := svc.CreateMedia(ctx, CreateInput{
		ResourceType: "property",
		ResourceID:   resourceID,
		MimeType:     "image/jpeg",
		Resolution:   Resolution{Width: 1024, Height: 768},
		Filesize:     12345,
		Kind:         KindReal,
		CategoryID:   categoryID,
		Tags:         []uuid.UUID{tagID},
		Metadata:     map[string]any{"camera": "test"},
		Enabled:      true,
		Variants: map[string]Variant{
			"thumb": {Path: "thumb.jpg", Width: 150, Height: 150},
		},
		Data: bytes.NewBufferString("fake-image"),
	})
	if err != nil {
		t.Fatalf("CreateMedia failed: %v", err)
	}
	if media.StoragePath != "stored/custom-path" {
		t.Fatalf("expected storage path override, got %s", media.StoragePath)
	}
	if !media.CroppingApplied || !media.CompressionApplied {
		t.Fatalf("expected processing flags to propagate")
	}
	if len(dict.categories) != 1 || dict.categories[0] != categoryID {
		t.Fatalf("expected category validation to run")
	}
	if len(dict.tags) != 1 || len(dict.tags[0]) != 1 || dict.tags[0][0] != tagID {
		t.Fatalf("expected tag validation to run")
	}

	list, err := svc.ListMedia(ctx, ListFilter{ResourceType: "property", ResourceID: resourceID})
	if err != nil {
		t.Fatalf("ListMedia failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected single media, got %d", len(list))
	}

	allItems, err := svc.ListMedia(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("ListMedia without filter failed: %v", err)
	}
	if len(allItems) != 1 {
		t.Fatalf("expected global media list to include asset, got %d", len(allItems))
	}

	// Update kind and metadata
	newKind := KindAIGenerated
	newMeta := map[string]any{"camera": "updated"}
	newPath := "stored/updated"
	updated, err := svc.UpdateMedia(ctx, UpdateInput{
		ID:          media.ID,
		Kind:        &newKind,
		Metadata:    &newMeta,
		StoragePath: &newPath,
	})
	if err != nil {
		t.Fatalf("UpdateMedia failed: %v", err)
	}
	if updated.Kind != newKind {
		t.Fatalf("expected kind to change")
	}
	if updated.StoragePath != newPath {
		t.Fatalf("expected storage path to change")
	}

	// Toggle disable
	disabled, err := svc.SetMediaEnabled(ctx, media.ID, false)
	if err != nil {
		t.Fatalf("SetMediaEnabled failed: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("media should be disabled")
	}

	// Delete cleans up storage
	if err := svc.DeleteMedia(ctx, media.ID); err != nil {
		t.Fatalf("DeleteMedia failed: %v", err)
	}
	if len(store.deletes) != 1 || store.deletes[0] != newPath {
		t.Fatalf("expected delete to remove stored file")
	}

	list, err = svc.ListMedia(ctx, ListFilter{ResourceType: "property", ResourceID: resourceID})
	if err != nil {
		t.Fatalf("ListMedia after delete failed: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected media list to be empty after delete")
	}

	allItems, err = svc.ListMedia(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("ListMedia without filter after delete failed: %v", err)
	}
	if len(allItems) != 0 {
		t.Fatalf("expected global media list to be empty after delete")
	}
}

func TestServiceValidation(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	svc := NewService(repo, storage.NewNoopBackend(), dictionary.NewNoopClient(), ServiceOptions{})

	_, err := svc.CreateMedia(ctx, CreateInput{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

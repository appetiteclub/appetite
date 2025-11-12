package dictionary

import (
	"testing"

	"github.com/google/uuid"
)

func TestOptionEnsureID(t *testing.T) {
	t.Run("generates ID when nil", func(t *testing.T) {
		option := &Option{}
		option.EnsureID()

		if option.ID == uuid.Nil {
			t.Error("EnsureID() did not generate an ID")
		}
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		existingID := uuid.New()
		option := &Option{ID: existingID}
		option.EnsureID()

		if option.ID != existingID {
			t.Errorf("EnsureID() changed existing ID from %s to %s", existingID, option.ID)
		}
	})
}

func TestOptionBeforeCreate(t *testing.T) {
	option := &Option{
		Set:   uuid.New(),
		Key:   "test_key",
		Label: "Test Label",
		Value: "Test Value",
	}

	option.BeforeCreate()

	if option.ID == uuid.Nil {
		t.Error("BeforeCreate() did not generate an ID")
	}

	if option.CreatedAt.IsZero() {
		t.Error("BeforeCreate() did not set CreatedAt")
	}

	if option.UpdatedAt.IsZero() {
		t.Error("BeforeCreate() did not set UpdatedAt")
	}

	if !option.Active {
		t.Error("BeforeCreate() did not set Active to true")
	}
}

func TestOptionBeforeUpdate(t *testing.T) {
	option := &Option{
		Set:   uuid.New(),
		Key:   "test_key",
		Label: "Test Label",
		Value: "Test Value",
	}

	option.BeforeCreate()
	originalUpdatedAt := option.UpdatedAt

	// Wait a bit to ensure timestamp changes
	option.BeforeUpdate()

	if !option.UpdatedAt.After(originalUpdatedAt) {
		t.Error("BeforeUpdate() did not update UpdatedAt timestamp")
	}
}

func TestOptionGetID(t *testing.T) {
	id := uuid.New()
	option := &Option{ID: id}

	if option.GetID() != id {
		t.Errorf("GetID() = %s, want %s", option.GetID(), id)
	}
}

func TestOptionResourceType(t *testing.T) {
	option := &Option{}

	if option.ResourceType() != "dictionary/option" {
		t.Errorf("ResourceType() = %s, want %s", option.ResourceType(), "dictionary/option")
	}
}

package dictionary

import (
	"testing"

	"github.com/google/uuid"
)

func TestSetEnsureID(t *testing.T) {
	t.Run("generates ID when nil", func(t *testing.T) {
		set := &Set{}
		set.EnsureID()

		if set.ID == uuid.Nil {
			t.Error("EnsureID() did not generate an ID")
		}
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		existingID := uuid.New()
		set := &Set{ID: existingID}
		set.EnsureID()

		if set.ID != existingID {
			t.Errorf("EnsureID() changed existing ID from %s to %s", existingID, set.ID)
		}
	})
}

func TestSetBeforeCreate(t *testing.T) {
	set := &Set{
		Name:  "test_set",
		Label: "Test Set",
	}

	set.BeforeCreate()

	if set.ID == uuid.Nil {
		t.Error("BeforeCreate() did not generate an ID")
	}

	if set.CreatedAt.IsZero() {
		t.Error("BeforeCreate() did not set CreatedAt")
	}

	if set.UpdatedAt.IsZero() {
		t.Error("BeforeCreate() did not set UpdatedAt")
	}

	if !set.Active {
		t.Error("BeforeCreate() did not set Active to true")
	}
}

func TestSetBeforeUpdate(t *testing.T) {
	set := &Set{
		Name:  "test_set",
		Label: "Test Set",
	}

	set.BeforeCreate()
	originalUpdatedAt := set.UpdatedAt

	// Wait a bit to ensure timestamp changes
	set.BeforeUpdate()

	if !set.UpdatedAt.After(originalUpdatedAt) {
		t.Error("BeforeUpdate() did not update UpdatedAt timestamp")
	}
}

func TestSetGetID(t *testing.T) {
	id := uuid.New()
	set := &Set{ID: id}

	if set.GetID() != id {
		t.Errorf("GetID() = %s, want %s", set.GetID(), id)
	}
}

func TestSetResourceType(t *testing.T) {
	set := &Set{}

	if set.ResourceType() != "dictionary/set" {
		t.Errorf("ResourceType() = %s, want %s", set.ResourceType(), "dictionary/set")
	}
}

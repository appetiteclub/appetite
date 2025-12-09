package order

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewTableStateCache(t *testing.T) {
	tests := []struct {
		name       string
		withLogger bool
	}{
		{
			name:       "withNilLogger",
			withLogger: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewTableStateCache(nil, nil)

			if cache == nil {
				t.Fatal("NewTableStateCache() returned nil")
			}

			if cache.state == nil {
				t.Error("NewTableStateCache() should initialize state map")
			}

			if cache.logger == nil {
				t.Error("NewTableStateCache() should set a noop logger when nil is passed")
			}
		})
	}
}

func TestTableStateCacheGetSet(t *testing.T) {
	cache := NewTableStateCache(nil, nil)
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440030")

	t.Run("getFromEmptyCache", func(t *testing.T) {
		status, ok := cache.Get(tableID)
		if ok {
			t.Error("Get() should return false for non-existent key")
		}
		if status != "" {
			t.Errorf("Get() status = %q, want empty string", status)
		}
	})

	t.Run("setAndGet", func(t *testing.T) {
		cache.Set(tableID, "occupied")

		status, ok := cache.Get(tableID)
		if !ok {
			t.Error("Get() should return true for existing key")
		}
		if status != "occupied" {
			t.Errorf("Get() status = %q, want %q", status, "occupied")
		}
	})

	t.Run("overwriteExisting", func(t *testing.T) {
		cache.Set(tableID, "available")

		status, ok := cache.Get(tableID)
		if !ok {
			t.Error("Get() should return true for existing key")
		}
		if status != "available" {
			t.Errorf("Get() status = %q, want %q", status, "available")
		}
	})
}

func TestTableStateCacheGetSetConcurrency(t *testing.T) {
	cache := NewTableStateCache(nil, nil)
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440031")

	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			cache.Set(tableID, "status1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			cache.Set(tableID, "status2")
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			cache.Get(tableID)
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	// If we got here without a race condition, the test passes
	_, ok := cache.Get(tableID)
	if !ok {
		t.Error("Expected table to be in cache after concurrent operations")
	}
}

func TestTableStateCacheEnsureInvalidID(t *testing.T) {
	cache := NewTableStateCache(nil, nil)

	_, err := cache.Ensure(nil, uuid.Nil)
	if err == nil {
		t.Error("Ensure() should return error for nil UUID")
	}

	expectedMsg := "invalid table id"
	if err.Error() != expectedMsg {
		t.Errorf("Ensure() error = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestTableStateCacheEnsureCached(t *testing.T) {
	cache := NewTableStateCache(nil, nil)
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440032")

	// Pre-populate cache
	cache.Set(tableID, "reserved")

	status, err := cache.Ensure(nil, tableID)
	if err != nil {
		t.Errorf("Ensure() unexpected error: %v", err)
	}
	if status != "reserved" {
		t.Errorf("Ensure() status = %q, want %q", status, "reserved")
	}
}

func TestTableStateCacheWarmNilClient(t *testing.T) {
	cache := NewTableStateCache(nil, nil)

	err := cache.Warm(nil)
	if err != nil {
		t.Errorf("Warm() with nil client should return nil, got: %v", err)
	}
}

func TestTableStateCacheRefreshNilClient(t *testing.T) {
	cache := NewTableStateCache(nil, nil)
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440033")

	_, err := cache.Refresh(nil, tableID)
	if err == nil {
		t.Error("Refresh() with nil client should return error")
	}

	expectedMsg := "table cache uninitialized"
	if err.Error() != expectedMsg {
		t.Errorf("Refresh() error = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestRehydrate(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name: "validSingleRecord",
			input: map[string]interface{}{
				"id":     "550e8400-e29b-41d4-a716-446655440034",
				"status": "available",
			},
			wantErr: false,
		},
		{
			name: "validSlice",
			input: []map[string]interface{}{
				{"id": "550e8400-e29b-41d4-a716-446655440035", "status": "occupied"},
				{"id": "550e8400-e29b-41d4-a716-446655440036", "status": "reserved"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out tableStateDTO
			if tt.name == "validSlice" {
				var outSlice []tableStateDTO
				err := rehydrate(tt.input, &outSlice)
				if (err != nil) != tt.wantErr {
					t.Errorf("rehydrate() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && len(outSlice) != 2 {
					t.Errorf("rehydrate() got %d records, want 2", len(outSlice))
				}
			} else {
				err := rehydrate(tt.input, &out)
				if (err != nil) != tt.wantErr {
					t.Errorf("rehydrate() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && out.Status != "available" {
					t.Errorf("rehydrate() status = %q, want %q", out.Status, "available")
				}
			}
		})
	}
}

func TestIngestCollection(t *testing.T) {
	cache := NewTableStateCache(nil, nil)

	tests := []struct {
		name          string
		input         interface{}
		expectedCount int
		wantErr       bool
	}{
		{
			name: "validRecords",
			input: []map[string]interface{}{
				{"id": "550e8400-e29b-41d4-a716-446655440037", "status": "available"},
				{"id": "550e8400-e29b-41d4-a716-446655440038", "status": "occupied"},
			},
			expectedCount: 2,
			wantErr:       false,
		},
		{
			name: "withInvalidUUID",
			input: []map[string]interface{}{
				{"id": "valid-uuid-550e8400-e29b-41d4-a716-446655440039", "status": "available"},
				{"id": "not-a-valid-uuid", "status": "occupied"},
			},
			expectedCount: 0, // invalid UUIDs are skipped
			wantErr:       false,
		},
		{
			name:          "emptySlice",
			input:         []map[string]interface{}{},
			expectedCount: 0,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset cache state
			cache.state = make(map[uuid.UUID]string)

			err := cache.ingestCollection(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ingestCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

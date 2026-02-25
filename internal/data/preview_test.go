package data

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestLinkPreviewOperations(t *testing.T) {
	// Setup in-memory SQLite DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}

	store := NewGormStore(db)
	if err := store.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Failed to bootstrap db: %v", err)
	}

	ctx := context.Background()
	url := "http://example.com/test"
	data := []byte(`{"title":"Test Title"}`)

	// Test 1: Get non-existent
	preview, err := store.GetLinkPreview(ctx, url)
	if err != nil {
		t.Errorf("GetLinkPreview returned error for missing key: %v", err)
	}
	if preview != nil {
		t.Errorf("GetLinkPreview should return nil for missing key, got %v", preview)
	}

	// Test 2: Insert and Get
	if err := store.InsertLinkPreview(ctx, url, data); err != nil {
		t.Fatalf("InsertLinkPreview failed: %v", err)
	}

	preview, err = store.GetLinkPreview(ctx, url)
	if err != nil {
		t.Fatalf("GetLinkPreview failed: %v", err)
	}
	if preview == nil {
		t.Fatalf("GetLinkPreview returned nil after insert")
	}
	if preview.URL != url {
		t.Errorf("Expected URL %s, got %s", url, preview.URL)
	}
	if string(preview.Data) != string(data) {
		t.Errorf("Expected Data %s, got %s", data, preview.Data)
	}

	// Test 3: Update (Insert again)
	newData := []byte(`{"title":"Updated Title"}`)
	time.Sleep(10 * time.Millisecond) // Ensure timestamp update
	if err := store.InsertLinkPreview(ctx, url, newData); err != nil {
		t.Fatalf("InsertLinkPreview (update) failed: %v", err)
	}

	preview, err = store.GetLinkPreview(ctx, url)
	if err != nil {
		t.Fatalf("GetLinkPreview failed: %v", err)
	}
	if string(preview.Data) != string(newData) {
		t.Errorf("Expected Data %s, got %s", newData, preview.Data)
	}

	// Test 4: Delete
	if err := store.DeleteLinkPreview(ctx, url); err != nil {
		t.Fatalf("DeleteLinkPreview failed: %v", err)
	}

	preview, err = store.GetLinkPreview(ctx, url)
	if err != nil {
		t.Errorf("GetLinkPreview returned error after delete: %v", err)
	}
	if preview != nil {
		t.Errorf("GetLinkPreview should return nil after delete")
	}
}

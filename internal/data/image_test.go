package data

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestInsertImage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	store := NewGormStore(db)
	if err := store.Bootstrap(context.Background()); err != nil {
		t.Fatalf("failed to bootstrap: %v", err)
	}

	// Insert an image
	id, err := store.InsertImage(context.Background(), &Image{Title: "Daily Kitten", Link: "cat AAS", URL: "https://cataas.com/cat/abc123"})
	if err != nil {
		t.Fatalf("InsertImage failed: %v", err)
	}

	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	// Verify it's in the database via GetRecentImages
	images, err := store.GetRecentImages(context.Background(), 1, 0, SourceFilter{})
	if err != nil {
		t.Fatalf("GetRecentImages failed: %v", err)
	}

	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}

	img := images[0]
	if img.Title != "Daily Kitten" {
		t.Errorf("expected title 'Daily Kitten', got '%s'", img.Title)
	}
	if img.Link != "cat AAS" {
		t.Errorf("expected link 'cat AAS', got '%s'", img.Link)
	}
	if img.URL != "https://cataas.com/cat/abc123" {
		t.Errorf("expected URL 'https://cataas.com/cat/abc123', got '%s'", img.URL)
	}
}

func TestGetTodayImageByLink(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	store := NewGormStore(db)
	if err := store.Bootstrap(context.Background()); err != nil {
		t.Fatalf("failed to bootstrap: %v", err)
	}

	// No image yet
	exists, err := store.GetTodayImageByLink(context.Background(), "cat AAS")
	if err != nil {
		t.Fatalf("GetTodayImageByLink failed: %v", err)
	}
	if exists != nil {
		t.Error("expected nil when no image exists")
	}

	// Insert an image
	_, err = store.InsertImage(context.Background(), &Image{Title: "Daily Kitten", Link: "cat AAS", URL: "https://cataas.com/cat/abc123"})
	if err != nil {
		t.Fatalf("InsertImage failed: %v", err)
	}

	// Now it should exist
	exists, err = store.GetTodayImageByLink(context.Background(), "cat AAS")
	if err != nil {
		t.Fatalf("GetTodayImageByLink failed: %v", err)
	}
	if exists == nil {
		t.Error("expected image to exist")
	}
}

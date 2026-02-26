package data

import (
	"context"
	"testing"
)

func TestFindRelatedLinks_ByTitle(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, &IRCLink{
		User: "alice", Title: "Getting Started with Go",
		URL: "https://go.dev/learn", ContentType: "text/html",
	})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	_, err = store.InsertIRCLink(ctx, &IRCLink{
		User: "bob", Title: "Best Pizza in Chicago",
		URL: "https://pizza.example.com", ContentType: "text/html",
	})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	results, err := store.FindRelatedLinks(ctx, "Go Tutorial for Beginners", "https://gobyexample.com", 0, 10)
	if err != nil {
		t.Fatalf("FindRelatedLinks failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one related link, got 0")
	}

	found := false
	for _, r := range results {
		if r.Title == "Getting Started with Go" {
			found = true
		}
		if r.Title == "Best Pizza in Chicago" {
			t.Error("unrelated link should not appear in results")
		}
	}
	if !found {
		t.Error("expected 'Getting Started with Go' in results")
	}
}

func TestFindRelatedLinks_ByDomain(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, &IRCLink{
		User: "alice", Title: "How DNS Works",
		URL: "https://jvns.ca/blog/dns", ContentType: "text/html",
	})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	results, err := store.FindRelatedLinks(ctx, "Networking Zine", "https://jvns.ca/blog/zines", 0, 10)
	if err != nil {
		t.Fatalf("FindRelatedLinks failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one related link from same domain")
	}
	if results[0].URL != "https://jvns.ca/blog/dns" {
		t.Errorf("expected jvns.ca link, got %s", results[0].URL)
	}
}

func TestFindRelatedLinks_ExcludesSelf(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	id, err := store.InsertIRCLink(ctx, &IRCLink{
		User: "alice", Title: "Go Tutorial",
		URL: "https://go.dev/learn", ContentType: "text/html",
	})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	results, err := store.FindRelatedLinks(ctx, "Go Tutorial", "https://go.dev/learn", id, 10)
	if err != nil {
		t.Fatalf("FindRelatedLinks failed: %v", err)
	}

	for _, r := range results {
		if r.ID == id {
			t.Error("result should not include the excluded link")
		}
	}
}

func TestFindRelatedLinks_RespectsLimit(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := store.InsertIRCLink(ctx, &IRCLink{
			User: "alice", Title: "Go Programming Tips",
			URL:         "https://go.dev/tip/" + string(rune('a'+i)),
			ContentType: "text/html",
		})
		if err != nil {
			t.Fatalf("InsertIRCLink failed: %v", err)
		}
	}

	results, err := store.FindRelatedLinks(ctx, "Go Programming Guide", "https://example.com", 0, 3)
	if err != nil {
		t.Fatalf("FindRelatedLinks failed: %v", err)
	}

	if len(results) > 3 {
		t.Errorf("expected at most 3 results, got %d", len(results))
	}
}

func TestFindRelatedLinks_EmptyTitle(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, &IRCLink{
		User: "alice", Title: "Something",
		URL: "https://example.com/page", ContentType: "text/html",
	})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	results, err := store.FindRelatedLinks(ctx, "", "https://example.com/other", 0, 10)
	if err != nil {
		t.Fatalf("FindRelatedLinks failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result from domain match")
	}
}

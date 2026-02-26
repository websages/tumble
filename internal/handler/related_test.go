package handler

import (
	"testing"
	"time"

	"tumble/internal/data"
)

func TestScoreRelatedLinks_DomainBoost(t *testing.T) {
	now := time.Now()
	candidates := []data.RelatedLink{
		{ID: 1, Title: "Unrelated Article", URL: "https://other.com/page", User: "alice", Clicks: 0, Timestamp: now},
		{ID: 2, Title: "Unrelated Article", URL: "https://jvns.ca/other-post", User: "bob", Clicks: 0, Timestamp: now},
	}

	scored := scoreRelatedLinks(candidates, "Some Title", "https://jvns.ca/my-post", nil)

	// The domain-matched link (jvns.ca) should score higher
	if len(scored) == 0 {
		t.Fatal("expected at least one scored link")
	}

	// Find the jvns.ca link in results
	var domainMatch *ScoredLink
	var noMatch *ScoredLink
	for i := range scored {
		if scored[i].ID == 2 {
			domainMatch = &scored[i]
		}
		if scored[i].ID == 1 {
			noMatch = &scored[i]
		}
	}

	if domainMatch == nil {
		t.Fatal("expected domain-matched link (ID 2) to be in results")
	}

	if noMatch != nil && noMatch.Score >= domainMatch.Score {
		t.Errorf("domain-matched link (score=%.4f) should score higher than non-matched (score=%.4f)", domainMatch.Score, noMatch.Score)
	}
}

func TestScoreRelatedLinks_TitleOverlap(t *testing.T) {
	now := time.Now()
	candidates := []data.RelatedLink{
		{ID: 1, Title: "Completely Different Topic Altogether", URL: "https://a.com/1", User: "alice", Clicks: 0, Timestamp: now},
		{ID: 2, Title: "Understanding Kubernetes Networking Deep Dive", URL: "https://b.com/2", User: "bob", Clicks: 0, Timestamp: now},
	}

	scored := scoreRelatedLinks(candidates, "Understanding Kubernetes Networking Basics", "https://c.com/3", nil)

	// The title-overlapping link should score higher
	var titleMatch *ScoredLink
	var noMatch *ScoredLink
	for i := range scored {
		if scored[i].ID == 2 {
			titleMatch = &scored[i]
		}
		if scored[i].ID == 1 {
			noMatch = &scored[i]
		}
	}

	if titleMatch == nil {
		t.Fatal("expected title-matched link (ID 2) to be in results")
	}

	if noMatch != nil && noMatch.Score >= titleMatch.Score {
		t.Errorf("title-matched link (score=%.4f) should score higher than non-matched (score=%.4f)", titleMatch.Score, noMatch.Score)
	}
}

func TestScoreRelatedLinks_ClickBonus(t *testing.T) {
	now := time.Now()
	candidates := []data.RelatedLink{
		{ID: 1, Title: "Same Domain Post One", URL: "https://example.com/a", User: "alice", Clicks: 0, Timestamp: now},
		{ID: 2, Title: "Same Domain Post Two", URL: "https://example.com/b", User: "bob", Clicks: 100, Timestamp: now},
	}

	scored := scoreRelatedLinks(candidates, "Same Domain Post Source", "https://example.com/src", nil)

	if len(scored) < 2 {
		t.Fatalf("expected at least 2 scored links, got %d", len(scored))
	}

	// High-click link should rank first (both have domain match and similar title)
	if scored[0].ID != 2 {
		t.Errorf("expected high-click link (ID 2) to rank first, got ID %d", scored[0].ID)
	}
}

func TestScoreRelatedLinks_MinScoreFilter(t *testing.T) {
	now := time.Now()
	candidates := []data.RelatedLink{
		{ID: 1, Title: "Completely Unrelated Title XYZZY", URL: "https://unrelated-domain.com/page", User: "alice", Clicks: 0, Timestamp: now},
	}

	scored := scoreRelatedLinks(candidates, "Kubernetes Networking Guide", "https://different-domain.com/article", nil)

	// The completely unrelated link should be filtered out (score < 0.15)
	for _, s := range scored {
		if s.ID == 1 {
			t.Errorf("unrelated link (ID 1) should have been filtered out, but got score=%.4f", s.Score)
		}
	}
}

func TestScoreRelatedLinks_TagOverlap(t *testing.T) {
	now := time.Now()
	candidates := []data.RelatedLink{
		{ID: 1, Title: "Some Tech Article", URL: "https://a.com/1", User: "alice", Clicks: 10, Timestamp: now.Add(-180 * 24 * time.Hour)},
		{ID: 2, Title: "Another Tech Article", URL: "https://b.com/2", User: "bob", Clicks: 10, Timestamp: now.Add(-180 * 24 * time.Hour)},
	}

	tags := map[int][]string{
		0: {"golang", "kubernetes", "networking"},
		1: {"golang", "kubernetes"},
		2: {"python", "django"},
	}

	scored := scoreRelatedLinks(candidates, "Random Title ABCDEF", "https://c.com/3", tags)

	// ID 1 shares tags with the source, ID 2 does not
	var withTags *ScoredLink
	var noTags *ScoredLink
	for i := range scored {
		if scored[i].ID == 1 {
			withTags = &scored[i]
		}
		if scored[i].ID == 2 {
			noTags = &scored[i]
		}
	}

	if withTags != nil && noTags != nil && noTags.Score >= withTags.Score {
		t.Errorf("tag-matched link (score=%.4f) should score higher than non-matched (score=%.4f)", withTags.Score, noTags.Score)
	}
}

func TestScoreRelatedLinks_AgeBonus(t *testing.T) {
	now := time.Now()
	candidates := []data.RelatedLink{
		{ID: 1, Title: "Same Title Exactly Here", URL: "https://same.com/a", User: "alice", Clicks: 0, Timestamp: now},
		{ID: 2, Title: "Same Title Exactly Here", URL: "https://same.com/b", User: "bob", Clicks: 0, Timestamp: now.Add(-400 * 24 * time.Hour)},
	}

	scored := scoreRelatedLinks(candidates, "Same Title Exactly Here", "https://same.com/src", nil)

	if len(scored) < 2 {
		t.Fatalf("expected at least 2 scored links, got %d", len(scored))
	}

	// Older link (ID 2) should rank slightly higher due to age bonus
	var newer *ScoredLink
	var older *ScoredLink
	for i := range scored {
		if scored[i].ID == 1 {
			newer = &scored[i]
		}
		if scored[i].ID == 2 {
			older = &scored[i]
		}
	}

	if newer == nil || older == nil {
		t.Fatal("expected both links in results")
	}

	if older.Score <= newer.Score {
		t.Errorf("older link (score=%.4f) should score slightly higher than newer link (score=%.4f)", older.Score, newer.Score)
	}
}

func TestScoreRelatedLinks_EmptyCandidates(t *testing.T) {
	scored := scoreRelatedLinks(nil, "Some Title", "https://example.com", nil)
	if scored != nil {
		t.Errorf("expected nil for empty candidates, got %v", scored)
	}

	scored = scoreRelatedLinks([]data.RelatedLink{}, "Some Title", "https://example.com", nil)
	if scored != nil {
		t.Errorf("expected nil for empty slice candidates, got %v", scored)
	}
}

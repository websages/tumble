package data

import (
	"context"
	"testing"
	"time"
)

func TestChannelNames_UpsertAndRename(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	slack, team, chanID := "slack", "T1", "C1"
	if err := store.UpsertChannel(ctx, &Channel{ClientType: slack, ClientNetwork: team, ClientChannel: chanID, Name: "beer", UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("UpsertChannel: %v", err)
	}
	if err := store.UpsertChannel(ctx, &Channel{ClientType: slack, ClientNetwork: team, ClientChannel: chanID, Name: "craft-beer", UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("UpsertChannel rename: %v", err)
	}

	names, err := store.GetChannelNames(ctx)
	if err != nil {
		t.Fatalf("GetChannelNames: %v", err)
	}
	if len(names) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(names))
	}
	if got := names.Lookup(&slack, &team, &chanID); got != "craft-beer" {
		t.Errorf("expected renamed channel 'craft-beer', got %q", got)
	}
	other := "C2"
	if got := names.Lookup(&slack, &team, &other); got != "" {
		t.Errorf("expected empty name for unknown channel, got %q", got)
	}
	if got := names.Lookup(nil, &team, &chanID); got != "" {
		t.Errorf("expected empty name for nil client type, got %q", got)
	}
}

func TestGetGlobalTimeline_IncludesClientChannel(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	slack, team, chanID := "slack", "T1", "C1"
	if _, err := store.InsertIRCLink(ctx, &IRCLink{User: "alice", URL: "https://example.com", Title: "x", ClientType: &slack, ClientNetwork: &team, ClientChannel: &chanID}); err != nil {
		t.Fatalf("InsertIRCLink: %v", err)
	}
	items, err := store.GetGlobalTimeline(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetGlobalTimeline: %v", err)
	}
	if len(items) != 1 || items[0].ClientChannel == nil || *items[0].ClientChannel != chanID {
		t.Fatalf("expected client_channel %q in timeline item, got %+v", chanID, items)
	}
}

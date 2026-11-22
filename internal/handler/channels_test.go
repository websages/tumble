package handler

import (
	"context"
	"testing"

	"tumble/internal/data"
)

type mockChannelStore struct {
	data.Store
	upserted []data.Channel
}

func (m *mockChannelStore) UpsertChannel(ctx context.Context, c *data.Channel) error {
	m.upserted = append(m.upserted, *c)
	return nil
}

func TestRecordChannelName(t *testing.T) {
	s := func(v string) *string { return &v }

	tests := []struct {
		name                  string
		ctype, network, chanI *string
		channelName           *string
		wantName              string
	}{
		{"stores name", s("slack"), s("T1"), s("C1"), s("beer"), "beer"},
		{"strips leading hash", s("slack"), s("T1"), s("C1"), s(" #beer "), "beer"},
		{"no name", s("slack"), s("T1"), s("C1"), nil, ""},
		{"empty name", s("slack"), s("T1"), s("C1"), s("#"), ""},
		{"no channel id", s("slack"), s("T1"), nil, s("beer"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockChannelStore{}
			h := &Handler{Store: store}
			h.recordChannelName(context.Background(), tt.ctype, tt.network, tt.chanI, tt.channelName)

			if tt.wantName == "" {
				if len(store.upserted) != 0 {
					t.Fatalf("expected no upsert, got %+v", store.upserted)
				}
				return
			}
			if len(store.upserted) != 1 {
				t.Fatalf("expected 1 upsert, got %d", len(store.upserted))
			}
			got := store.upserted[0]
			if got.Name != tt.wantName || got.ClientChannel != *tt.chanI || got.ClientNetwork != *tt.network || got.ClientType != *tt.ctype {
				t.Errorf("unexpected channel row: %+v", got)
			}
		})
	}
}

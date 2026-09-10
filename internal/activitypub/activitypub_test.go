package activitypub

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/go-fed/httpsig"

	"tumble/internal/config"
	"tumble/internal/data"
)

// memStore implements data.Store for testing by embedding the interface
// (so unused methods panic if called) and overriding only what's needed.
type memStore struct {
	data.Store

	mu         sync.Mutex
	key        *data.ActivityPubKey
	followers  []data.ActivityPubFollower
	deliveries []data.ActivityPubDelivery
	nextID     int
}

func newMemStore() *memStore {
	return &memStore{nextID: 1}
}

func (m *memStore) GetActivityPubKey(ctx context.Context) (*data.ActivityPubKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.key, nil
}

func (m *memStore) InsertActivityPubKey(ctx context.Context, key *data.ActivityPubKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.key = key
	return nil
}

func (m *memStore) UpsertActivityPubFollower(ctx context.Context, f *data.ActivityPubFollower) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, existing := range m.followers {
		if existing.ActorURI == f.ActorURI {
			m.followers[i] = *f
			return nil
		}
	}
	m.followers = append(m.followers, *f)
	return nil
}

func (m *memStore) DeleteActivityPubFollowerByActorURI(ctx context.Context, actorURI string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.followers[:0]
	for _, f := range m.followers {
		if f.ActorURI != actorURI {
			out = append(out, f)
		}
	}
	m.followers = out
	return nil
}

func (m *memStore) ListActivityPubFollowers(ctx context.Context, limit, offset int) ([]data.ActivityPubFollower, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.followers, nil
}

func (m *memStore) CountActivityPubFollowers(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.followers)), nil
}

func (m *memStore) ListActivityPubFollowerInboxes(ctx context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	seen := map[string]bool{}
	var inboxes []string
	for _, f := range m.followers {
		target := f.InboxURL
		if f.SharedInbox != nil && *f.SharedInbox != "" {
			target = *f.SharedInbox
		}
		if !seen[target] {
			seen[target] = true
			inboxes = append(inboxes, target)
		}
	}
	return inboxes, nil
}

func (m *memStore) EnqueueActivityPubDelivery(ctx context.Context, inboxURL string, payload string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deliveries = append(m.deliveries, data.ActivityPubDelivery{
		ID:          m.nextID,
		InboxURL:    inboxURL,
		Payload:     payload,
		Status:      "pending",
		NextAttempt: time.Now(),
	})
	m.nextID++
	return nil
}

func (m *memStore) GetDueActivityPubDeliveries(ctx context.Context, limit int) ([]data.ActivityPubDelivery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var due []data.ActivityPubDelivery
	for _, d := range m.deliveries {
		if d.Status == "pending" && !d.NextAttempt.After(time.Now()) {
			due = append(due, d)
		}
	}
	return due, nil
}

func (m *memStore) MarkActivityPubDeliverySucceeded(ctx context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.deliveries {
		if m.deliveries[i].ID == id {
			m.deliveries[i].Status = "sent"
			m.deliveries[i].Attempts++
		}
	}
	return nil
}

func (m *memStore) MarkActivityPubDeliveryFailed(ctx context.Context, id int, errMsg string, nextAttempt time.Time, giveUp bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.deliveries {
		if m.deliveries[i].ID == id {
			m.deliveries[i].Attempts++
			m.deliveries[i].NextAttempt = nextAttempt
			m.deliveries[i].LastError = &errMsg
			if giveUp {
				m.deliveries[i].Status = "failed"
			}
		}
	}
	return nil
}

func testService() (*Service, *memStore) {
	store := newMemStore()
	cfg := &config.Config{
		BaseURL:         "https://tumble.example.com",
		SiteName:        "tumblefish",
		SiteDescription: "test site",
		ActivityPub: config.ActivityPub{
			Enabled:   true,
			ActorName: "tumble",
		},
	}
	return NewService(cfg, store), store
}

func TestKeyGeneration_RoundTrip(t *testing.T) {
	s, _ := testService()
	ctx := context.Background()

	key, err := s.getOrCreateKey(ctx)
	if err != nil {
		t.Fatalf("getOrCreateKey: %v", err)
	}
	if key.PrivateKey == "" || key.PublicKey == "" {
		t.Fatal("expected non-empty key material")
	}

	// Reuses the persisted key rather than generating a new one.
	key2, err := s.getOrCreateKey(ctx)
	if err != nil {
		t.Fatalf("getOrCreateKey (2nd): %v", err)
	}
	if key.PrivateKey != key2.PrivateKey {
		t.Fatal("expected the same key to be reused")
	}

	priv, err := parsePrivateKeyPEM(key.PrivateKey)
	if err != nil {
		t.Fatalf("parsePrivateKeyPEM: %v", err)
	}
	pub, err := parsePublicKeyPEM(key.PublicKey)
	if err != nil {
		t.Fatalf("parsePublicKeyPEM: %v", err)
	}
	if priv.PublicKey.N.Cmp(pub.N) != 0 {
		t.Fatal("public key does not match private key")
	}
}

func TestBuildActor(t *testing.T) {
	s, _ := testService()
	actor, err := s.BuildActor(context.Background())
	if err != nil {
		t.Fatalf("BuildActor: %v", err)
	}

	if actor.ID != "https://tumble.example.com/activitypub/actor" {
		t.Errorf("unexpected actor ID: %s", actor.ID)
	}
	if actor.Type != "Service" {
		t.Errorf("expected type Service, got %s", actor.Type)
	}
	if actor.PreferredUsername != "tumble" {
		t.Errorf("unexpected preferredUsername: %s", actor.PreferredUsername)
	}
	if actor.Inbox != "https://tumble.example.com/activitypub/inbox" {
		t.Errorf("unexpected inbox: %s", actor.Inbox)
	}
	if actor.PublicKey.PublicKeyPem == "" {
		t.Error("expected a public key")
	}
	if actor.PublicKey.ID != actor.ID+"#main-key" {
		t.Errorf("unexpected public key id: %s", actor.PublicKey.ID)
	}
}

func TestBuildWebfinger(t *testing.T) {
	s, _ := testService()

	doc := s.BuildWebfinger("acct:tumble@tumble.example.com")
	if doc == nil {
		t.Fatal("expected a webfinger match")
	}
	if len(doc.Links) != 1 || doc.Links[0].Href != s.actorID() {
		t.Errorf("unexpected links: %+v", doc.Links)
	}

	if s.BuildWebfinger("acct:someone-else@tumble.example.com") != nil {
		t.Error("expected no match for a different resource")
	}
}

func TestNoteForLink(t *testing.T) {
	s, _ := testService()
	link := &data.IRCLink{ID: 42, Title: "Cool <site>", URL: "https://example.com/x", User: "alice", Timestamp: time.Now()}

	note := s.NoteForLink(link)
	if note.ID != "https://tumble.example.com/link/42" {
		t.Errorf("unexpected note ID: %s", note.ID)
	}
	if note.AttributedTo != s.actorID() {
		t.Errorf("unexpected attributedTo: %s", note.AttributedTo)
	}
	if want := "&lt;site&gt;"; !contains(note.Content, want) {
		t.Errorf("expected escaped title in content, got %s", note.Content)
	}
}

func TestNoteForImage_HasAttachment(t *testing.T) {
	s, _ := testService()
	img := &data.Image{ID: 7, Title: "Daily Kitten", URL: "https://cataas.com/cat/abc.jpg", Timestamp: time.Now()}

	note := s.NoteForImage(img)
	if len(note.Attachment) != 1 {
		t.Fatalf("expected one attachment, got %d", len(note.Attachment))
	}
	if note.Attachment[0].URL != img.URL {
		t.Errorf("unexpected attachment URL: %s", note.Attachment[0].URL)
	}
	if note.Attachment[0].MediaType != "image/jpeg" {
		t.Errorf("unexpected media type: %s", note.Attachment[0].MediaType)
	}
}

func TestPublishNote_DedupesBySharedInbox(t *testing.T) {
	s, store := testService()
	ctx := context.Background()

	shared := "https://mastodon.example/inbox"
	store.followers = []data.ActivityPubFollower{
		{ActorURI: "https://mastodon.example/users/a", InboxURL: "https://mastodon.example/users/a/inbox", SharedInbox: &shared},
		{ActorURI: "https://mastodon.example/users/b", InboxURL: "https://mastodon.example/users/b/inbox", SharedInbox: &shared},
		{ActorURI: "https://other.example/users/c", InboxURL: "https://other.example/users/c/inbox"},
	}

	note := s.NoteForQuote(&data.Quote{ID: 1, Quote: "hello", Timestamp: time.Now()})
	s.PublishNote(ctx, note)

	if len(store.deliveries) != 2 {
		t.Fatalf("expected 2 deliveries (deduped shared inbox), got %d", len(store.deliveries))
	}
}

func TestPublishNote_DisabledIsNoop(t *testing.T) {
	s, store := testService()
	s.Config.ActivityPub.Enabled = false
	store.followers = []data.ActivityPubFollower{{ActorURI: "a", InboxURL: "https://x.example/inbox"}}

	s.PublishNote(context.Background(), s.NoteForQuote(&data.Quote{ID: 1, Quote: "hi"}))
	if len(store.deliveries) != 0 {
		t.Fatalf("expected no deliveries when disabled, got %d", len(store.deliveries))
	}
}

func TestSignRequest_VerifiesWithOwnPublicKey(t *testing.T) {
	s, _ := testService()
	ctx := context.Background()

	key, err := s.getOrCreateKey(ctx)
	if err != nil {
		t.Fatalf("getOrCreateKey: %v", err)
	}
	pub, err := parsePublicKeyPEM(key.PublicKey)
	if err != nil {
		t.Fatalf("parsePublicKeyPEM: %v", err)
	}

	body := []byte(`{"type":"Create"}`)
	req, err := http.NewRequest(http.MethodPost, "https://follower.example/inbox", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if err := s.signRequest(ctx, req, body); err != nil {
		t.Fatalf("signRequest: %v", err)
	}
	if req.Header.Get("Signature") == "" {
		t.Fatal("expected a Signature header to be set")
	}
	if req.Header.Get("Digest") == "" {
		t.Fatal("expected a Digest header to be set")
	}

	verifier, err := httpsig.NewVerifier(req)
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	if verifier.KeyId() != s.publicKeyID() {
		t.Errorf("unexpected key id: %s", verifier.KeyId())
	}
	if err := verifier.Verify(pub, httpsig.RSA_SHA256); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestProcessDueDeliveries_MarksFailureWithBackoff(t *testing.T) {
	s, store := testService()
	ctx := context.Background()

	// safeurl blocks loopback/private targets by default (SSRF protection),
	// so any inbox reachable in a unit test will fail delivery. That's the
	// behavior under test: failures are recorded with an incremented
	// attempt count and a future retry time.
	store.deliveries = []data.ActivityPubDelivery{
		{ID: 1, InboxURL: "http://127.0.0.1:1/inbox", Payload: `{"type":"Create"}`, Status: "pending", NextAttempt: time.Now()},
	}
	store.nextID = 2

	s.ProcessDueDeliveries(ctx)

	if len(store.deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(store.deliveries))
	}
	d := store.deliveries[0]
	if d.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", d.Attempts)
	}
	if d.Status != "pending" {
		t.Errorf("expected still pending (not given up), got %s", d.Status)
	}
	if !d.NextAttempt.After(time.Now()) {
		t.Error("expected next attempt to be scheduled in the future (backoff)")
	}
	if d.LastError == nil || *d.LastError == "" {
		t.Error("expected an error message to be recorded")
	}
}

func TestProcessDueDeliveries_GivesUpAfterMaxAttempts(t *testing.T) {
	s, store := testService()
	ctx := context.Background()

	store.deliveries = []data.ActivityPubDelivery{
		{ID: 1, InboxURL: "http://127.0.0.1:1/inbox", Payload: `{"type":"Create"}`, Status: "pending", Attempts: maxDeliveryAttempts - 1, NextAttempt: time.Now()},
	}
	store.nextID = 2

	s.ProcessDueDeliveries(ctx)

	if store.deliveries[0].Status != "failed" {
		t.Errorf("expected delivery to be marked failed after reaching max attempts, got %s", store.deliveries[0].Status)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

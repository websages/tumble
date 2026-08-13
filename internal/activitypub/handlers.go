package activitypub

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"tumble/internal/data"
)

const activityContentType = "application/activity+json"

// RegisterRoutes wires the ActivityPub discovery/inbox endpoints into mux.
// It's a no-op if federation is disabled.
func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	if !s.Enabled() {
		return
	}
	mux.HandleFunc("/.well-known/webfinger", s.WebfingerHandler)
	mux.HandleFunc("/activitypub/actor", s.ActorHandler)
	mux.HandleFunc("/activitypub/followers", s.FollowersHandler)
	mux.HandleFunc("/activitypub/outbox", s.OutboxHandler)
	mux.HandleFunc("/activitypub/inbox", s.InboxHandler)
}

func writeJSON(w http.ResponseWriter, contentType string, v any) {
	w.Header().Set("Content-Type", contentType)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func (s *Service) ActorHandler(w http.ResponseWriter, r *http.Request) {
	actor, err := s.BuildActor(r.Context())
	if err != nil {
		slog.Error("ActivityPub: failed to build actor", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, activityContentType, actor)
}

func (s *Service) WebfingerHandler(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	doc := s.BuildWebfinger(resource)
	if doc == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, "application/jrd+json", doc)
}

func (s *Service) FollowersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pageParam := r.URL.Query().Get("page")
	if pageParam == "" {
		col, err := s.BuildFollowersCollection(ctx)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, activityContentType, col)
		return
	}

	page, err := strconv.Atoi(pageParam)
	if err != nil || page < 1 {
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return
	}
	p, err := s.BuildFollowersPage(ctx, page)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, activityContentType, p)
}

// OutboxHandler serves a minimal, unpaginated outbox: it advertises the
// total published item count. Followers receive posts via inbox delivery,
// so a full paginated history isn't required for the publish-only scope.
func (s *Service) OutboxHandler(w http.ResponseWriter, r *http.Request) {
	linkCount, err := s.Store.CountIRCLinks(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	quoteCount, err := s.Store.CountQuotes(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	col := s.BuildOutboxCollection(r.Context(), int(linkCount+quoteCount))
	writeJSON(w, activityContentType, col)
}

// InboxHandler accepts Follow requests (persisting the follower and replying
// with Accept) and Undo(Follow) requests (removing the follower). Any other
// activity type is acknowledged but otherwise ignored, since this site only
// publishes.
func (s *Service) InboxHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var incoming IncomingActivity
	if err := json.Unmarshal(body, &incoming); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	remoteActor, err := s.verifyInboundSignature(r)
	if err != nil {
		slog.Warn("ActivityPub: inbox signature verification failed", "error", err, "type", incoming.Type)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if remoteActor.ID != incoming.Actor {
		slog.Warn("ActivityPub: signing actor does not match activity actor", "signer", remoteActor.ID, "actor", incoming.Actor)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	switch incoming.Type {
	case "Follow":
		s.handleFollow(r, remoteActor, incoming)
	case "Undo":
		s.handleUndo(r, remoteActor)
	default:
		// Not part of the publish-only scope (Like, Announce, replies, ...).
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Service) handleFollow(r *http.Request, remoteActor *Actor, incoming IncomingActivity) {
	ctx := r.Context()

	follower := &data.ActivityPubFollower{
		ActorURI: remoteActor.ID,
		InboxURL: remoteActor.Inbox,
	}
	if remoteActor.Endpoints != nil && remoteActor.Endpoints.SharedInbox != "" {
		shared := remoteActor.Endpoints.SharedInbox
		follower.SharedInbox = &shared
	}

	if err := s.Store.UpsertActivityPubFollower(ctx, follower); err != nil {
		slog.Error("ActivityPub: failed to save follower", "actor", remoteActor.ID, "error", err)
		return
	}

	accept := &Activity{
		Context: ContextURL,
		ID:      s.actorID() + "/accepts/" + remoteActor.ID,
		Type:    "Accept",
		Actor:   s.actorID(),
		Object:  incoming,
	}
	if err := s.deliverActivity(ctx, remoteActor.Inbox, accept); err != nil {
		slog.Error("ActivityPub: failed to deliver Accept", "actor", remoteActor.ID, "error", err)
	}
}

func (s *Service) handleUndo(r *http.Request, remoteActor *Actor) {
	if err := s.Store.DeleteActivityPubFollowerByActorURI(r.Context(), remoteActor.ID); err != nil {
		slog.Error("ActivityPub: failed to remove follower", "actor", remoteActor.ID, "error", err)
	}
}

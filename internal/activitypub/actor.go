package activitypub

import (
	"context"
	"fmt"
)

// BuildActor constructs the site's Actor document.
func (s *Service) BuildActor(ctx context.Context) (*Actor, error) {
	key, err := s.getOrCreateKey(ctx)
	if err != nil {
		return nil, err
	}

	name := s.Config.SiteTitle
	if name == "" {
		name = s.Config.SiteName
	}

	actor := &Actor{
		Context:                   []string{ContextURL, SecurityContextURL},
		ID:                        s.actorID(),
		Type:                      "Service",
		PreferredUsername:         s.Config.ActivityPub.ActorName,
		Name:                      name,
		Summary:                   s.Config.SiteDescription,
		Inbox:                     s.inboxURL(),
		Outbox:                    s.outboxURL(),
		Followers:                 s.followersURL(),
		URL:                       s.Config.BaseURL,
		ManuallyApprovesFollowers: false,
		PublicKey: PublicKey{
			ID:           s.publicKeyID(),
			Owner:        s.actorID(),
			PublicKeyPem: key.PublicKey,
		},
	}

	if avatar := s.Config.ActivityPub.AvatarURL; avatar != "" {
		actor.Icon = &Image{
			Type:      "Image",
			MediaType: guessImageMediaType(avatar),
			URL:       avatar,
		}
	}

	return actor, nil
}

// BuildWebfinger constructs the JRD document for a webfinger lookup of the
// site actor. It returns nil if the resource doesn't match the site actor.
func (s *Service) BuildWebfinger(resource string) *WebfingerResponse {
	if resource != s.acct() {
		return nil
	}
	return &WebfingerResponse{
		Subject: s.acct(),
		Links: []WebfingerLink{
			{Rel: "self", Type: "application/activity+json", Href: s.actorID()},
		},
	}
}

// BuildFollowersCollection returns the root (unpaginated) followers collection.
func (s *Service) BuildFollowersCollection(ctx context.Context) (*OrderedCollection, error) {
	count, err := s.Store.CountActivityPubFollowers(ctx)
	if err != nil {
		return nil, err
	}
	col := &OrderedCollection{
		Context:    ContextURL,
		ID:         s.followersURL(),
		Type:       "OrderedCollection",
		TotalItems: int(count),
	}
	if count > 0 {
		col.First = s.followersURL() + "?page=1"
	}
	return col, nil
}

const collectionPageSize = 50

// BuildFollowersPage returns a single page of the followers collection.
func (s *Service) BuildFollowersPage(ctx context.Context, page int) (*OrderedCollectionPage, error) {
	if page < 1 {
		page = 1
	}
	followers, err := s.Store.ListActivityPubFollowers(ctx, collectionPageSize, (page-1)*collectionPageSize)
	if err != nil {
		return nil, err
	}
	items := make([]any, len(followers))
	for i, f := range followers {
		items[i] = f.ActorURI
	}
	p := &OrderedCollectionPage{
		Context:      ContextURL,
		ID:           fmt.Sprintf("%s?page=%d", s.followersURL(), page),
		Type:         "OrderedCollectionPage",
		PartOf:       s.followersURL(),
		OrderedItems: items,
	}
	if len(followers) == collectionPageSize {
		p.Next = fmt.Sprintf("%s?page=%d", s.followersURL(), page+1)
	}
	return p, nil
}

// BuildOutboxCollection returns the root (unpaginated) outbox collection.
func (s *Service) BuildOutboxCollection(ctx context.Context, totalItems int) *OrderedCollection {
	col := &OrderedCollection{
		Context:    ContextURL,
		ID:         s.outboxURL(),
		Type:       "OrderedCollection",
		TotalItems: totalItems,
	}
	if totalItems > 0 {
		col.First = s.outboxURL() + "?page=1"
	}
	return col
}

// WrapCreate wraps a Note in a Create activity for delivery/outbox display.
func (s *Service) WrapCreate(note *Note) *Activity {
	return &Activity{
		Context:   ContextURL,
		ID:        note.ID + "#create",
		Type:      "Create",
		Actor:     s.actorID(),
		Published: note.Published,
		To:        []string{PublicCollection},
		Object:    note,
	}
}

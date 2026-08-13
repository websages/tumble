// Package activitypub implements publish-only ActivityPub federation for
// Tumble: a single site-wide actor that remote Mastodon-like servers can
// discover and follow, receiving newly published links, quotes, and images
// as Create(Note) activities.
package activitypub

import (
	"net/url"
	"strings"

	"tumble/internal/config"
	"tumble/internal/data"
)

// Service holds the dependencies needed to build ActivityPub documents,
// handle inbox traffic, and deliver activities to followers.
type Service struct {
	Store  data.Store
	Config *config.Config
}

func NewService(cfg *config.Config, store data.Store) *Service {
	return &Service{Store: store, Config: cfg}
}

// Enabled reports whether ActivityPub federation is turned on in config.
func (s *Service) Enabled() bool {
	return s != nil && s.Config.ActivityPub.Enabled
}

func (s *Service) baseURL() string {
	return strings.TrimRight(s.Config.BaseURL, "/")
}

func (s *Service) host() string {
	u, err := url.Parse(s.Config.BaseURL)
	if err != nil {
		return s.Config.BaseURL
	}
	return u.Host
}

func (s *Service) actorID() string {
	return s.baseURL() + "/activitypub/actor"
}

func (s *Service) publicKeyID() string {
	return s.actorID() + "#main-key"
}

func (s *Service) inboxURL() string {
	return s.baseURL() + "/activitypub/inbox"
}

func (s *Service) outboxURL() string {
	return s.baseURL() + "/activitypub/outbox"
}

func (s *Service) followersURL() string {
	return s.baseURL() + "/activitypub/followers"
}

// acct is the webfinger "acct:name@host" identifier for the site actor.
func (s *Service) acct() string {
	return "acct:" + s.Config.ActivityPub.ActorName + "@" + s.host()
}

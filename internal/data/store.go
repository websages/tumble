package data

import (
	"context"
	"time"
)

type IRCLink struct {
	ID          int       `json:"ircLinkID"`
	Timestamp   time.Time `json:"timestamp"`
	User        string    `json:"user"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Clicks      int       `json:"clicks"`
	ContentType string    `json:"content_type"`
}

type Image struct {
	ID        int       `json:"imageID"`
	Timestamp time.Time `json:"timestamp"`
	Title     string    `json:"title"`
	Link      string    `json:"link"`
	URL       string    `json:"url"`
	MD5Sum    string    `json:"md5sum"`
}

type Quote struct {
	ID        int       `json:"quoteID"`
	Timestamp time.Time `json:"timestamp"`
	Quote     string    `json:"quote"`
	Author    string    `json:"author"`
}

type UserStat struct {
	User       string `json:"user"`
	LinkCount  int    `json:"link_count"`
	QuoteCount int    `json:"quote_count"`
}

type TimelineItem struct {
	Type      string    `json:"type"` // "link", "quote", or "image"
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Title     string    `json:"title"`   // For links and images
	URL       string    `json:"url"`     // For links and images
	Content   string    `json:"content"` // For quotes
	Author    string    `json:"author"`  // For quotes (and links/images as User)
	MD5Sum    string    `json:"md5sum"`  // For images
}

type Store interface {
	GetRecentIRCLinks(ctx context.Context, days int, offsetDays int) ([]IRCLink, error)
	GetRecentImages(ctx context.Context, days int, offsetDays int) ([]Image, error)
	GetRecentQuotes(ctx context.Context, days int, offsetDays int) ([]Quote, error)

	SearchIRCLinks(ctx context.Context, query string) ([]IRCLink, error)
	GetTopIRCLinks(ctx context.Context, startDays int, endDays int, limit int) ([]IRCLink, error)
	GetIRCLinkByID(ctx context.Context, id int) (*IRCLink, error)
	GetIRCLinkURL(ctx context.Context, id int) (string, error)
	IncrementClicks(ctx context.Context, id int) error
	InsertIRCLink(ctx context.Context, user, title, url, contentType string) (int, error)
	InsertQuote(ctx context.Context, quote, author string) error
	GetRandomQuote(ctx context.Context) (*Quote, error)

	// Stats
	GetUserStats(ctx context.Context, sortBy string, limit int, offset int) ([]UserStat, error)
	GetLinksByUser(ctx context.Context, user string, limit int, offset int) ([]IRCLink, error)
	GetUserTimeline(ctx context.Context, user string, filterType string, limit int, offset int) ([]TimelineItem, error)
	GetGlobalTimeline(ctx context.Context, limit int, offset int) ([]TimelineItem, error)

	Bootstrap(ctx context.Context) error

	Close() error
}

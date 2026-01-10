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

	Bootstrap(ctx context.Context) error

	Close() error
}

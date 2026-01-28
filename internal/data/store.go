package data

import (
	"context"
	"time"
)

type IRCLink struct {
	ID          int       `json:"ircLinkID" gorm:"column:ircLinkID;primaryKey"`
	Timestamp   time.Time `json:"timestamp" gorm:"column:timestamp"`
	User        string    `json:"user" gorm:"column:user;index"`
	Title       string    `json:"title" gorm:"column:title"`
	URL         string    `json:"url" gorm:"column:url"`
	Clicks      int       `json:"clicks" gorm:"column:clicks;default:0"`
	ContentType string    `json:"content_type" gorm:"column:content_type"`
}

// TableName overrides the table name used by User to `ircLink`
func (IRCLink) TableName() string {
	return "ircLink"
}

type Image struct {
	ID        int       `json:"imageID" gorm:"column:imageID;primaryKey"`
	Timestamp time.Time `json:"timestamp" gorm:"column:timestamp"`
	Title     string    `json:"title" gorm:"column:title"`
	Link      string    `json:"link" gorm:"column:link"`
	URL       string    `json:"url" gorm:"column:url"`
	MD5Sum    string    `json:"md5sum" gorm:"column:md5sum"`
}

// TableName overrides the table name used by User to `image`
func (Image) TableName() string {
	return "image"
}

type Quote struct {
	ID        int       `json:"quoteID" gorm:"column:quoteID;primaryKey"`
	Timestamp time.Time `json:"timestamp" gorm:"column:timestamp"`
	Quote     string    `json:"quote" gorm:"column:quote"`
	Author    string    `json:"author" gorm:"column:author;type:varchar(255);index"`
}

// TableName overrides the table name used by User to `quote`
func (Quote) TableName() string {
	return "quote"
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

type LinkPreview struct {
	URL       string    `json:"url" gorm:"column:url;primaryKey"`
	Data      []byte    `json:"data" gorm:"column:data;type:text"` // JSON blob of the map[string]string metadata
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// TableName overrides the table name to `link_previews`
func (LinkPreview) TableName() string {
	return "link_previews"
}

type Store interface {
	GetRecentIRCLinks(ctx context.Context, days int, offsetDays int) ([]IRCLink, error)
	GetRecentImages(ctx context.Context, days int, offsetDays int) ([]Image, error)
	GetRecentQuotes(ctx context.Context, days int, offsetDays int) ([]Quote, error)

	SearchIRCLinks(ctx context.Context, query string) ([]IRCLink, error)
	GetTopIRCLinks(ctx context.Context, startDays int, endDays int, limit int) ([]IRCLink, error)
	GetIRCLinkByID(ctx context.Context, id int) (*IRCLink, error)
	GetIRCLinkURL(ctx context.Context, id int) (string, error)
	GetIRCLinksByURL(ctx context.Context, url string) ([]IRCLink, error)
	IncrementClicks(ctx context.Context, id int) error
	InsertIRCLink(ctx context.Context, user, title, url, contentType string) (int, error)
	DeleteIRCLink(ctx context.Context, id int) error
	InsertQuote(ctx context.Context, quote, author string) error
	GetRandomQuote(ctx context.Context) (*Quote, error)

	// Stats
	GetUserStats(ctx context.Context, sortBy string, limit int, offset int) ([]UserStat, error)
	GetLinksByUser(ctx context.Context, user string, limit int, offset int) ([]IRCLink, error)
	GetUserTimeline(ctx context.Context, user string, filterType string, limit int, offset int) ([]TimelineItem, error)
	GetGlobalTimeline(ctx context.Context, limit int, offset int) ([]TimelineItem, error)

	// Caching
	GetLinkPreview(ctx context.Context, url string) (*LinkPreview, error)
	InsertLinkPreview(ctx context.Context, url string, data []byte) error
	DeleteLinkPreview(ctx context.Context, url string) error

	Bootstrap(ctx context.Context) error

	Close() error
}

package data

import (
	"context"
	"fmt"
	"time"
)

type IRCLink struct {
	ID             int       `json:"ircLinkID" gorm:"column:ircLinkID;primaryKey"`
	Timestamp      time.Time `json:"timestamp" gorm:"column:timestamp"`
	User           string    `json:"user" gorm:"column:user;index"`
	Title          string    `json:"title" gorm:"column:title"`
	URL            string    `json:"url" gorm:"column:url"`
	Clicks         int       `json:"clicks" gorm:"column:clicks;default:0"`
	ContentType    string    `json:"content_type" gorm:"column:content_type"`
	ClientType     *string   `json:"client_type,omitempty" gorm:"column:client_type;type:varchar(50);index:idx_link_client,priority:1"`
	ClientNetwork  *string   `json:"client_network,omitempty" gorm:"column:client_network;type:varchar(255);index:idx_link_client,priority:2"`
	ClientChannel  *string   `json:"client_channel,omitempty" gorm:"column:client_channel;type:varchar(255);index:idx_link_client,priority:3"`
	ClientUserID   *string   `json:"client_user_id,omitempty" gorm:"column:client_user_id;type:varchar(255)"`
	ClientUserName *string   `json:"client_user_name,omitempty" gorm:"column:client_user_name;type:varchar(255)"`
}

// TableName overrides the table name used by User to `ircLink`
func (IRCLink) TableName() string {
	return "ircLink"
}

type Image struct {
	ID             int       `json:"imageID" gorm:"column:imageID;primaryKey"`
	Timestamp      time.Time `json:"timestamp" gorm:"column:timestamp"`
	Title          string    `json:"title" gorm:"column:title"`
	Link           string    `json:"link" gorm:"column:link"`
	URL            string    `json:"url" gorm:"column:url"`
	MD5Sum         string    `json:"md5sum" gorm:"column:md5sum"`
	ClientType     *string   `json:"client_type,omitempty" gorm:"column:client_type;type:varchar(50);index:idx_image_client,priority:1"`
	ClientNetwork  *string   `json:"client_network,omitempty" gorm:"column:client_network;type:varchar(255);index:idx_image_client,priority:2"`
	ClientChannel  *string   `json:"client_channel,omitempty" gorm:"column:client_channel;type:varchar(255);index:idx_image_client,priority:3"`
	ClientUserID   *string   `json:"client_user_id,omitempty" gorm:"column:client_user_id;type:varchar(255)"`
	ClientUserName *string   `json:"client_user_name,omitempty" gorm:"column:client_user_name;type:varchar(255)"`
}

// TableName overrides the table name used by User to `image`
func (Image) TableName() string {
	return "image"
}

type Quote struct {
	ID             int       `json:"quoteID" gorm:"column:quoteID;primaryKey"`
	Timestamp      time.Time `json:"timestamp" gorm:"column:timestamp"`
	Quote          string    `json:"quote" gorm:"column:quote"`
	Author         string    `json:"author" gorm:"column:author;type:varchar(255);index"`
	Poster         string    `json:"poster,omitempty" gorm:"column:poster;type:varchar(255);index"`
	ClientType     *string   `json:"client_type,omitempty" gorm:"column:client_type;type:varchar(50);index:idx_quote_client,priority:1"`
	ClientNetwork  *string   `json:"client_network,omitempty" gorm:"column:client_network;type:varchar(255);index:idx_quote_client,priority:2"`
	ClientChannel  *string   `json:"client_channel,omitempty" gorm:"column:client_channel;type:varchar(255);index:idx_quote_client,priority:3"`
	ClientUserID   *string   `json:"client_user_id,omitempty" gorm:"column:client_user_id;type:varchar(255)"`
	ClientUserName *string   `json:"client_user_name,omitempty" gorm:"column:client_user_name;type:varchar(255)"`
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
	Type           string    `json:"type"` // "link", "quote", or "image"
	ID             int       `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	Title          string    `json:"title"`                                  // For links and images
	URL            string    `json:"url"`                                    // For links and images
	Content        string    `json:"content"`                                // For quotes
	Author         string    `json:"author"`                                 // For quotes (and links/images as User)
	MD5Sum         string    `json:"md5sum"`                                 // For images
	ContentType    string    `json:"contentType" gorm:"column:content_type"` // For links (to detect images)
	ClientType     *string   `json:"client_type,omitempty"`
	ClientNetwork  *string   `json:"client_network,omitempty"`
	ClientChannel  *string   `json:"client_channel,omitempty"`
	ClientUserID   *string   `json:"client_user_id,omitempty"`
	ClientUserName *string   `json:"client_user_name,omitempty"`
}

// ValidClientTypes is the set of allowed client_type values.
var ValidClientTypes = map[string]bool{
	"irc":     true,
	"slack":   true,
	"discord": true,
	"api":     true,
	"web":     true,
}

type ClientFilter struct {
	ClientType    *string
	ClientNetwork *string
	ClientChannel *string
}

func (f ClientFilter) IsEmpty() bool {
	return f.ClientType == nil && f.ClientNetwork == nil && f.ClientChannel == nil
}

// Validate checks that hierarchical filter dependencies are satisfied.
// client_network requires client_type, and client_channel requires both.
func (f ClientFilter) Validate() error {
	if f.ClientType != nil && !ValidClientTypes[*f.ClientType] {
		return fmt.Errorf("invalid client_type: must be one of irc, slack, discord, api, web")
	}
	if f.ClientNetwork != nil && f.ClientType == nil {
		return fmt.Errorf("client_network requires client_type")
	}
	if f.ClientChannel != nil && (f.ClientType == nil || f.ClientNetwork == nil) {
		return fmt.Errorf("client_channel requires client_type and client_network")
	}
	return nil
}

type Tag struct {
	ID           int       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Tag          string    `json:"tag" gorm:"column:tag;type:varchar(255);index"`
	ResourceType string    `json:"resource_type" gorm:"column:resource_type;type:varchar(50);index:idx_tag_resource"`
	ResourceID   int       `json:"resource_id" gorm:"column:resource_id;index:idx_tag_resource"`
	CreatedBy    string    `json:"created_by" gorm:"column:created_by;type:varchar(255)"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

// TableName overrides the table name to `tags`
func (Tag) TableName() string {
	return "tags"
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

type ArchiveLookup struct {
	URL        string     `json:"url" gorm:"column:url;primaryKey"`
	ArchiveURL *string    `json:"archive_url" gorm:"column:archive_url"`
	SnapshotAt *time.Time `json:"snapshot_at" gorm:"column:snapshot_at"`
	Status     string     `json:"status" gorm:"column:status;index"`
	CheckedAt  time.Time  `json:"checked_at" gorm:"column:checked_at"`
}

func (ArchiveLookup) TableName() string {
	return "archive_lookups"
}

// ActivityPubKey stores the RSA keypair used to sign outgoing ActivityPub
// activities on behalf of the site-wide actor. Only one row is ever used.
type ActivityPubKey struct {
	ID         int       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PrivateKey string    `json:"-" gorm:"column:private_key;type:text"`
	PublicKey  string    `json:"public_key" gorm:"column:public_key;type:text"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

// TableName overrides the table name to `activitypub_key`
func (ActivityPubKey) TableName() string {
	return "activitypub_key"
}

// ActivityPubFollower represents a remote actor following the site.
type ActivityPubFollower struct {
	ID          int       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ActorURI    string    `json:"actor_uri" gorm:"column:actor_uri;type:varchar(500);uniqueIndex"`
	InboxURL    string    `json:"inbox_url" gorm:"column:inbox_url;type:varchar(500)"`
	SharedInbox *string   `json:"shared_inbox,omitempty" gorm:"column:shared_inbox;type:varchar(500)"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

// TableName overrides the table name to `activitypub_follower`
func (ActivityPubFollower) TableName() string {
	return "activitypub_follower"
}

// ActivityPubDelivery is a queued outbound activity delivery to a single inbox.
type ActivityPubDelivery struct {
	ID          int       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	InboxURL    string    `json:"inbox_url" gorm:"column:inbox_url;type:varchar(500);index"`
	Payload     string    `json:"payload" gorm:"column:payload;type:text"`
	Status      string    `json:"status" gorm:"column:status;type:varchar(20);index;default:pending"` // pending, sent, failed
	Attempts    int       `json:"attempts" gorm:"column:attempts;default:0"`
	NextAttempt time.Time `json:"next_attempt" gorm:"column:next_attempt;index"`
	LastError   *string   `json:"last_error,omitempty" gorm:"column:last_error;type:text"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

// TableName overrides the table name to `activitypub_delivery`
func (ActivityPubDelivery) TableName() string {
	return "activitypub_delivery"
}

type Store interface {
	GetRecentIRCLinks(ctx context.Context, days int, offsetDays int, filter ClientFilter) ([]IRCLink, error)
	GetRecentImages(ctx context.Context, days int, offsetDays int, filter ClientFilter) ([]Image, error)
	GetRecentQuotes(ctx context.Context, days int, offsetDays int, filter ClientFilter) ([]Quote, error)

	SearchIRCLinks(ctx context.Context, query string, filter ClientFilter) ([]IRCLink, error)
	SearchQuotes(ctx context.Context, query string, filter ClientFilter) ([]Quote, error)
	GetTopIRCLinks(ctx context.Context, startDays int, endDays int, limit int) ([]IRCLink, error)
	GetIRCLinkByID(ctx context.Context, id int) (*IRCLink, error)
	GetIRCLinkURL(ctx context.Context, id int) (string, error)
	GetIRCLinksByURL(ctx context.Context, url string, filter ClientFilter) ([]IRCLink, error)
	IncrementClicks(ctx context.Context, id int) error
	InsertIRCLink(ctx context.Context, link *IRCLink) (int, error)
	DeleteIRCLink(ctx context.Context, id int) error
	InsertQuote(ctx context.Context, quote *Quote) (int, error)
	GetRandomQuote(ctx context.Context) (*Quote, error)
	GetQuoteByID(ctx context.Context, id int) (*Quote, error)
	DeleteQuote(ctx context.Context, id int) error

	// Stats
	CountIRCLinks(ctx context.Context) (int64, error)
	CountQuotes(ctx context.Context) (int64, error)
	GetUserStats(ctx context.Context, sortBy string, limit int, offset int) ([]UserStat, error)
	GetLinksByUser(ctx context.Context, user string, limit int, offset int) ([]IRCLink, error)
	GetUserTimeline(ctx context.Context, user string, filterType string, limit int, offset int) ([]TimelineItem, error)
	GetGlobalTimeline(ctx context.Context, limit int, offset int) ([]TimelineItem, error)
	GetLinksByPopularity(ctx context.Context, limit int, offset int) ([]IRCLink, error)

	// Caching
	GetLinkPreview(ctx context.Context, url string) (*LinkPreview, error)
	InsertLinkPreview(ctx context.Context, url string, data []byte) error
	DeleteLinkPreview(ctx context.Context, url string) error
	DeleteAllLinkPreviews(ctx context.Context) (int, error)

	// Archive lookups
	GetArchiveLookup(ctx context.Context, url string) (*ArchiveLookup, error)
	UpsertArchiveLookup(ctx context.Context, lookup *ArchiveLookup) error
	GetUncheckedDeadLinkURLs(ctx context.Context) ([]string, error)
	GetStaleArchiveLookups(ctx context.Context, status string, recheckAfter time.Duration) ([]string, error)

	// Tag operations
	CreateTag(ctx context.Context, tag Tag) (*Tag, error)
	GetTagsByResource(ctx context.Context, resourceType string, resourceID int) ([]Tag, error)
	GetTagByID(ctx context.Context, id int) (*Tag, error)
	DeleteTag(ctx context.Context, id int) error
	DeleteTagsByResource(ctx context.Context, resourceType string, resourceID int) error

	// Image operations
	InsertImage(ctx context.Context, image *Image) (int, error)
	GetImageByID(ctx context.Context, id int) (*Image, error)
	GetTodayImageByLink(ctx context.Context, link string) (*Image, error)
	DeleteTodayImageByLink(ctx context.Context, link string) error

	// ActivityPub
	GetActivityPubKey(ctx context.Context) (*ActivityPubKey, error)
	InsertActivityPubKey(ctx context.Context, key *ActivityPubKey) error
	UpsertActivityPubFollower(ctx context.Context, follower *ActivityPubFollower) error
	DeleteActivityPubFollowerByActorURI(ctx context.Context, actorURI string) error
	ListActivityPubFollowers(ctx context.Context, limit int, offset int) ([]ActivityPubFollower, error)
	CountActivityPubFollowers(ctx context.Context) (int64, error)
	ListActivityPubFollowerInboxes(ctx context.Context) ([]string, error)
	EnqueueActivityPubDelivery(ctx context.Context, inboxURL string, payload string) error
	GetDueActivityPubDeliveries(ctx context.Context, limit int) ([]ActivityPubDelivery, error)
	MarkActivityPubDeliverySucceeded(ctx context.Context, id int) error
	MarkActivityPubDeliveryFailed(ctx context.Context, id int, errMsg string, nextAttempt time.Time, giveUp bool) error

	Bootstrap(ctx context.Context) error

	Close() error
}

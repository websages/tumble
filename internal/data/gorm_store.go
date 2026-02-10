package data

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

func (s *GormStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *GormStore) Bootstrap(ctx context.Context) error {
	return s.db.AutoMigrate(&IRCLink{}, &Image{}, &Quote{}, &LinkPreview{}, &Tag{})
}

func (s *GormStore) GetRecentIRCLinks(ctx context.Context, startDays int, endDays int) ([]IRCLink, error) {
	var links []IRCLink
	// timestamp >= NOW() - startDays AND timestamp <= NOW() - endDays
	// Note: startDays is "further back" (larger number), endDays is "closer" (smaller number)
	now := time.Now()
	startDate := now.AddDate(0, 0, -startDays)
	endDate := now.AddDate(0, 0, -endDays)

	err := s.db.WithContext(ctx).
		Where("timestamp >= ? AND timestamp <= ?", startDate, endDate).
		Order("timestamp DESC").
		Find(&links).Error
	return links, err
}

func (s *GormStore) GetRecentImages(ctx context.Context, startDays int, endDays int) ([]Image, error) {
	var images []Image
	now := time.Now()
	startDate := now.AddDate(0, 0, -startDays)
	endDate := now.AddDate(0, 0, -endDays)

	err := s.db.WithContext(ctx).
		Where("timestamp >= ? AND timestamp <= ?", startDate, endDate).
		Order("timestamp DESC").
		Find(&images).Error
	return images, err
}

func (s *GormStore) InsertImage(ctx context.Context, title, link, url string) (int, error) {
	img := Image{
		Title:     title,
		Link:      link,
		URL:       url,
		Timestamp: time.Now(),
	}
	err := s.db.WithContext(ctx).Create(&img).Error
	return img.ID, err
}

func (s *GormStore) GetTodayImageByLink(ctx context.Context, link string) (*Image, error) {
	var img Image
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	err := s.db.WithContext(ctx).
		Where("link = ? AND timestamp >= ? AND timestamp < ?", link, startOfDay, endOfDay).
		First(&img).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &img, nil
}

func (s *GormStore) DeleteTodayImageByLink(ctx context.Context, link string) error {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	return s.db.WithContext(ctx).
		Where("link = ? AND timestamp >= ? AND timestamp < ?", link, startOfDay, endOfDay).
		Delete(&Image{}).Error
}

func (s *GormStore) GetRecentQuotes(ctx context.Context, startDays int, endDays int) ([]Quote, error) {
	var quotes []Quote
	now := time.Now()
	startDate := now.AddDate(0, 0, -startDays)
	endDate := now.AddDate(0, 0, -endDays)

	err := s.db.WithContext(ctx).
		Where("timestamp >= ? AND timestamp <= ?", startDate, endDate).
		Order("timestamp DESC").
		Find(&quotes).Error
	return quotes, err
}

func (s *GormStore) SearchIRCLinks(ctx context.Context, query string) ([]IRCLink, error) {
	var links []IRCLink
	// Simple LIKE search for cross-db compatibility
	term := "%" + query + "%"
	// Exclude links with cached error previews within the 180-day TTL.
	// CAST(data AS TEXT) is required because glebarez/sqlite stores []byte
	// as BLOB, and SQLite's LIKE doesn't match text patterns against BLOBs.
	// The error LIKE pattern is inlined to avoid glebarez double-quote escaping.
	errorCutoff := time.Now().Add(-180 * 24 * time.Hour)
	err := s.db.WithContext(ctx).
		Where(`(title LIKE ? OR url LIKE ? OR ircLinkID IN (SELECT resource_id FROM tags WHERE resource_type = 'link' AND tag LIKE ?)) AND url NOT IN (SELECT url FROM link_previews WHERE CAST(data AS TEXT) LIKE '%"error":%' AND updated_at > ?)`, term, term, term, errorCutoff).
		Order("clicks DESC").
		Limit(50).
		Find(&links).Error
	return links, err
}

func (s *GormStore) SearchQuotes(ctx context.Context, query string) ([]Quote, error) {
	var quotes []Quote
	// Simple LIKE search for cross-db compatibility
	term := "%" + query + "%"
	err := s.db.WithContext(ctx).
		Where("quote LIKE ? OR author LIKE ? OR quoteID IN (SELECT resource_id FROM tags WHERE resource_type = 'quote' AND tag LIKE ?)", term, term, term).
		Order("timestamp DESC").
		Limit(50).
		Find(&quotes).Error
	return quotes, err
}

func (s *GormStore) GetTopIRCLinks(ctx context.Context, startDays int, endDays int, limit int) ([]IRCLink, error) {
	var links []IRCLink
	now := time.Now()
	startDate := now.AddDate(0, 0, -startDays)
	endDate := now.AddDate(0, 0, -endDays)

	err := s.db.WithContext(ctx).
		Where("timestamp >= ? AND timestamp <= ?", startDate, endDate).
		Order("clicks DESC").
		Limit(limit).
		Find(&links).Error
	return links, err
}

func (s *GormStore) GetIRCLinkByID(ctx context.Context, id int) (*IRCLink, error) {
	var link IRCLink
	err := s.db.WithContext(ctx).First(&link, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Or specific error? Store interface implication seems to be nil on not found or error
		}
		return nil, err
	}
	return &link, nil
}

func (s *GormStore) GetIRCLinkURL(ctx context.Context, id int) (string, error) {
	var link IRCLink
	// Select only URL to optimize?
	err := s.db.WithContext(ctx).Select("url").First(&link, id).Error
	return link.URL, err
}

func (s *GormStore) GetIRCLinksByURL(ctx context.Context, url string) ([]IRCLink, error) {
	var links []IRCLink
	err := s.db.WithContext(ctx).
		Where("url = ?", url).
		Order("timestamp DESC").
		Find(&links).Error
	return links, err
}

func (s *GormStore) IncrementClicks(ctx context.Context, id int) error {
	return s.db.WithContext(ctx).Model(&IRCLink{}).Where("ircLinkID = ?", id).UpdateColumn("clicks", gorm.Expr("clicks + ?", 1)).Error
}

func (s *GormStore) InsertIRCLink(ctx context.Context, user, title, url, contentType string) (int, error) {
	link := IRCLink{
		User:        user,
		Title:       title,
		URL:         url,
		ContentType: contentType,
		Timestamp:   time.Now(),
		Clicks:      0,
	}
	err := s.db.WithContext(ctx).Create(&link).Error
	return link.ID, err
}

func (s *GormStore) DeleteIRCLink(ctx context.Context, id int) error {
	result := s.db.WithContext(ctx).Delete(&IRCLink{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("link not found")
	}
	return nil
}

func (s *GormStore) InsertQuote(ctx context.Context, quoteText, author, poster string) (int, error) {
	quote := Quote{
		Quote:     quoteText,
		Author:    author,
		Poster:    poster,
		Timestamp: time.Now(),
	}
	err := s.db.WithContext(ctx).Create(&quote).Error
	return quote.ID, err
}

func (s *GormStore) GetRandomQuote(ctx context.Context) (*Quote, error) {
	var quote Quote
	// DB agnostic random order
	// MySQL: RAND(), SQLite: RANDOM(), Postgres: RANDOM()
	orderBy := "RAND()"
	if s.db.Dialector.Name() == "sqlite" {
		orderBy = "RANDOM()"
	} else if s.db.Dialector.Name() == "postgres" {
		orderBy = "RANDOM()"
	}

	err := s.db.WithContext(ctx).Order(clause.Expr{SQL: orderBy}).First(&quote).Error
	return &quote, err
}

func (s *GormStore) GetQuoteByID(ctx context.Context, id int) (*Quote, error) {
	var quote Quote
	err := s.db.WithContext(ctx).First(&quote, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &quote, nil
}

func (s *GormStore) DeleteQuote(ctx context.Context, id int) error {
	result := s.db.WithContext(ctx).Delete(&Quote{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("quote not found")
	}
	return nil
}

func (s *GormStore) GetUserStats(ctx context.Context, sortBy string, limit int, offset int) ([]UserStat, error) {
	var stats []UserStat

	// Complex aggregation, implementation depends on SQL dialect but standard SQL should work
	query := `
		SELECT
			u.user,
			COALESCE(l.count, 0) as link_count,
			COALESCE(q.count, 0) as quote_count
		FROM
			(SELECT DISTINCT user FROM ircLink UNION SELECT DISTINCT author as user FROM quote) u
		LEFT JOIN
			(SELECT user, COUNT(*) as count FROM ircLink GROUP BY user) l ON u.user = l.user
		LEFT JOIN
			(SELECT author, COUNT(*) as count FROM quote GROUP BY author) q ON u.user = q.author
	`

	orderBy := "link_count DESC"
	switch sortBy {
	case "user":
		orderBy = "u.user ASC"
	case "quotes":
		orderBy = "quote_count DESC"
	case "links":
		orderBy = "link_count DESC"
	}

	query += " ORDER BY " + orderBy

	// GORM raw query with limit offset
	err := s.db.WithContext(ctx).Raw(query).Scan(&stats).Error

	// If the raw query doesn't support Limit/Offset directly in GORM chaining for Raw, we need to append it.
	// But let's limit slice manually or append SQL?
	// Appending SQL is safer for pagination on DB side.
	// Re-implementing query construction properly:

	// Since we are using Raw, we must include LIMIT/OFFSET in the SQL string or use a subquery approach with GORM.
	// Sticking to Raw string manipulation for this complex query.

	// To perform limit/offset on the result of the UNION/JOIN, it's best to wrap it?
	// Or just append.

	// NOTE: GORM's `Scan` will map columns to struct fields nicely.

	// For pagination, we can slice the result if dataset is small, but DB pagination is better.
	// Let's use `Limit` and `Offset` methods on the *result*? No, `Raw` executes.

	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	err = s.db.WithContext(ctx).Raw(query).Scan(&stats).Error

	return stats, err
}

func (s *GormStore) GetLinksByUser(ctx context.Context, user string, limit int, offset int) ([]IRCLink, error) {
	var links []IRCLink
	err := s.db.WithContext(ctx).
		Where("user = ?", user).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&links).Error
	return links, err
}

// Helper struct for Timeline UNION scan
type timelineResult struct {
	Type      string
	ID        int
	Timestamp time.Time
	Title     string
	URL       string
	Content   string
	Author    string // User for links, Author for quotes
	MD5Sum    string
}

func (s *GormStore) GetUserTimeline(ctx context.Context, user string, filterType string, limit int, offset int) ([]TimelineItem, error) {
	var results []TimelineItem

	linkSelect := "SELECT 'link' as type, ircLinkID as id, timestamp, title, url, '' as content, user as author, '' as md5sum, content_type FROM ircLink WHERE user = ?"
	quoteSelect := "SELECT 'quote' as type, quoteID as id, timestamp, '' as title, '' as url, quote as content, author as author, '' as md5sum, '' as content_type FROM quote WHERE author = ?"

	var query string
	var args []interface{}

	if filterType == "links" {
		query = linkSelect
		args = append(args, user)
	} else if filterType == "quotes" {
		query = quoteSelect
		args = append(args, user)
	} else {
		query = linkSelect + " UNION ALL " + quoteSelect
		args = append(args, user, user)
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&results).Error
	return results, err
}

func (s *GormStore) GetGlobalTimeline(ctx context.Context, limit int, offset int) ([]TimelineItem, error) {
	var results []TimelineItem

	query := `
		SELECT
			'link' as type, ircLinkID as id, timestamp, title, url, '' as content, user as author, '' as md5sum
		FROM ircLink
		UNION ALL
		SELECT
			'quote' as type, quoteID as id, timestamp, '' as title, '' as url, quote as content, author as author, '' as md5sum
		FROM quote
		UNION ALL
		SELECT
			'image' as type, imageID as id, timestamp, title, url, '' as content, '' as author, md5sum
		FROM image
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`

	err := s.db.WithContext(ctx).Raw(query, limit, offset).Scan(&results).Error
	return results, err
}

func (s *GormStore) GetLinkPreview(ctx context.Context, url string) (*LinkPreview, error) {
	var preview LinkPreview
	err := s.db.WithContext(ctx).Where("url = ?", url).First(&preview).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &preview, nil
}

func (s *GormStore) InsertLinkPreview(ctx context.Context, url string, data []byte) error {
	preview := LinkPreview{
		URL:  url,
		Data: data,
	}
	// Use Save to handle upserts (update if exists) or explicit Replace
	// Clause properties for Upsert
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "url"}},
		DoUpdates: clause.AssignmentColumns([]string{"data", "updated_at"}),
	}).Create(&preview).Error
}

func (s *GormStore) DeleteLinkPreview(ctx context.Context, url string) error {
	return s.db.WithContext(ctx).Delete(&LinkPreview{}, "url = ?", url).Error
}

func (s *GormStore) DeleteAllLinkPreviews(ctx context.Context) (int, error) {
	result := s.db.WithContext(ctx).Where("1 = 1").Delete(&LinkPreview{})
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

func (s *GormStore) CreateTag(ctx context.Context, tag Tag) (*Tag, error) {
	err := s.db.WithContext(ctx).Create(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (s *GormStore) GetTagsByResource(ctx context.Context, resourceType string, resourceID int) ([]Tag, error) {
	var tags []Tag
	err := s.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at ASC").
		Find(&tags).Error
	return tags, err
}

func (s *GormStore) GetTagByID(ctx context.Context, id int) (*Tag, error) {
	var tag Tag
	err := s.db.WithContext(ctx).First(&tag, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tag, nil
}

func (s *GormStore) DeleteTag(ctx context.Context, id int) error {
	result := s.db.WithContext(ctx).Delete(&Tag{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("tag not found")
	}
	return nil
}

func (s *GormStore) DeleteTagsByResource(ctx context.Context, resourceType string, resourceID int) error {
	return s.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Delete(&Tag{}).Error
}

func (s *GormStore) GetLinksByPopularity(ctx context.Context, limit int, offset int) ([]IRCLink, error) {
	var links []IRCLink
	err := s.db.WithContext(ctx).
		Order("clicks DESC, timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&links).Error
	return links, err
}

package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	// Recommended settings
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	return &MySQLStore{db: db}, nil
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}

func (s *MySQLStore) GetRecentIRCLinks(ctx context.Context, startDays int, endDays int) ([]IRCLink, error) {
	query := `
		SELECT ircLinkID, timestamp, user, title, url, clicks, content_type
		FROM ircLink
		WHERE timestamp >= DATE_SUB(NOW(), INTERVAL ? DAY)
		AND timestamp <= DATE_SUB(NOW(), INTERVAL ? DAY)
		ORDER BY timestamp DESC
	`
	// Note: Perl logic was: start_days <= timestamp AND end_days >= timestamp
	// But start_days is the LARGER number (further back in time).
	// So timestamp >= (NOW - start) AND timestamp <= (NOW - end)

	rows, err := s.db.QueryContext(ctx, query, startDays, endDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []IRCLink
	for rows.Next() {
		var l IRCLink
		var contentType sql.NullString // Handle nullable
		if err := rows.Scan(&l.ID, &l.Timestamp, &l.User, &l.Title, &l.URL, &l.Clicks, &contentType); err != nil {
			return nil, err
		}
		if contentType.Valid {
			l.ContentType = contentType.String
		}
		links = append(links, l)
	}
	return links, nil
}

func (s *MySQLStore) GetRecentImages(ctx context.Context, startDays int, endDays int) ([]Image, error) {
	query := `
		SELECT imageID, timestamp, title, link, url, md5sum
		FROM image
		WHERE timestamp >= DATE_SUB(NOW(), INTERVAL ? DAY)
		AND timestamp <= DATE_SUB(NOW(), INTERVAL ? DAY)
		ORDER BY timestamp DESC
	`
	rows, err := s.db.QueryContext(ctx, query, startDays, endDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []Image
	for rows.Next() {
		var i Image
		if err := rows.Scan(&i.ID, &i.Timestamp, &i.Title, &i.Link, &i.URL, &i.MD5Sum); err != nil {
			return nil, err
		}
		images = append(images, i)
	}
	return images, nil
}

func (s *MySQLStore) GetRecentQuotes(ctx context.Context, startDays int, endDays int) ([]Quote, error) {
	query := `
		SELECT quoteID, timestamp, quote, author
		FROM quote
		WHERE timestamp >= DATE_SUB(NOW(), INTERVAL ? DAY)
		AND timestamp <= DATE_SUB(NOW(), INTERVAL ? DAY)
		ORDER BY timestamp DESC
	`
	rows, err := s.db.QueryContext(ctx, query, startDays, endDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotes []Quote
	for rows.Next() {
		var q Quote
		if err := rows.Scan(&q.ID, &q.Timestamp, &q.Quote, &q.Author); err != nil {
			return nil, err
		}
		quotes = append(quotes, q)
	}
	return quotes, nil
}

func (s *MySQLStore) SearchIRCLinks(ctx context.Context, searchTerm string) ([]IRCLink, error) {
	// MySQL FULLTEXT search
	query := `
		SELECT ircLinkID, timestamp, user, title, url, clicks, content_type
		FROM ircLink
		WHERE MATCH(title, url) AGAINST(? IN BOOLEAN MODE)
		ORDER BY clicks DESC
		LIMIT 50
	`
	rows, err := s.db.QueryContext(ctx, query, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []IRCLink
	for rows.Next() {
		var l IRCLink
		var contentType sql.NullString
		if err := rows.Scan(&l.ID, &l.Timestamp, &l.User, &l.Title, &l.URL, &l.Clicks, &contentType); err != nil {
			return nil, err
		}
		if contentType.Valid {
			l.ContentType = contentType.String
		}
		links = append(links, l)
	}
	return links, nil
}

func (s *MySQLStore) GetTopIRCLinks(ctx context.Context, startDays int, endDays int, limit int) ([]IRCLink, error) {
	query := `
		SELECT ircLinkID, timestamp, user, title, url, clicks, content_type
		FROM ircLink
		WHERE timestamp >= DATE_SUB(NOW(), INTERVAL ? DAY)
		AND timestamp <= DATE_SUB(NOW(), INTERVAL ? DAY)
		AND clicks > 1
		ORDER BY clicks DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, startDays, endDays, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []IRCLink
	for rows.Next() {
		var l IRCLink
		var contentType sql.NullString
		if err := rows.Scan(&l.ID, &l.Timestamp, &l.User, &l.Title, &l.URL, &l.Clicks, &contentType); err != nil {
			return nil, err
		}
		if contentType.Valid {
			l.ContentType = contentType.String
		}
		links = append(links, l)
	}
	return links, nil
}

func (s *MySQLStore) GetIRCLinkByID(ctx context.Context, id int) (*IRCLink, error) {
	query := `
		SELECT ircLinkID, timestamp, user, title, url, clicks, content_type
		FROM ircLink
		WHERE ircLinkID = ?
	`
	var l IRCLink
	var contentType sql.NullString
	err := s.db.QueryRowContext(ctx, query, id).Scan(&l.ID, &l.Timestamp, &l.User, &l.Title, &l.URL, &l.Clicks, &contentType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Or error?
		}
		return nil, err
	}
	if contentType.Valid {
		l.ContentType = contentType.String
	}
	return &l, nil
}

func (s *MySQLStore) GetIRCLinkURL(ctx context.Context, id int) (string, error) {
	query := `SELECT url FROM ircLink WHERE ircLinkID = ?`
	var url string
	err := s.db.QueryRowContext(ctx, query, id).Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (s *MySQLStore) IncrementClicks(ctx context.Context, id int) error {
	query := `UPDATE ircLink SET timestamp = timestamp, clicks = clicks + 1 WHERE ircLinkID = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *MySQLStore) InsertIRCLink(ctx context.Context, user, title, url, contentType string) (int, error) {
	query := `INSERT INTO ircLink (user, title, url, content_type) VALUES (?, ?, ?, ?)`
	res, err := s.db.ExecContext(ctx, query, user, title, url, contentType)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *MySQLStore) InsertQuote(ctx context.Context, quote, author string) error {
	query := `INSERT INTO quote (quote, author) VALUES (?, ?)`
	_, err := s.db.ExecContext(ctx, query, quote, author)
	return err
}

func (s *MySQLStore) Bootstrap(ctx context.Context) error {
	schema, err := SchemaFS.ReadFile("schema.mysql")
	if err != nil {
		return err
	}
	// Simple split by ; might fail on complex SQL, but for this schema it's fine.
	// Actually, the schema has multi-line statements.
	// A robust solution executes the whole script if the driver supports it, or splits carefully.
	// MySQL driver often supports multiple statements if enabled, but better to execute one by one if split properly.
	// For this specific schema, splitting by `;` works because there are no semicolons inside strings/triggers.
	// HOWEVER, creating a new method to execute script is cleaner.

	// Actually, just executing the whole thing might work if multiStatements=true in DSN, but let's assume not.
	// We'll follow a simple split approach for now, or just execute the known CREATE statements.
	// Since we want to use the embedded file, we should parse it.

	// Simpler: Just execute the file content?
	// Drivers behave differently.
	// Let's rely on the file content being simple enough.

	queries := splitSQL(string(schema))
	for _, q := range queries {
		if q == "" {
			continue
		}
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", q, err)
		}
	}
	return nil
}

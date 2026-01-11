package data

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) GetRecentIRCLinks(ctx context.Context, startDays int, endDays int) ([]IRCLink, error) {
	// timestamp >= datetime('now', '-' || ? || ' days')
	query := `
		SELECT ircLinkID, timestamp, user, title, url, clicks, content_type
		FROM ircLink
		WHERE timestamp >= datetime('now', '-' || ? || ' days')
		AND timestamp <= datetime('now', '-' || ? || ' days')
		ORDER BY timestamp DESC
	`
	slog.Debug("GetRecentIRCLinks", "query", query, "startDays", startDays, "endDays", endDays)
	rows, err := s.db.QueryContext(ctx, query, startDays, endDays)
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

func (s *SQLiteStore) GetRecentImages(ctx context.Context, startDays int, endDays int) ([]Image, error) {
	query := `
		SELECT imageID, timestamp, title, link, url, md5sum
		FROM image
		WHERE timestamp >= datetime('now', '-' || ? || ' days')
		AND timestamp <= datetime('now', '-' || ? || ' days')
		ORDER BY timestamp DESC
	`
	slog.Debug("GetRecentImages", "query", query, "startDays", startDays, "endDays", endDays)
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

func (s *SQLiteStore) GetRecentQuotes(ctx context.Context, startDays int, endDays int) ([]Quote, error) {
	query := `
		SELECT quoteID, timestamp, quote, author
		FROM quote
		WHERE timestamp >= datetime('now', '-' || ? || ' days')
		AND timestamp <= datetime('now', '-' || ? || ' days')
		ORDER BY timestamp DESC
	`
	slog.Debug("GetRecentQuotes", "query", query, "startDays", startDays, "endDays", endDays)
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

func (s *SQLiteStore) SearchIRCLinks(ctx context.Context, searchTerm string) ([]IRCLink, error) {
	// SQLite LIKE-based search (Perl compat)
	query := `
		SELECT ircLinkID, timestamp, user, title, url, clicks, content_type
		FROM ircLink
		WHERE title LIKE ? OR url LIKE ?
		ORDER BY clicks DESC
		LIMIT 50
	`
	likeTerm := fmt.Sprintf("%%%s%%", searchTerm)
	slog.Debug("SearchIRCLinks", "query", query, "searchTerm", searchTerm)
	rows, err := s.db.QueryContext(ctx, query, likeTerm, likeTerm)
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

func (s *SQLiteStore) GetTopIRCLinks(ctx context.Context, startDays int, endDays int, limit int) ([]IRCLink, error) {
	query := `
		SELECT ircLinkID, timestamp, user, title, url, clicks, content_type
		FROM ircLink
		WHERE timestamp >= datetime('now', '-' || ? || ' days')
		AND timestamp <= datetime('now', '-' || ? || ' days')
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

func (s *SQLiteStore) GetIRCLinkByID(ctx context.Context, id int) (*IRCLink, error) {
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
			return nil, nil
		}
		return nil, err
	}
	if contentType.Valid {
		l.ContentType = contentType.String
	}
	return &l, nil
}

func (s *SQLiteStore) GetIRCLinkURL(ctx context.Context, id int) (string, error) {
	query := `SELECT url FROM ircLink WHERE ircLinkID = ?`
	var url string
	err := s.db.QueryRowContext(ctx, query, id).Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (s *SQLiteStore) IncrementClicks(ctx context.Context, id int) error {
	// timestamp hack to match MySQL behavior if needed, but SQLite defaults current_timestamp on update usually triggers only if trigger exists?
	// The schema said default current timestamp.
	// Updating timestamp explicitly:
	query := `UPDATE ircLink SET timestamp = datetime('now'), clicks = clicks + 1 WHERE ircLinkID = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *SQLiteStore) InsertIRCLink(ctx context.Context, user, title, url, contentType string) (int, error) {
	query := `INSERT INTO ircLink (user, title, url, content_type) VALUES (?, ?, ?, ?)`
	res, err := s.db.ExecContext(ctx, query, user, title, url, contentType)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *SQLiteStore) InsertQuote(ctx context.Context, quote, author string) error {
	query := `INSERT INTO quote (quote, author) VALUES (?, ?)`
	_, err := s.db.ExecContext(ctx, query, quote, author)
	return err
}

func (s *SQLiteStore) Bootstrap(ctx context.Context) error {
	schema, err := SchemaFS.ReadFile("schema.sqlite")
	if err != nil {
		return err
	}

	// modernc.org/sqlite usually handles multiple statements in one Exec.
	// Let's try executing the whole block.
	if _, err := s.db.ExecContext(ctx, string(schema)); err != nil {
		return err
	}
	return nil
}

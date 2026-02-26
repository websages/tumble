package data

import (
	"context"
	"net/url"
	"sort"
	"strings"
)

// stopWords is a set of common English words filtered out during title
// tokenisation so that FTS and LIKE queries focus on meaningful terms.
var stopWords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true,
	"at": true, "be": true, "but": true, "by": true, "for": true,
	"from": true, "has": true, "have": true, "he": true, "her": true,
	"his": true, "how": true, "i": true, "if": true, "in": true,
	"into": true, "is": true, "it": true, "its": true, "my": true,
	"no": true, "not": true, "of": true, "on": true, "or": true,
	"our": true, "she": true, "so": true, "that": true, "the": true,
	"their": true, "them": true, "then": true, "there": true,
	"these": true, "they": true, "this": true, "to": true, "too": true,
	"us": true, "was": true, "we": true, "were": true, "what": true,
	"when": true, "where": true, "which": true, "who": true,
	"will": true, "with": true, "would": true, "you": true,
	"your": true,
}

// candidateLimit is the maximum number of raw candidates collected
// before deduplication.  The caller's limit is applied afterwards.
const candidateLimit = 50

// buildFTSOrQuery builds an FTS5 query string where each word is
// combined with OR so that a match on any single word returns results.
func buildFTSOrQuery(words []string) string {
	if len(words) == 0 {
		return ""
	}
	quoted := make([]string, len(words))
	for i, w := range words {
		w = strings.ReplaceAll(w, `"`, `""`)
		quoted[i] = `"` + w + `"`
	}
	return strings.Join(quoted, " OR ")
}

// ExtractDomain returns the hostname of rawURL with a leading "www."
// stripped.  It returns an empty string when the URL cannot be parsed.
func ExtractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Hostname()
	host = strings.TrimPrefix(host, "www.")
	return host
}

// ExtractTitleWords tokenises a title string and returns meaningful
// words (no stop words, minimum 2 characters).
func ExtractTitleWords(title string) []string {
	raw := strings.Fields(title)
	var words []string
	for _, w := range raw {
		w = strings.ToLower(w)
		// Strip common trailing punctuation.
		w = strings.TrimRight(w, ".,;:!?\"'")
		if len(w) < 2 {
			continue
		}
		if stopWords[w] {
			continue
		}
		words = append(words, w)
	}
	return words
}

// FindRelatedLinks returns candidate links related to the given title
// and URL.  The implementation dispatches to an FTS5-based query for
// SQLite or a LIKE-based fallback for MySQL.
func (s *GormStore) FindRelatedLinks(ctx context.Context, title string, rawURL string, excludeID int, limit int) ([]RelatedLink, error) {
	if s.db.Dialector.Name() == "sqlite" {
		return s.findRelatedSQLite(ctx, title, rawURL, excludeID, limit)
	}
	return s.findRelatedMySQL(ctx, title, rawURL, excludeID, limit)
}

// findRelatedSQLite uses FTS5 MATCH on the ircLink_fts virtual table
// combined with a domain LIKE clause.
func (s *GormStore) findRelatedSQLite(ctx context.Context, title string, rawURL string, excludeID int, limit int) ([]RelatedLink, error) {
	var candidates []RelatedLink
	seen := make(map[int]bool)

	// --- FTS title match ---
	words := ExtractTitleWords(title)
	if len(words) > 0 {
		ftsQuery := buildFTSOrQuery(words)
		if ftsQuery != "" {
			var ftsResults []RelatedLink
			q := s.db.WithContext(ctx).
				Table("ircLink").
				Select("ircLinkID as id, title, url, user, clicks, timestamp").
				Where("ircLinkID IN (SELECT rowid FROM ircLink_fts WHERE ircLink_fts MATCH ?)", ftsQuery)
			if excludeID > 0 {
				q = q.Where("ircLinkID != ?", excludeID)
			}
			if err := q.Order("clicks DESC").Limit(candidateLimit).Find(&ftsResults).Error; err != nil {
				return nil, err
			}
			for _, r := range ftsResults {
				if !seen[r.ID] {
					seen[r.ID] = true
					candidates = append(candidates, r)
				}
			}
		}
	}

	// --- Domain match ---
	domain := ExtractDomain(rawURL)
	if domain != "" {
		domainPattern := "%://" + escapeLike(domain) + "/%"
		domainWWWPattern := "%://www." + escapeLike(domain) + "/%"
		var domainResults []RelatedLink
		q := s.db.WithContext(ctx).
			Table("ircLink").
			Select("ircLinkID as id, title, url, user, clicks, timestamp").
			Where("(url LIKE ? ESCAPE '\\' OR url LIKE ? ESCAPE '\\')", domainPattern, domainWWWPattern)
		if excludeID > 0 {
			q = q.Where("ircLinkID != ?", excludeID)
		}
		if err := q.Order("clicks DESC").Limit(candidateLimit).Find(&domainResults).Error; err != nil {
			return nil, err
		}
		for _, r := range domainResults {
			if !seen[r.ID] {
				seen[r.ID] = true
				candidates = append(candidates, r)
			}
		}
	}

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
}

// findRelatedMySQL uses LIKE queries on the top 5 longest title words
// combined with a domain LIKE clause.
func (s *GormStore) findRelatedMySQL(ctx context.Context, title string, rawURL string, excludeID int, limit int) ([]RelatedLink, error) {
	var candidates []RelatedLink
	seen := make(map[int]bool)

	// --- Title word LIKE match ---
	words := ExtractTitleWords(title)
	if len(words) > 0 {
		// Sort words by length descending and keep top 5.
		sort.Slice(words, func(i, j int) bool {
			return len(words[i]) > len(words[j])
		})
		if len(words) > 5 {
			words = words[:5]
		}

		// Build OR conditions for each word.
		var conditions []string
		var args []interface{}
		for _, w := range words {
			pattern := "%" + escapeLike(w) + "%"
			conditions = append(conditions, "title LIKE ? ESCAPE '\\'")
			args = append(args, pattern)
		}
		whereClause := "(" + strings.Join(conditions, " OR ") + ")"
		if excludeID > 0 {
			whereClause += " AND ircLinkID != ?"
			args = append(args, excludeID)
		}

		var likeResults []RelatedLink
		if err := s.db.WithContext(ctx).
			Table("ircLink").
			Select("ircLinkID as id, title, url, user, clicks, timestamp").
			Where(whereClause, args...).
			Order("clicks DESC").
			Limit(candidateLimit).
			Find(&likeResults).Error; err != nil {
			return nil, err
		}
		for _, r := range likeResults {
			if !seen[r.ID] {
				seen[r.ID] = true
				candidates = append(candidates, r)
			}
		}
	}

	// --- Domain match ---
	domain := ExtractDomain(rawURL)
	if domain != "" {
		domainPattern := "%://" + escapeLike(domain) + "/%"
		domainWWWPattern := "%://www." + escapeLike(domain) + "/%"

		var domainArgs []interface{}
		whereClause := "(url LIKE ? ESCAPE '\\' OR url LIKE ? ESCAPE '\\')"
		domainArgs = append(domainArgs, domainPattern, domainWWWPattern)
		if excludeID > 0 {
			whereClause += " AND ircLinkID != ?"
			domainArgs = append(domainArgs, excludeID)
		}

		var domainResults []RelatedLink
		if err := s.db.WithContext(ctx).
			Table("ircLink").
			Select("ircLinkID as id, title, url, user, clicks, timestamp").
			Where(whereClause, domainArgs...).
			Order("clicks DESC").
			Limit(candidateLimit).
			Find(&domainResults).Error; err != nil {
			return nil, err
		}
		for _, r := range domainResults {
			if !seen[r.ID] {
				seen[r.ID] = true
				candidates = append(candidates, r)
			}
		}
	}

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
}

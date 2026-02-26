package handler

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"tumble/internal/data"
)

// ScoredLink pairs a candidate RelatedLink with a computed relevance score.
type ScoredLink struct {
	data.RelatedLink
	Score float64 `json:"score"`
}

// Scoring weights for the related-link ranking function.
const (
	weightDomain = 0.40
	weightTitle  = 0.30
	weightTag    = 0.15
	weightClick  = 0.10
	weightAge    = 0.05

	minScore        = 0.15
	maxRelatedLinks = 5
)

// titleWordSet tokenises a title into a set of lower-cased,
// de-punctuated words using data.ExtractTitleWords.
func titleWordSet(title string) map[string]bool {
	words := data.ExtractTitleWords(title)
	set := make(map[string]bool, len(words))
	for _, w := range words {
		set[strings.ToLower(w)] = true
	}
	return set
}

// scoreRelatedLinks scores and ranks candidate links relative to the
// source link described by title, rawURL, and tags.  Candidates
// scoring below minScore are discarded and the result is capped at
// maxRelatedLinks entries.
func scoreRelatedLinks(candidates []data.RelatedLink, title string, rawURL string, tags map[int][]string) []ScoredLink {
	if len(candidates) == 0 {
		return nil
	}

	sourceDomain := data.ExtractDomain(rawURL)
	sourceWords := titleWordSet(title)

	scored := make([]ScoredLink, 0, len(candidates))
	now := time.Now()

	for _, c := range candidates {
		var score float64

		// 1. Domain match (+0.40)
		if sourceDomain != "" && data.ExtractDomain(c.URL) == sourceDomain {
			score += weightDomain
		}

		// 2. Title overlap - Jaccard similarity (+0.30)
		candidateWords := titleWordSet(c.Title)
		if len(sourceWords) > 0 || len(candidateWords) > 0 {
			intersection := 0
			for w := range sourceWords {
				if candidateWords[w] {
					intersection++
				}
			}
			union := len(sourceWords) + len(candidateWords) - intersection
			if union > 0 {
				score += weightTitle * (float64(intersection) / float64(union))
			}
		}

		// 3. Tag overlap (+0.15)
		candidateTags := tags[c.ID]
		if len(candidateTags) > 0 {
			// Collect all unique tag strings across source and candidate.
			allTags := make(map[string]bool)
			shared := 0
			// We treat tags from ALL candidates' tag lists vs the
			// candidate's own list.  The source link's tags are in
			// the map under its own ID (key 0 by convention) but
			// the caller may not include them.  We compare each
			// candidate's tags against all other candidates' tags
			// for overlap.  However, the spec says "shared tags /
			// total tags" which implies comparing candidate tags
			// against the source.  We'll look for a source-link
			// entry (ID 0) in the map.
			sourceTags := tags[0]
			if len(sourceTags) > 0 {
				for _, t := range sourceTags {
					allTags[t] = true
				}
				for _, t := range candidateTags {
					allTags[t] = true
					for _, st := range sourceTags {
						if t == st {
							shared++
							break
						}
					}
				}
				total := len(allTags)
				if total > 0 {
					score += weightTag * (float64(shared) / float64(total))
				}
			}
		}

		// 4. Click bonus (+0.10): min(clicks/50, 1.0)
		clickRatio := math.Min(float64(c.Clicks)/50.0, 1.0)
		score += weightClick * clickRatio

		// 5. Age bonus (+0.05): min(ageDays/365, 1.0)
		ageDays := now.Sub(c.Timestamp).Hours() / 24.0
		if ageDays < 0 {
			ageDays = 0
		}
		ageRatio := math.Min(ageDays/365.0, 1.0)
		score += weightAge * ageRatio

		// Cap at 1.0
		if score > 1.0 {
			score = 1.0
		}

		if score >= minScore {
			scored = append(scored, ScoredLink{
				RelatedLink: c,
				Score:       score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Cap to max
	if len(scored) > maxRelatedLinks {
		scored = scored[:maxRelatedLinks]
	}

	return scored
}

// apiV1GetRelatedLinks handles GET /api/v1/links/{id}/related.
func (h *Handler) apiV1GetRelatedLinks(w http.ResponseWriter, r *http.Request, id int) {
	ctx := r.Context()

	// 1. Look up source link by ID
	link, err := h.Store.GetIRCLinkByID(ctx, id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch link")
		return
	}
	if link == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "Link not found")
		return
	}

	// Parse limit from query string (default 5, max 20)
	limit := parseIntParam(r, "limit", 5, 20)

	// 2. FindRelatedLinks: fetch limit*10 candidates for scoring
	candidates, err := h.Store.FindRelatedLinks(ctx, link.Title, link.URL, link.ID, limit*10)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to find related links")
		return
	}

	// 3. Build tag map for scoring
	tagMap := h.buildTagMap(ctx, candidates)

	// Also include source link's tags under ID 0
	sourceTags := h.getTagStrings(ctx, "link", link.ID)
	if len(sourceTags) > 0 {
		tagMap[0] = sourceTags
	}

	// 4. Score and rank
	scored := scoreRelatedLinks(candidates, link.Title, link.URL, tagMap)

	// 5. Cap to requested limit
	if len(scored) > limit {
		scored = scored[:limit]
	}

	// 6. Build response
	responseData := make([]APIRelatedLinkResponse, 0, len(scored))
	for _, s := range scored {
		responseData = append(responseData, APIRelatedLinkResponse{
			ID:        s.ID,
			Title:     s.Title,
			URL:       s.URL,
			User:      s.User,
			Clicks:    s.Clicks,
			CreatedAt: s.Timestamp,
			Score:     s.Score,
		})
	}

	writeJSON(w, http.StatusOK, APIRelatedResponse{Data: responseData})
}

// buildTagMap fetches tags for each candidate from the store and returns
// a map of link ID to tag strings.
func (h *Handler) buildTagMap(ctx context.Context, candidates []data.RelatedLink) map[int][]string {
	tagMap := make(map[int][]string, len(candidates))
	for _, c := range candidates {
		tags := h.getTagStrings(ctx, "link", c.ID)
		if len(tags) > 0 {
			tagMap[c.ID] = tags
		}
	}
	return tagMap
}

package handler

import (
	"net/url"
	"regexp"
	"strings"
)

// Google Maps URL patterns for extracting place/search/directions info
var (
	mapsPlaceRe  = regexp.MustCompile(`/maps/place/([^/@]+)`)
	mapsSearchRe = regexp.MustCompile(`/maps/search/([^/@]+)`)
	mapsDirRe    = regexp.MustCompile(`/maps/dir/([^/]+)/([^/@]+)`)
)

// isGoogleMapsShortURL returns true for shortened Google Maps URLs
// that need redirect resolution before title extraction.
func isGoogleMapsShortURL(rawURL string) bool {
	return strings.Contains(rawURL, "maps.app.goo.gl") ||
		strings.Contains(rawURL, "goo.gl/maps")
}

// GetGoogleMapsPreview handles preview logic for Google Maps URLs.
// Google Maps serves OG tags including a static map image, but the title
// is always "Google Maps" and the description is generic. This handler
// extracts the place name from the URL to produce a useful title.
func (h *Handler) GetGoogleMapsPreview(targetURL string) (map[string]string, error) {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	// Resolve short URLs to canonical form so we can extract the place name
	resolvedURL := targetURL
	if isGoogleMapsShortURL(targetURL) {
		if resolved, err := h.resolveRedirect(targetURL); err == nil && resolved != "" {
			resolvedURL = resolved
		}
	}

	meta, err := h.scrapeOpenGraph(resolvedURL, ua)
	if err != nil {
		return nil, err
	}

	// Extract a meaningful title from the URL structure
	title := extractMapsTitle(resolvedURL)
	if title != "" {
		meta["title"] = title
		// Delete the generic description when we have a real title —
		// it adds no value and the map image speaks for itself
		delete(meta, "description")
	}

	// Ensure provider_name is set
	meta["provider_name"] = "Google Maps"

	return meta, nil
}

// extractMapsTitle pulls a human-readable title from a Google Maps URL.
// Supports /place/, /search/, and /dir/ URL formats.
func extractMapsTitle(rawURL string) string {
	if m := mapsPlaceRe.FindStringSubmatch(rawURL); len(m) > 1 {
		return decodeMapsName(m[1])
	}

	if m := mapsSearchRe.FindStringSubmatch(rawURL); len(m) > 1 {
		return decodeMapsName(m[1])
	}

	if m := mapsDirRe.FindStringSubmatch(rawURL); len(m) > 2 {
		from := decodeMapsName(m[1])
		to := decodeMapsName(m[2])
		return from + " → " + to
	}

	return ""
}

// decodeMapsName converts URL-encoded place names to readable text.
// Google Maps uses "+" for spaces in place names.
func decodeMapsName(encoded string) string {
	// URL-decode first (handles %20, %2C, etc.)
	decoded, err := url.PathUnescape(encoded)
	if err != nil {
		decoded = encoded
	}
	// Google Maps also uses "+" for spaces in place names
	decoded = strings.ReplaceAll(decoded, "+", " ")
	return strings.TrimSpace(decoded)
}

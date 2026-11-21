package handler

import (
	"regexp"
	"strings"
)

// tweetMediaSuffix matches the /photo/N or /video/N that X appends when a
// tweet's media is opened. The oEmbed API returns 404 for those URLs.
var tweetMediaSuffix = regexp.MustCompile(`/(photo|video)/\d+/?$`)

// canonicalTweetURL strips media suffixes so oEmbed recognizes the tweet.
func canonicalTweetURL(u string) string {
	return tweetMediaSuffix.ReplaceAllString(u, "")
}

// GetTwitterPreview handles preview logic for Twitter/X.
// Currently, it leverages the default OEmbed behavior but suppresses the output
// to avoid duplicating the client-side widget if OEmbed is successful.
func (h *Handler) GetTwitterPreview(targetURL string) (map[string]string, error) {
	// Try OEmbed first
	meta, err := h.tryOEmbed(canonicalTweetURL(targetURL))
	if err == nil {
		// Existing logic:
		// "If valid (200), return empty success so frontend keeps the existing embed"
		// If we stick to that for Twitter/X ONLY:
		if strings.Contains(targetURL, "twitter.com") || strings.Contains(targetURL, "x.com") {
			return map[string]string{}, nil
		}
		return meta, nil
	}
	return nil, err
}

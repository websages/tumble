package handler

import (
	"fmt"
	"strings"
)

// GetYouTubePreview handles preview logic for YouTube.
// It checks generic OG metadata but includes logic to detect "soft 404s"
func (h *Handler) GetYouTubePreview(targetURL string) (map[string]string, error) {
	// Default UA
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	meta, err := h.scrapeOpenGraph(targetURL, ua)
	if err != nil {
		return nil, err
	}

	// Check for YouTube "soft 404"
	// Extracted from original scrapeOpenGraph
	title, hasTitle := meta["title"]

	if !hasTitle {
		return nil, fmt.Errorf("status 404")
	}

	// Detect YouTube "soft 404": unavailable videos return a generic title
	// like "YouTube" without the normal "Video Title - YouTube" format.
	if title == "YouTube" {
		return nil, fmt.Errorf("status 404")
	}

	// Strip the standard " - YouTube" suffix from the title
	title = strings.TrimSuffix(title, " - YouTube")
	meta["title"] = title

	meta["type"] = "video"

	return meta, nil
}

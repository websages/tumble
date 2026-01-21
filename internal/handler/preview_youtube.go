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

	// Make check more permissible using Contains
	if strings.Contains(title, " - YouTube") || title == "YouTube" {
		// Special handling for caller to know it's a 404
		return nil, fmt.Errorf("status 404")
	}

	meta["type"] = "video"

	return meta, nil
}

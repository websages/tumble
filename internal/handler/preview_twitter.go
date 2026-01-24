package handler

import "strings"

// GetTwitterPreview handles preview logic for Twitter/X.
// Currently, it leverages the default OEmbed behavior but suppresses the output
// to avoid duplicating the client-side widget if OEmbed is successful.
func (h *Handler) GetTwitterPreview(targetURL string) (map[string]string, error) {
	// Try OEmbed first
	meta, err := h.tryOEmbed(targetURL)
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

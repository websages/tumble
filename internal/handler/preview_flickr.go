package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
)

// GetFlickrPreview handles preview logic for Flickr URLs.
// Detects single photo URLs and returns image data for inline rendering.
// Albums, sets, and photostreams return normal preview card metadata.
func (h *Handler) GetFlickrPreview(targetURL string) (map[string]string, error) {
	// Detect URL type
	// Single photo: /photos/[username]/[photo_id]/
	// Album/Set: /photos/[username]/sets/[set_id] or /photos/[username]/albums/[album_id]
	// Photostream: /photos/[username]/ (no trailing photo ID)

	// Match single photo pattern
	singlePhotoRe := regexp.MustCompile(`flickr\.com/photos/[^/]+/(\d+)/?`)
	matches := singlePhotoRe.FindStringSubmatch(targetURL)

	// If this is NOT a single photo URL, return error to fall through to normal OEmbed
	if len(matches) < 2 {
		return nil, fmt.Errorf("not a single photo URL")
	}

	// Check if it's an album or set (these should use preview cards)
	if regexp.MustCompile(`/(?:sets|albums)/`).MatchString(targetURL) {
		return nil, fmt.Errorf("album or set URL")
	}

	// This is a single photo - fetch OEmbed data to get the image URL
	meta, err := h.tryOEmbed(targetURL)
	if err != nil {
		return nil, err
	}

	// For Flickr photo type, we want to extract the actual image URL
	// and signal that this should be rendered inline
	if meta["type"] == "photo" {
		// Flickr OEmbed returns the image URL in the "url" field for photo type
		// We need to fetch this separately if not already in meta
		if meta["image"] == "" && meta["url"] != "" {
			meta["image"] = meta["url"]
		}

		// If we still don't have an image, try fetching from OEmbed directly
		if meta["image"] == "" {
			imgURL, err := h.fetchFlickrImageURL(targetURL)
			if err == nil && imgURL != "" {
				meta["image"] = imgURL
			}
		}

		// Mark this as a photo type for inline rendering
		meta["type"] = "photo"
		meta["render_inline"] = "true"
	}

	return meta, nil
}

// fetchFlickrImageURL fetches the direct image URL from Flickr's OEmbed API
func (h *Handler) fetchFlickrImageURL(targetURL string) (string, error) {
	oembedURL := fmt.Sprintf("https://www.flickr.com/services/oembed?url=%s&format=json", url.QueryEscape(targetURL))

	req, err := http.NewRequest("GET", oembedURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout: h.Config.RequestTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("oembed status %d", resp.StatusCode)
	}

	var data OEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	// For photo type, the URL field contains the direct image URL
	if data.Type == "photo" && data.URL != "" {
		return data.URL, nil
	}

	return "", fmt.Errorf("no image URL found")
}

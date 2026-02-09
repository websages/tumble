package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// GetRedditPreview implements the hybrid Scrape + OEmbed approach
func (h *Handler) GetRedditPreview(targetURL string) (map[string]string, error) {
	// 1. Scrape with Slackbot UA for rich metadata (Title, Image, Description, Icon)
	slackbotUA := "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)"
	meta, err := h.scrapeOpenGraph(targetURL, slackbotUA)
	if err != nil {
		// If scrape fails, we might still try OEmbed, but usually if scrape fails, OEmbed might also fail/be lesser.
		// Let's rely on fallback. But for now, let's proceed to OEmbed only if scrape worked partially?
		// Or if scrape failed completely, just try OEmbed as last resort.
		// For simplicity, let's initialize map if nil
		if meta == nil {
			meta = make(map[string]string)
		}
	}

	// 2. Fetch OEmbed for the Embed HTML (Video Player)
	// We manually fetch from Reddit's OEmbed endpoint to bypass the provider check in global tryOEmbed.
	oembedEndpoint := "https://www.reddit.com/oembed"
	reqURL := fmt.Sprintf("%s?url=%s&format=json", oembedEndpoint, url.QueryEscape(targetURL))

	oembedMeta, err2 := h.manualFetchOEmbed(reqURL)
	if err2 == nil {
		// Merge OEmbed HTML into Scraped Meta
		if html, ok := oembedMeta["html"]; ok {
			meta["embed_html"] = html
		}
		// If Scrape failed to get type, use OEmbed type
		if _, ok := meta["type"]; !ok {
			if t, ok := oembedMeta["type"]; ok {
				meta["type"] = t
			}
		}
		// Force type to video/rich if we have html
		if meta["embed_html"] != "" {
			meta["type"] = "rich"
		}
	}

	// Ensure provider_name is always set for Reddit URLs
	if meta["provider_name"] == "" {
		meta["provider_name"] = "Reddit"
	}

	return meta, nil
}

// manualFetchOEmbed fetches OEmbed data from a fully constructed URL
func (h *Handler) manualFetchOEmbed(reqURL string) (map[string]string, error) {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	// Use generic browser UA
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout: h.Config.RequestTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("oembed status %d", resp.StatusCode)
	}

	var data OEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	// Map to generic map
	m := make(map[string]string)
	m["title"] = data.Title
	m["author_name"] = data.AuthorName
	m["provider_name"] = data.ProviderName
	m["type"] = data.Type
	m["html"] = data.HTML
	m["thumbnail_url"] = data.ThumbnailURL

	return m, nil
}

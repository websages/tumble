package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// OEmbed Providers Configuration
var oembedProviders = []struct {
	Pattern  string
	Endpoint string
	Format   string // "json" (default) or "xml" (not used here yet as we assume json)
}{
	{`^https?://(www\.)?(youtube\.com|youtu\.be)/.+`, "https://www.youtube.com/oembed", ""},
	{`^https?://(open\.)?spotify\.com/.+`, "https://open.spotify.com/oembed", ""},
	{`^https?://(www\.)?tiktok\.com/.+`, "https://www.tiktok.com/oembed", ""},
	// {`^https?://(www\.)?reddit\.com/.+`, "https://www.reddit.com/oembed", ""}, // Use scraping for better metadata
	{`^https?://(www\.)?flickr\.com/.+`, "https://www.flickr.com/services/oembed", ""},
	{`^https?://(www\.|mobile\.)?(twitter|x)\.com/.+`, "https://publish.twitter.com/oembed", ""},
	{`^https?://(www\.)?instagram\.com/.+`, "https://api.instagram.com/oembed", ""}, // Note: Often requires token
	{`^https?://(www\.)?dailymotion\.com/.+`, "https://www.dailymotion.com/services/oembed", ""},
	{`^https?://(www\.)?kickstarter\.com/projects/.+`, "https://www.kickstarter.com/services/oembed", ""},
	{`^https?://(www\.)?slideshare\.net/.+`, "https://www.slideshare.net/api/oembed/2", ""},
	{`^https?://speakerdeck\.com/.+`, "https://speakerdeck.com/oembed.json", ""},
	{`^https?://giphy\.com/gifs/.+`, "https://giphy.com/services/oembed", ""},
}

// OEmbedResponse represents standard OEmbed keys
type OEmbedResponse struct {
	Type         string `json:"type"`
	Version      string `json:"version"`
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	AuthorURL    string `json:"author_url"`
	ProviderName string `json:"provider_name"`
	ProviderURL  string `json:"provider_url"`
	CacheAge     int64  `json:"cache_age"`
	ThumbnailURL string `json:"thumbnail_url"`
	ThumbnailW   int    `json:"thumbnail_width"`
	ThumbnailH   int    `json:"thumbnail_height"`
	HTML         string `json:"html"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Description  string `json:"description"` // Non-standard but common
	URL          string `json:"url"`         // Required for type=photo
}

// OGPreviewHandler handles /ogpreview.cgi
func (h *Handler) OGPreviewHandler(w http.ResponseWriter, r *http.Request) {
	urlParam := r.URL.Query().Get("url")
	w.Header().Set("Content-Type", "application/json")

	if urlParam == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "No URL provided"})
		return
	}

	// 0. Special Hybrid Handlers
	if strings.Contains(urlParam, "reddit.com") {
		meta, err := h.GetRedditPreview(urlParam)
		if err == nil && len(meta) > 0 {
			json.NewEncoder(w).Encode(meta)
			return
		}
	} else if strings.Contains(urlParam, "twitter.com") || strings.Contains(urlParam, "x.com") {
		meta, err := h.GetTwitterPreview(urlParam)
		if err == nil {
			json.NewEncoder(w).Encode(meta)
			return
		}
	} else if strings.Contains(urlParam, "youtube.com") || strings.Contains(urlParam, "youtu.be") {
		meta, err := h.GetYouTubePreview(urlParam)
		if err == nil {
			json.NewEncoder(w).Encode(meta)
			return
		}
		// If we detected a soft 404, stop here and return 404 so UI can render "missing" badge
		if err != nil && err.Error() == "status 404" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  "Video Unavailable",
				"status": 404,
			})
			return
		}
	}

	// 1. Try OEmbed
	if meta, err := h.tryOEmbed(urlParam); err == nil {
		json.NewEncoder(w).Encode(meta)
		return
	}

	// 2. Fallback to generic scraping
	h.fetchOGScrape(w, urlParam)
}

func (h *Handler) tryOEmbed(targetURL string) (map[string]string, error) {
	var endpoint string
	for _, p := range oembedProviders {
		if matched, _ := regexp.MatchString(p.Pattern, targetURL); matched {
			endpoint = p.Endpoint
			break
		}
	}

	if endpoint == "" {
		return nil, fmt.Errorf("no provider")
	}

	// Build OEmbed URL
	reqURL := fmt.Sprintf("%s?url=%s&format=json", endpoint, url.QueryEscape(targetURL))

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	// Use strict browser UA to avoid bot detection (Kickstarter, TikTok, etc.)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{}
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

	// Validate "richness" of data
	// If we get just a title (TikTok sometimes), it's not enough for a preview card if we want rich media.
	// We need at least an Image OR HTML.
	// For type=photo, URL is the image.
	hasImage := data.ThumbnailURL != "" || (data.Type == "photo" && data.URL != "")
	if !hasImage && data.HTML == "" {
		// If we only have title, maybe fallback to OG is better?
		return nil, fmt.Errorf("incomplete oembed data")
	}

	// Map to our expected metadata format
	meta := make(map[string]string)
	meta["title"] = data.Title
	meta["provider_name"] = data.ProviderName
	meta["type"] = data.Type

	// Icon Logic for OEmbed
	if data.ProviderURL != "" {
		meta["icon"] = fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=32", data.ProviderURL)
	}

	// Image preference
	if data.ThumbnailURL != "" {
		meta["image"] = data.ThumbnailURL
	} else if data.Type == "photo" && data.URL != "" {
		meta["image"] = data.URL
	}

	// Description
	// If standard OEmbed doesn't have it, we might want to leave it empty or use Author?
	if data.Description != "" {
		meta["description"] = data.Description
	} else if data.AuthorName != "" {
		meta["description"] = fmt.Sprintf("By %s", data.AuthorName)
	}

	// Pass the embed HTML for video/rich types
	if (data.Type == "video" || data.Type == "rich") && data.HTML != "" {
		meta["embed_html"] = data.HTML
	}

	return meta, nil
}

func (h *Handler) fetchOGScrape(w http.ResponseWriter, urlParam string) {
	// Default UA
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	meta, err := h.scrapeOpenGraph(urlParam, ua)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		// If it's a 4xx/5xx error from the helper, we might want to pass that through
		// For now, generic error or basic mapping
		if strings.Contains(err.Error(), "status") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "HTTP Error",
				// Extract status code if possible, or default to 400
				"status": 400,
			})
		} else {
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch metadata"})
		}
		return
	}

	json.NewEncoder(w).Encode(meta)
}

func (h *Handler) scrapeOpenGraph(targetURL, userAgent string) (map[string]string, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	// Parse HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]string)

	// Helper to extract text from a node's children
	var extractText func(*html.Node) string
	extractText = func(n *html.Node) string {
		var sb strings.Builder
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sb.WriteString(extractText(c))
		}
		return sb.String()
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, content, name string
			for _, a := range n.Attr {
				if a.Key == "property" {
					property = a.Val
				}
				if a.Key == "content" {
					content = a.Val
				}
				if a.Key == "name" {
					name = a.Val
				}
			}

			if property == "og:title" {
				metadata["title"] = content
			} else if property == "og:description" {
				metadata["description"] = content
			} else if property == "og:image" {
				metadata["image"] = content
			} else if property == "og:site_name" {
				metadata["provider_name"] = content
			} else if name == "twitter:image" {
				metadata["twitter_image"] = content
			} else if name == "twitter:title" {
				metadata["twitter_title"] = content
			} else if name == "twitter:description" {
				metadata["twitter_description"] = content
			} else if name == "description" {
				if _, ok := metadata["description"]; !ok {
					metadata["description"] = content
				}
			}
		}
		// Look for link rel=icon
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, href string
			for _, a := range n.Attr {
				if a.Key == "rel" {
					rel = a.Val
				}
				if a.Key == "href" {
					href = a.Val
				}
			}
			if (rel == "icon" || rel == "shortcut icon") && href != "" {
				// Resolve absolute URL
				if strings.HasPrefix(href, "http") {
					metadata["icon"] = href
				} else if strings.HasPrefix(href, "//") {
					metadata["icon"] = "https:" + href
				} else if strings.HasPrefix(href, "/") {
					// Need base URL.
					// Simple hack: use the scheme/host from targetURL
					u, _ := url.Parse(targetURL)
					if u != nil {
						metadata["icon"] = fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, href)
					}
				}
			}
		}

		// Also look for title tag
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil {
				if _, ok := metadata["title"]; !ok {
					metadata["title"] = n.FirstChild.Data
				}
			}
		}

		// Look for first paragraph if no description yet
		if n.Type == html.ElementNode && n.Data == "p" {
			if _, hasDesc := metadata["description"]; !hasDesc {
				text := strings.TrimSpace(extractText(n))
				// Simple heuristic: length > 50
				if len(text) > 50 {
					if !strings.HasPrefix(text, "Coordinates:") {
						metadata["description"] = text
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	// Fallback icon if not found in Scrape
	if metadata["icon"] == "" {
		// Try using google favicon service on the scraped URL
		u, _ := url.Parse(targetURL)
		if u != nil {
			metadata["icon"] = fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s://%s&sz=32", u.Scheme, u.Host)
		}
	}

	return metadata, nil
}

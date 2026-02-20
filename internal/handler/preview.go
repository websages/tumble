package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"tumble/internal/data"

	"golang.org/x/net/html"
)

const (
	previewCacheTTL     = 10 * 24 * time.Hour // Normal previews: 10 days
	errorCacheTTLRecent = 24 * time.Hour      // Error for links < 10 days old: 24h
	errorCacheTTLOld    = 60 * 24 * time.Hour // Error for links > 10 days old: 60 days
	linkAgeCutoff       = 10 * 24 * time.Hour // Boundary between "recent" and "old"
)

// OEmbed Providers Configuration
var oembedProviders = []struct {
	Pattern  string
	Endpoint string
	Format   string // "json" (default) or "xml" (not used here yet as we assume json)
}{
	{`^https?://(www\.)?(youtube\.com|youtu\.be)/.+`, "https://www.youtube.com/oembed", ""},
	{`^https?://(open\.)?spotify\.com/.+`, "https://open.spotify.com/oembed", ""},
	{`^https?://(www\.)?soundcloud\.com/.+`, "https://soundcloud.com/oembed", ""},
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
	Type         string      `json:"type"`
	Version      interface{} `json:"version"`
	Title        string      `json:"title"`
	AuthorName   string      `json:"author_name"`
	AuthorURL    string      `json:"author_url"`
	ProviderName string      `json:"provider_name"`
	ProviderURL  string      `json:"provider_url"`

	// CacheAge     int64  `json:"cache_age"`
	ThumbnailURL string `json:"thumbnail_url"`
	ThumbnailW   int    `json:"thumbnail_width"`
	ThumbnailH   int    `json:"thumbnail_height"`
	HTML         string `json:"html"`
	// Width        int    `json:"width"`
	// Height       int    `json:"height"`
	Description string `json:"description"` // Non-standard but common
	URL         string `json:"url"`         // Required for type=photo
}

// isPreviewExpired checks whether a cached link preview has exceeded its TTL.
// Error previews use a tiered TTL based on link age: recent links get a short
// TTL (24h) so errors are retried quickly, while old links get a long TTL (60d).
func isPreviewExpired(cached *data.LinkPreview, linkTimestamp time.Time) bool {
	if !strings.Contains(string(cached.Data), `"error":`) {
		return time.Since(cached.UpdatedAt) > previewCacheTTL
	}
	ttl := errorCacheTTLOld
	if !linkTimestamp.IsZero() && time.Since(linkTimestamp) < linkAgeCutoff {
		ttl = errorCacheTTLRecent
	}
	return time.Since(cached.UpdatedAt) > ttl
}

// TryServeCachedOGPreview checks the cache and serves the response if found.
// Returns true if served from cache, false if cache miss (caller should proceed).
func (h *Handler) TryServeCachedOGPreview(w http.ResponseWriter, r *http.Request) bool {
	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		return false // Let the main handler deal with the error
	}

	if !h.Config.Caching.Enabled {
		return false
	}

	cached, err := h.Store.GetLinkPreview(r.Context(), urlParam)
	if err != nil || cached == nil {
		return false
	}

	// For error entries, look up the link's timestamp to determine TTL tier
	var linkTimestamp time.Time
	if strings.Contains(string(cached.Data), `"error":`) {
		if links, err := h.Store.GetIRCLinksByURL(r.Context(), urlParam, data.ClientFilter{}); err == nil && len(links) > 0 {
			linkTimestamp = links[len(links)-1].Timestamp // oldest link (results ordered DESC)
		}
	}

	if isPreviewExpired(cached, linkTimestamp) {
		h.Store.DeleteLinkPreview(r.Context(), urlParam)
		return false
	}

	var meta map[string]string
	if err := json.Unmarshal(cached.Data, &meta); err != nil {
		return false
	}

	if _, isError := meta["error"]; isError {
		archiveData := h.attachArchiveData(r.Context(), urlParam)
		for k, v := range archiveData {
			meta[k] = v
		}
	}

	// Cache hit - serve response
	w.Header().Set("Content-Type", "application/json")
	if !h.Config.Caching.Enabled {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	json.NewEncoder(w).Encode(meta)
	return true
}

// OGPreviewHandler handles /ogpreview.cgi
func (h *Handler) OGPreviewHandler(w http.ResponseWriter, r *http.Request) {
	urlParam := r.URL.Query().Get("url")
	w.Header().Set("Content-Type", "application/json")

	if urlParam == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "No URL provided"})
		return
	}

	// Note: Cache check is now done in TryServeCachedOGPreview before rate limiting.
	// If we reach here, it's a cache miss.

	// 0. Special Hybrid Handlers
	if strings.Contains(urlParam, "reddit.com") {
		meta, err := h.GetRedditPreview(urlParam)
		if err == nil && len(meta) > 0 {
			h.cacheAndRespond(w, r, urlParam, meta)
			return
		}
	} else if strings.Contains(urlParam, "twitter.com") || strings.Contains(urlParam, "x.com") {
		meta, err := h.GetTwitterPreview(urlParam)
		if err == nil {
			h.cacheAndRespond(w, r, urlParam, meta)
			return
		}
		// If OEmbed returned ANY error (404, 403, etc.), we trust it. Do not fall back to scrape.
		if strings.Contains(err.Error(), "oembed status") {
			var code int
			if n, _ := fmt.Sscanf(err.Error(), "oembed status %d", &code); n != 1 {
				code = 404
			}
			h.cacheErrorPreview(r, urlParam, "Tweet Unavailable", code)
			archiveData := h.attachArchiveData(r.Context(), urlParam)
			resp := map[string]interface{}{
				"error":  "Tweet Unavailable",
				"status": code,
			}
			for k, v := range archiveData {
				resp[k] = v
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
	} else if strings.Contains(urlParam, "flickr.com") {
		meta, err := h.GetFlickrPreview(urlParam)
		if err == nil {
			h.cacheAndRespond(w, r, urlParam, meta)
			return
		}
		// If not a single photo, fall through to normal OEmbed
	} else if strings.Contains(urlParam, "youtube.com") || strings.Contains(urlParam, "youtu.be") {
		meta, err := h.GetYouTubePreview(urlParam)
		if err == nil {
			h.cacheAndRespond(w, r, urlParam, meta)
			return
		}
		// If we detected a soft 404 from scraping (e.g. consent page with generic
		// "YouTube" title), verify with OEmbed before declaring dead. The OEmbed API
		// returns proper data regardless of consent screens or JS rendering.
		if err.Error() == "status 404" {
			if oeMeta, oeErr := h.tryOEmbed(urlParam); oeErr == nil {
				h.cacheAndRespond(w, r, urlParam, oeMeta)
				return
			}
			h.cacheErrorPreview(r, urlParam, "Video Unavailable", 404)
			archiveData := h.attachArchiveData(r.Context(), urlParam)
			resp := map[string]interface{}{
				"error":  "Video Unavailable",
				"status": 404,
			}
			for k, v := range archiveData {
				resp[k] = v
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
	}

	// 1. Try OEmbed
	if meta, err := h.tryOEmbed(urlParam); err == nil {
		h.cacheAndRespond(w, r, urlParam, meta)
		return
	}

	// 2. Fallback to generic scraping
	h.fetchOGScrape(w, r, urlParam)
}

func (h *Handler) cacheErrorPreview(r *http.Request, urlParam string, errorMsg string, status int) {
	// Always cache errors regardless of caching config, since error entries
	// are used to exclude dead links from search results.
	meta := map[string]string{
		"error":  errorMsg,
		"status": fmt.Sprintf("%d", status),
	}
	if data, err := json.Marshal(meta); err == nil {
		h.Store.InsertLinkPreview(r.Context(), urlParam, data)
	}
}

const archiveNotFoundRecheckAge = 30 * 24 * time.Hour

// attachArchiveData checks for an archive.org snapshot of a dead link URL.
// If a fresh lookup exists, it returns the archive data to include in the
// response. If no lookup exists (or it's stale), it performs a lazy check.
func (h *Handler) attachArchiveData(ctx context.Context, urlParam string) map[string]string {
	lookup, err := h.Store.GetArchiveLookup(ctx, urlParam)
	if err != nil {
		return nil
	}

	needsCheck := lookup == nil ||
		(lookup.Status == "error" && time.Since(lookup.CheckedAt) > 24*time.Hour) ||
		(lookup.Status == "not_found" && time.Since(lookup.CheckedAt) > archiveNotFoundRecheckAge)

	if needsCheck && h.ArchiveClient != nil {
		result, err := h.ArchiveClient.Check(ctx, urlParam)
		if err != nil {
			h.Store.UpsertArchiveLookup(ctx, &data.ArchiveLookup{
				URL:       urlParam,
				Status:    "error",
				CheckedAt: time.Now(),
			})
			return nil
		}

		newLookup := &data.ArchiveLookup{
			URL:       urlParam,
			CheckedAt: time.Now(),
		}
		if result.Found {
			newLookup.Status = "found"
			newLookup.ArchiveURL = &result.ArchiveURL
			if !result.SnapshotAt.IsZero() {
				newLookup.SnapshotAt = &result.SnapshotAt
			}
		} else {
			newLookup.Status = "not_found"
		}
		h.Store.UpsertArchiveLookup(ctx, newLookup)
		lookup = newLookup
	}

	if lookup != nil && lookup.Status == "found" && lookup.ArchiveURL != nil {
		result := map[string]string{
			"archive_url": *lookup.ArchiveURL,
		}
		if lookup.SnapshotAt != nil {
			result["archive_snapshot_at"] = lookup.SnapshotAt.Format(time.RFC3339)
		}
		return result
	}

	return nil
}

func (h *Handler) cacheAndRespond(w http.ResponseWriter, r *http.Request, urlParam string, meta map[string]string) {
	// Cache if enabled
	if h.Config.Caching.Enabled {
		if data, err := json.Marshal(meta); err == nil {
			h.Store.InsertLinkPreview(r.Context(), urlParam, data)
		}
	}
	if !h.Config.Caching.Enabled {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	} else {
		// Client-side Caching Header (24h)
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	json.NewEncoder(w).Encode(meta)
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
	// Skip description for Giphy - it's redundant with the title
	if data.Description != "" && data.ProviderName != "GIPHY" {
		meta["description"] = data.Description
	} else if data.AuthorName != "" && data.ProviderName != "GIPHY" {
		meta["description"] = fmt.Sprintf("By %s", data.AuthorName)
	}

	// Pass the embed HTML for video/rich types
	if (data.Type == "video" || data.Type == "rich") && data.HTML != "" {
		meta["embed_html"] = data.HTML
	}

	return meta, nil
}

func (h *Handler) fetchOGScrape(w http.ResponseWriter, r *http.Request, urlParam string) {
	// Try with browser UA first
	browserUA := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	// Facebook's crawler UA - widely whitelisted for OpenGraph fetching
	crawlerUA := "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)"

	meta, err := h.scrapeOpenGraph(urlParam, browserUA)
	if err != nil {
		// If we got a 403, retry with crawler UA (many sites whitelist known bots)
		if strings.Contains(err.Error(), "status 403") {
			meta, err = h.scrapeOpenGraph(urlParam, crawlerUA)
		}
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		// If it's a 4xx/5xx error from the helper, we might want to pass that through
		if strings.Contains(err.Error(), "status") {
			code := 400
			if n, _ := fmt.Sscanf(err.Error(), "status %d", &code); n != 1 {
				code = 400
			}
			h.cacheErrorPreview(r, urlParam, "HTTP Error", code)
			archiveData := h.attachArchiveData(r.Context(), urlParam)
			resp := map[string]interface{}{
				"error":  "HTTP Error",
				"status": code,
			}
			for k, v := range archiveData {
				resp[k] = v
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch metadata"})
		}
		return
	}

	h.cacheAndRespond(w, r, urlParam, meta)
}

func (h *Handler) scrapeOpenGraph(targetURL, userAgent string) (map[string]string, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{
		Timeout: h.Config.RequestTimeout,
	}
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
			} else if property == "og:video" || property == "og:video:url" {
				metadata["video"] = content
			} else if property == "og:video:secure_url" {
				metadata["video_secure_url"] = content
			} else if property == "og:video:width" {
				metadata["video_width"] = content
			} else if property == "og:video:height" {
				metadata["video_height"] = content
			} else if property == "og:type" {
				metadata["og_type"] = content
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

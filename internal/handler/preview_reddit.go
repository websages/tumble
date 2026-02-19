package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// GetRedditPreview implements the hybrid Scrape + OEmbed approach.
// For video posts, it builds a proper video embed using Reddit's media
// embed player instead of the OEmbed blockquote.
func (h *Handler) GetRedditPreview(targetURL string) (map[string]string, error) {
	// 0. Resolve share link redirects (/s/ URLs redirect to canonical post URL)
	resolvedURL := targetURL
	if strings.Contains(targetURL, "/s/") {
		if resolved, err := h.resolveRedirect(targetURL); err == nil && resolved != "" {
			resolvedURL = resolved
		}
	}

	// 1. Scrape with Slackbot UA for rich metadata (Title, Image, Description, Icon)
	slackbotUA := "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)"
	meta, err := h.scrapeOpenGraph(resolvedURL, slackbotUA)
	if err != nil {
		if meta == nil {
			meta = make(map[string]string)
		}
	}

	// 2. Check for video post via OG tags first, then Reddit JSON API
	postID := extractRedditPostID(resolvedURL)
	if postID != "" {
		// Enhance metadata with uncropped image from Reddit JSON API
		// This is critical for vertical videos where og:image is cropped to 16:9
		if imgURL, w, h, err := h.fetchRedditJSONDetails(postID); err == nil && imgURL != "" {
			meta["image"] = imgURL
			meta["embed_width"] = fmt.Sprintf("%d", w)
			meta["embed_height"] = fmt.Sprintf("%d", h)
		}

		isVideo := isRedditVideoPost(meta)
		var videoInfo *redditVideoInfo
		if !isVideo {
			videoInfo = h.getRedditVideoInfo(postID)
			isVideo = videoInfo != nil
		}
		if isVideo {
			embedURL := fmt.Sprintf("https://www.redditmedia.com/mediaembed/%s", postID)
			meta["embed_html"] = fmt.Sprintf(
				`<iframe src="%s" style="width:100%%;border:none;" allowfullscreen></iframe>`,
				embedURL,
			)
			meta["type"] = "video"
			// Note: fetchRedditJSONDetails already sets embed_width/height if available,
			// but we keep the fallback to videoInfo if needed (though JSON is preferred).
			if _, ok := meta["embed_width"]; !ok {
				if videoInfo != nil && videoInfo.Width > 0 && videoInfo.Height > 0 {
					meta["embed_width"] = fmt.Sprintf("%d", videoInfo.Width)
					meta["embed_height"] = fmt.Sprintf("%d", videoInfo.Height)
				}
			}
		}
	}

	// 3. If no video embed, fall back to OEmbed for blockquote embed
	if meta["embed_html"] == "" {
		oembedEndpoint := "https://www.reddit.com/oembed"
		reqURL := fmt.Sprintf("%s?url=%s&format=json", oembedEndpoint, url.QueryEscape(resolvedURL))

		oembedMeta, err2 := h.manualFetchOEmbed(reqURL)
		if err2 == nil {
			if html, ok := oembedMeta["html"]; ok {
				meta["embed_html"] = html
			}
			if _, ok := meta["type"]; !ok {
				if t, ok := oembedMeta["type"]; ok {
					meta["type"] = t
				}
			}
			if meta["embed_html"] != "" {
				meta["type"] = "rich"
			}
		}
	}

	// Clean up internal-only OG keys before responding
	delete(meta, "video")
	delete(meta, "video_secure_url")
	delete(meta, "video_width")
	delete(meta, "video_height")
	delete(meta, "og_type")

	// Ensure provider_name is always set for Reddit URLs
	if meta["provider_name"] == "" {
		meta["provider_name"] = "Reddit"
	}

	return meta, nil
}

// resolveRedirect follows HTTP redirects and returns the final URL.
func (h *Handler) resolveRedirect(targetURL string) (string, error) {
	req, err := http.NewRequest("HEAD", targetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)")

	client := &http.Client{
		Timeout: h.Config.RequestTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return resp.Request.URL.String(), nil
}

// isRedditVideoPost checks OG metadata for indicators that the post contains video.
func isRedditVideoPost(meta map[string]string) bool {
	if v := meta["video"]; v != "" {
		return true
	}
	if v := meta["video_secure_url"]; v != "" {
		return true
	}
	if t := meta["og_type"]; strings.Contains(t, "video") {
		return true
	}
	return false
}

type redditVideoInfo struct {
	Width  int // embed player's data-video-width (NOT the raw video dimensions)
	Height int // embed player's data-video-height
}

var (
	dataVideoWidthRe  = regexp.MustCompile(`data-video-width="(\d+)"`)
	dataVideoHeightRe = regexp.MustCompile(`data-video-height="(\d+)"`)
)

// getRedditVideoInfo fetches the redditmedia embed page to detect video posts
// and extract the embed player's actual display dimensions.
// Returns nil if the post has no video embed.
func (h *Handler) getRedditVideoInfo(postID string) *redditVideoInfo {
	embedURL := fmt.Sprintf("https://www.redditmedia.com/mediaembed/%s", postID)

	req, err := http.NewRequest("GET", embedURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout: h.Config.RequestTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	html := string(body)

	// Only treat as video if the embed page contains a video player
	if !strings.Contains(html, "video-player") {
		return nil
	}

	info := &redditVideoInfo{}
	if m := dataVideoWidthRe.FindStringSubmatch(html); len(m) >= 2 {
		info.Width, _ = strconv.Atoi(m[1])
	}
	if m := dataVideoHeightRe.FindStringSubmatch(html); len(m) >= 2 {
		info.Height, _ = strconv.Atoi(m[1])
	}
	return info
}

var redditPostIDRe = regexp.MustCompile(`/comments/([a-z0-9]+)`)

// extractRedditPostID extracts the post ID from a canonical Reddit URL.
// URL format: https://www.reddit.com/r/{subreddit}/comments/{post_id}/{slug}/
func extractRedditPostID(rawURL string) string {
	matches := redditPostIDRe.FindStringSubmatch(rawURL)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
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

// fetchRedditJSONDetails fetches the uncropped image and dimensions from the Reddit JSON API.
// This is critical because the standard og:image is often cropped to 16:9, breaking vertical video embeds.
func (h *Handler) fetchRedditJSONDetails(postID string) (imageURL string, width, height int, err error) {
	fmt.Printf("[Debug] fetchRedditJSONDetails called for %s\n", postID)
	jsonURL := fmt.Sprintf("https://www.reddit.com/comments/%s.json", postID)
	req, err := http.NewRequest("GET", jsonURL, nil)
	if err != nil {
		return "", 0, 0, err
	}
	// Use a standard browser UA to avoid bot detection/rate limiting
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout: h.Config.RequestTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, 0, fmt.Errorf("reddit json status: %d", resp.StatusCode)
	}

	// Define a minimal struct to parse the deep JSON structure
	var response []struct {
		Data struct {
			Children []struct {
				Data struct {
					Preview struct {
						Images []struct {
							Source struct {
								URL    string `json:"url"`
								Width  int    `json:"width"`
								Height int    `json:"height"`
							} `json:"source"`
						} `json:"images"`
					} `json:"preview"`
					Media struct {
						RedditVideo struct {
							Width       int    `json:"width"`
							Height      int    `json:"height"`
							DashURL     string `json:"dash_url"`
							HLSURL      string `json:"hls_url"`
							FallbackURL string `json:"fallback_url"`
						} `json:"reddit_video"`
					} `json:"media"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", 0, 0, err
	}

	if len(response) > 0 && len(response[0].Data.Children) > 0 {
		post := response[0].Data.Children[0].Data

		// Priority 1: Check deep media info (most accurate for videos)
		if post.Media.RedditVideo.Width > 0 && post.Media.RedditVideo.Height > 0 {
			// Use the fallback URL (mp4) or HLS if needed
			url := post.Media.RedditVideo.FallbackURL
			if url == "" {
				url = post.Media.RedditVideo.HLSURL
			}
			if url == "" {
				url = post.Media.RedditVideo.DashURL
			}
			if url == "" {
				// We need *some* string for the caller to accept the dimensions
				// Use the post ID as a placeholder if nothing else
				url = fmt.Sprintf("https://v.redd.it/%s", postID)
			}
			return url, post.Media.RedditVideo.Width, post.Media.RedditVideo.Height, nil
		}

		// Priority 2: Check preview images
		if len(post.Preview.Images) > 0 {
			source := post.Preview.Images[0].Source
			// Reddit JSON URLs often contain &amp; encoding
			url := strings.ReplaceAll(source.URL, "&amp;", "&")
			return url, source.Width, source.Height, nil
		}
	}

	return "", 0, 0, fmt.Errorf("no preview image found")
}

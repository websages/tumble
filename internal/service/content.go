package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"tumble/internal/config"
	"tumble/internal/data"
)

type ContentService struct {
	Config *config.Config
	Store  data.Store
}

// EmbedType identifies the type of embed for template rendering
type EmbedType string

const (
	EmbedTypeGeneric      EmbedType = "generic"
	EmbedTypeImage        EmbedType = "image"
	EmbedTypeTwitter      EmbedType = "twitter"
	EmbedTypeImgurGallery EmbedType = "imgur_gallery"
	EmbedTypeImgurSingle  EmbedType = "imgur_single"
	EmbedTypeFlickr       EmbedType = "flickr"
	EmbedTypeQuote        EmbedType = "quote"
)

// DisplayItem is a data-only struct for template rendering.
// Templates are responsible for generating HTML based on these fields.
type DisplayItem struct {
	ID            int       `json:"id"`
	Type          string    `json:"type"` // "ircLink", "image", "quote"
	Timestamp     time.Time `json:"timestamp"`
	FormattedDate string    `json:"formatted_date"`
	User          string    `json:"user"`
	Author        string    `json:"author"`
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	Clicks        int       `json:"clicks"`
	Description   string    `json:"description,omitempty"`
	ContentType   string    `json:"content_type"` // MIME type from HTTP response
	BaseURL       string    `json:"base_url"`

	// Embed rendering metadata (templates use these to decide how to render)
	EmbedType    EmbedType `json:"embed_type"`
	EmbedURL     string    `json:"embed_url,omitempty"`     // URL for embed (e.g., Twitter embed URL)
	ThumbnailURL string    `json:"thumbnail_url,omitempty"` // For gallery cards
	MediaURL     string    `json:"media_url,omitempty"`     // Direct media URL (image/video src)
	MediaType    string    `json:"media_type,omitempty"`    // "video", "image"
	PhotoPageURL string    `json:"photo_page_url,omitempty"`
	IsAnimated   bool      `json:"is_animated,omitempty"` // mp4, gifv, gif
	IsBroken     bool      `json:"is_broken,omitempty"`   // 404 detection
	IsNSFW       bool      `json:"is_nsfw,omitempty"`     // NSFW flag

	// Display text (truncated title for display)
	DisplayTitle string `json:"display_title,omitempty"`

	// Quote-specific fields
	Quote string `json:"quote,omitempty"`

	// Date components for grouping
	DateDay    string `json:"date_day"`     // e.g. "Mon"
	DateMonth  string `json:"date_month"`   // e.g. "Jan"
	DateRawDay string `json:"date_raw_day"` // e.g. "01"
	DateYear   string `json:"date_year"`    // e.g. "2006"
	FullDate   string `json:"full_date"`    // e.g. "20060102" for grouping

	// OG preview control
	SuppressOG bool `json:"suppress_og"`

	// Click signature for verified click tracking (computed at render time, not stored)
	ClickSig string `json:"click_sig,omitempty"`
}

func NewContentService(cfg *config.Config, store data.Store) *ContentService {
	return &ContentService{Config: cfg, Store: store}
}

// generateClickSignature creates an HMAC signature for click tracking.
// This ensures only clicks from legitimately rendered pages are counted.
func (s *ContentService) generateClickSignature(id int) string {
	secret := s.Config.ClickSigningKey
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d", id)))
	// Use first 16 hex chars (8 bytes) - sufficient for integrity, keeps URLs short
	return hex.EncodeToString(mac.Sum(nil))[:16]
}

func (s *ContentService) ProcessIRCLink(item data.IRCLink) DisplayItem {
	d := DisplayItem{
		ID:          item.ID,
		Type:        "ircLink",
		Timestamp:   item.Timestamp,
		User:        item.User,
		Title:       item.Title,
		URL:         item.URL,
		Clicks:      item.Clicks,
		ContentType: item.ContentType,
		BaseURL:     s.Config.BaseURL,
		EmbedType:   EmbedTypeGeneric, // Default
		ClickSig:    s.generateClickSignature(item.ID),
	}
	s.formatDate(&d)

	// Set display title (truncated if needed)
	d.DisplayTitle = item.Title
	if len(item.Title) > 40 && strings.HasPrefix(item.Title, "http://") {
		d.DisplayTitle = item.Title[:40] + "..."
	}

	// Check for NSFW content
	d.IsNSFW = strings.Contains(item.User, "nsfw") || strings.Contains(item.User, "otd")

	// IMPORTANT: Check for site-specific handlers BEFORE generic content type checks
	// This ensures Flickr, Imgur, etc. are handled correctly even when content-type is image/*

	// Twitter / X
	if strings.Contains(item.URL, "twitter.com") || strings.Contains(item.URL, "x.com") {
		re := regexp.MustCompile(`(?:twitter\.com|x\.com)\/.*\/status\/([0-9]+)`)
		matches := re.FindStringSubmatch(item.URL)
		if len(matches) > 1 {
			// Force twitter.com domain for embed compatibility
			d.EmbedType = EmbedTypeTwitter
			d.EmbedURL = strings.Replace(item.URL, "x.com", "twitter.com", 1)
			d.SuppressOG = false
			return d
		}
	}

	// Flickr - check before generic image handler
	if strings.Contains(item.URL, "flickr.com") || strings.Contains(item.URL, "staticflickr.com") {
		// Case 1: Static Flickr image URL (farm*.staticflickr.com)
		re := regexp.MustCompile(`\/([0-9]+)_[0-9a-z]+`)
		matches := re.FindStringSubmatch(item.URL)
		if len(matches) > 1 {
			photoID := matches[1]
			d.EmbedType = EmbedTypeFlickr
			d.MediaURL = item.URL
			d.PhotoPageURL = "https://www.flickr.com/photo.gne?id=" + photoID
			d.MediaType = "image"
			d.SuppressOG = true
			return d
		}

		// Case 2: Flickr photo page URL
		photoPageRe := regexp.MustCompile(`flickr\.com/photos/[^/]+/(\d+)`)
		photoMatches := photoPageRe.FindStringSubmatch(item.URL)
		if len(photoMatches) > 1 {
			imgURL := s.fetchFlickrImageURL(item.URL)
			if imgURL != "" {
				d.EmbedType = EmbedTypeFlickr
				d.MediaURL = imgURL
				d.MediaType = "image"
				d.SuppressOG = true
				return d
			}
		}
	}

	// Imgur - check before generic image handler
	if strings.Contains(item.URL, "imgur.com") {
		// Gallery Check
		if strings.Contains(item.URL, "/gallery/") || strings.Contains(item.URL, "/a/") {
			re := regexp.MustCompile(`imgur\.com/(?:gallery|a)/([a-zA-Z0-9]+)`)
			matches := re.FindStringSubmatch(item.URL)

			if len(matches) > 1 {
				// Try to get thumbnail from LinkPreview
				thumbnailURL := ""
				if s.Store != nil {
					if preview, err := s.Store.GetLinkPreview(context.TODO(), item.URL); err == nil && preview != nil {
						var meta map[string]string
						if err := json.Unmarshal(preview.Data, &meta); err == nil {
							if imgURL, ok := meta["image"]; ok && imgURL != "" {
								thumbnailURL = imgURL
							}
						}
					}
				}

				d.EmbedType = EmbedTypeImgurGallery
				d.ThumbnailURL = thumbnailURL
				d.IsBroken = thumbnailURL == ""
				d.SuppressOG = true
				return d
			}
		} else {
			// Single Image / Video Detection
			re := regexp.MustCompile(`imgur\.com/(?:.*[\\/-])?([a-zA-Z0-9]{5,})(?:\.(gif|gifv|jpg|jpeg|png|mp4))?`)
			matches := re.FindStringSubmatch(item.URL)
			if len(matches) > 1 {
				id := matches[1]
				ext := matches[2]
				// Default to .mp4 for animations
				if ext == "" {
					ext = "mp4"
				}

				d.EmbedType = EmbedTypeImgurSingle
				d.MediaURL = "https://i.imgur.com/" + id + "." + ext
				d.IsAnimated = ext == "mp4" || ext == "gifv" || ext == "gif"
				d.MediaType = "image"
				if d.IsAnimated {
					d.MediaType = "video"
				}
				d.SuppressOG = true
				return d
			}
		}
	}

	// Generic image content type check (after all site-specific handlers)
	// This handles direct image URLs that aren't from special sites
	if strings.Contains(item.ContentType, "image") && !d.IsNSFW {
		d.EmbedType = EmbedTypeImage
		d.MediaURL = item.URL
		d.MediaType = "image"
		return d
	}

	// Default: generic link
	d.EmbedType = EmbedTypeGeneric
	return d
}

func (s *ContentService) ProcessImage(item data.Image) DisplayItem {
	d := DisplayItem{
		ID:        item.ID,
		Type:      "image",
		Timestamp: item.Timestamp,
		Title:     item.Title,
		URL:       item.URL,
		BaseURL:   s.Config.BaseURL,
		EmbedType: EmbedTypeImage,
		MediaURL:  item.URL,
		MediaType: "image",
	}
	s.formatDate(&d)

	// Flickr Logic for Images
	if strings.Contains(item.URL, "flickr.com") {
		re := regexp.MustCompile(`\/([0-9]+)_[0-9a-z]+`)
		matches := re.FindStringSubmatch(item.URL)
		if len(matches) > 1 {
			photoID := matches[1]
			d.EmbedType = EmbedTypeFlickr
			d.PhotoPageURL = "https://www.flickr.com/photo.gne?id=" + photoID
		}
	}

	return d
}

func (s *ContentService) ProcessQuote(item data.Quote) DisplayItem {
	d := DisplayItem{
		ID:        item.ID,
		Type:      "quote",
		Timestamp: item.Timestamp,
		Author:    item.Author,
		Quote:     item.Quote,
		BaseURL:   s.Config.BaseURL,
		EmbedType: EmbedTypeQuote,
	}
	s.formatDate(&d)
	d.Description = item.Quote // For RSS usage
	return d
}

func (s *ContentService) formatDate(d *DisplayItem) {
	d.FormattedDate = d.Timestamp.Format("Mon, 02 Jan 2006 15:04:05 -0700")

	// Date components for grouping
	d.DateDay = d.Timestamp.Format("Mon")
	d.DateMonth = d.Timestamp.Format("Jan")
	d.DateRawDay = d.Timestamp.Format("02")
	d.DateYear = d.Timestamp.Format("2006")
	d.FullDate = d.Timestamp.Format("20060102") // For date grouping comparisons
}

// FetchOEmbed (Optional helper, untranslated for now due to API changes)
func (s *ContentService) fetchOEmbed(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	// Parse JSON...
	var r map[string]interface{}
	json.Unmarshal(body, &r)
	if html, ok := r["html"].(string); ok {
		return html, nil
	}
	return "", fmt.Errorf("no html")
}

// fetchFlickrImageURL fetches the direct image URL from Flickr's OEmbed API for a photo page URL
func (s *ContentService) fetchFlickrImageURL(photoURL string) string {
	// Build OEmbed URL
	oembedURL := fmt.Sprintf("https://www.flickr.com/services/oembed?url=%s&format=json", photoURL)

	// Create request
	req, err := http.NewRequest("GET", oembedURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return ""
	}

	// Parse response
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}

	// For photo type, extract the URL field
	if typeVal, ok := data["type"].(string); ok && typeVal == "photo" {
		if urlVal, ok := data["url"].(string); ok {
			return urlVal
		}
	}

	return ""
}

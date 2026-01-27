package service

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
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

type DisplayItem struct {
	ID            int           `json:"id"`
	Type          string        `json:"type"`
	Timestamp     time.Time     `json:"timestamp"`
	FormattedDate string        `json:"formatted_date"`
	User          string        `json:"user"`
	Author        string        `json:"author"`
	Title         string        `json:"title"`
	URL           string        `json:"url"`
	Clicks        int           `json:"clicks"`
	Content       template.HTML `json:"content"`
	Description   string        `json:"description,omitempty"`
	ContentType   string        `json:"content_type"`
	Quote         string        `json:"quote,omitempty"`
	BaseURL       string        `json:"base_url"`

	// Date components for grouping
	DateDay    string `json:"date_day"`     // e.g. "Mon"
	DateMonth  string `json:"date_month"`   // e.g. "Jan"
	DateRawDay string `json:"date_raw_day"` // e.g. "01"
	DateYear   string `json:"date_year"`    // e.g. "2026"
	SuppressOG bool   `json:"suppress_og"`
}

func NewContentService(cfg *config.Config, store data.Store) *ContentService {
	return &ContentService{Config: cfg, Store: store}
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
	}
	s.formatDate(&d)

	linkFiller := item.Title
	if len(item.Title) > 40 {
		if strings.HasPrefix(item.Title, "http://") {
			linkFiller = item.Title[:40] + "..."
		}
	}

	// Image check
	if strings.Contains(item.ContentType, "image") && !strings.Contains(item.User, "nsfw") && !strings.Contains(item.User, "otd") {
		// Add onerror handler to replace broken images with text
		linkFiller = fmt.Sprintf(`<img src="%s" onerror="this.parentNode.innerHTML='<span class=\'http-error-badge\'>404</span> <span class=\'missing-link\'>%s</span>'; this.parentNode.classList.add('missing-link');" />`, item.URL, item.URL)
	}

	isYoutube := false

	// Twitter / X
	isTwitter := false
	if strings.Contains(item.URL, "twitter.com") || strings.Contains(item.URL, "x.com") {
		// Extract ID to ensure it looks like a tweet URL
		re := regexp.MustCompile(`(?:twitter\.com|x\.com)\/.*\/status\/([0-9]+)`)
		matches := re.FindStringSubmatch(item.URL)
		if len(matches) > 1 {
			// standard embed code
			// Force twitter.com domain for embed compatibility as widgets.js might not support x.com fully yet
			embedURL := strings.Replace(item.URL, "x.com", "twitter.com", 1)
			embed := fmt.Sprintf(`<blockquote class="twitter-tweet"><a href="%s" target="_blank">%s</a></blockquote><script async src="https://platform.twitter.com/widgets.js" charset="utf-8"></script>`, embedURL, item.Title)
			d.Content = template.HTML(embed)
			isTwitter = true
			d.SuppressOG = false
		}
	}

	// Imgur
	isImgur := false
	if strings.Contains(item.URL, "imgur.com") {
		baseURL := s.Config.BaseURL

		// Gallery Check
		if strings.Contains(item.URL, "/gallery/") || strings.Contains(item.URL, "/a/") {
			// Extract gallery ID
			re := regexp.MustCompile(`imgur\.com/(?:gallery|a)/([a-zA-Z0-9]+)`)
			matches := re.FindStringSubmatch(item.URL)

			if len(matches) > 1 {
				baseURL := s.Config.BaseURL

				// Try to get thumbnail from LinkPreview (OpenGraph og:image)
				// Only render as gallery if we have a valid preview with an image
				thumbnailURL := ""
				if s.Store != nil {
					if preview, err := s.Store.GetLinkPreview(context.TODO(), item.URL); err == nil && preview != nil {
						var meta map[string]string
						if err := json.Unmarshal(preview.Data, &meta); err == nil {
							// Extract only the og:image, not description or other text
							if imgURL, ok := meta["image"]; ok && imgURL != "" {
								thumbnailURL = imgURL
							}
						}
					}
				}

				// Only render gallery card if we have a valid thumbnail
				// Otherwise, show 404 error for deleted/unavailable galleries
				if thumbnailURL != "" {
					// Build gallery card routing through IRC link handler
					// Detect and hide Imgur placeholder to maintain zero-tolerance requirement
					embed := fmt.Sprintf(
						`<span class="imgur-gallery-card">
							<a href="http://%s/irclink/?%d" target="_blank">
								<span class="gallery-image-container">
									<img src="%s"
										onload="if(this.naturalWidth===161 && this.naturalHeight===81){this.style.display='none'; this.nextElementSibling.style.display='block';}"
										onerror="this.style.display='none'; this.nextElementSibling.style.display='block';" />
									<span class="gallery-image-placeholder">📸</span>
								</span>
								<span class="gallery-card-content">
									<span class="gallery-card-title">Imgur Gallery</span>
									<span class="gallery-card-subtitle">%s</span>
								</span>
							</a>
						</span>`,
						baseURL, item.ID, thumbnailURL, item.Title)

					d.Content = template.HTML(embed)
					isImgur = true
					d.SuppressOG = true
				} else {
					// No valid preview - gallery is likely deleted/unavailable
					// Render as 404 error with gray link, same as other broken images
					d.Content = template.HTML(fmt.Sprintf(
						`<a href="http://%s/irclink/?%d" target="_blank"><span class='http-error-badge'>404</span> <span class='missing-link'>%s</span></a>`,
						baseURL, item.ID, item.URL))
					isImgur = true
					d.SuppressOG = true
				}
			}
		} else {
			// Single Image / Video Detection
			re := regexp.MustCompile(`imgur\.com/(?:.*[\\/-])?([a-zA-Z0-9]{5,})(?:\..*)?$`)
			matches := re.FindStringSubmatch(item.URL)
			if len(matches) > 1 {
				id := matches[1]
				imgURL := fmt.Sprintf("https://i.imgur.com/%s.jpg", id)

				// Render image wrapped in IRC link handler anchor
				// Use visibility toggle pattern (same as gallery) to avoid race conditions
				// Detect Imgur placeholder by dimensions (161x81px) or true 404 errors
				embed := fmt.Sprintf(
					`<a href="http://%s/irclink/?%d" target="_blank" style="display: inline-block; position: relative;">
						<img src="%s" style="max-width: 500px; display: block;"
							onload="if(this.naturalWidth===161 && this.naturalHeight===81){this.style.display='none'; this.nextElementSibling.style.display='inline';}"
							onerror="this.style.display='none'; this.nextElementSibling.style.display='inline';" />
						<span style="display: none;">
							<span class='http-error-badge'>404</span>
							<span class='missing-link'>%s</span>
						</span>
					</a>`,
					baseURL, item.ID, imgURL, item.URL)

				d.Content = template.HTML(embed)
				isImgur = true
				d.SuppressOG = true
			}
		}
	}

	// Flickr
	isFlickr := false
	if strings.Contains(item.URL, "flickr.com") {
		baseURL := s.Config.BaseURL
		imgURL := ""

		// Case 1: Static Flickr image URL (farm*.staticflickr.com)
		// Example: http://farm3.staticflickr.com/2362/2362225867_0a3b0b7e05.jpg
		re := regexp.MustCompile(`\/([0-9]+)_[0-9a-z]+`)
		matches := re.FindStringSubmatch(item.URL)
		if len(matches) > 1 {
			photoID := matches[1]
			photoPage := fmt.Sprintf("https://www.flickr.com/photo.gne?id=%s", photoID)
			// Use standard image tag but linked to photo page
			embed := fmt.Sprintf(`<a href="%s" target="_blank"><img src="%s" alt="%s" /></a>`, photoPage, item.URL, item.Title)
			d.Content = template.HTML(embed)
			isFlickr = true
			d.SuppressOG = true
		} else {
			// Case 2: Flickr photo page URL (www.flickr.com/photos/...)
			// Example: https://www.flickr.com/photos/cwage/402950834/
			photoPageRe := regexp.MustCompile(`flickr\.com/photos/[^/]+/(\d+)`)
			photoMatches := photoPageRe.FindStringSubmatch(item.URL)

			if len(photoMatches) > 1 {
				// Fetch image URL directly from Flickr's OEmbed API
				imgURL = s.fetchFlickrImageURL(item.URL)

				// If we have an image URL, render inline
				if imgURL != "" {
					embed := fmt.Sprintf(
						`<a href="http://%s/irclink/?%d" target="_blank"><img src="%s" style="max-width: 500px;" /></a>`,
						baseURL, item.ID, imgURL)
					d.Content = template.HTML(embed)
					isFlickr = true
					d.SuppressOG = true
				}
			}
		}
	}

	// YouTube logic removed: Handled client-side by OGPreview for "click to play" behavior
	// and to correctly handle unavailable videos (404s).

	if !isYoutube && !isTwitter && !isImgur && !isFlickr {
		baseURL := s.Config.BaseURL
		content := fmt.Sprintf(`<a href="http://%s/irclink/?%d" target="_blank">%s</a>`, baseURL, item.ID, linkFiller)
		d.Content = template.HTML(content)
	}

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
	}
	s.formatDate(&d)

	// Flickr Logic for Images
	if strings.Contains(item.URL, "flickr.com") {
		// Attempt to extract photo ID from URL
		re := regexp.MustCompile(`\/([0-9]+)_[0-9a-z]+`)
		matches := re.FindStringSubmatch(item.URL)
		if len(matches) > 1 {
			photoID := matches[1]
			photoPage := fmt.Sprintf("https://www.flickr.com/photo.gne?id=%s", photoID)
			// Linked Thumbnail
			d.Content = template.HTML(fmt.Sprintf(`<a href="%s" target="_blank"><img src="%s" alt="image" /></a>`, photoPage, item.URL))
			return d
		}
	}

	d.Content = template.HTML(fmt.Sprintf(`<img src="%s" alt="image" />`, item.URL))
	return d
}

func (s *ContentService) ProcessQuote(item data.Quote) DisplayItem {
	d := DisplayItem{
		ID:        item.ID,
		Type:      "quote",
		Timestamp: item.Timestamp,
		Author:    item.Author, // Quote author field
		Quote:     item.Quote,
		BaseURL:   s.Config.BaseURL,
	}
	s.formatDate(&d)
	// For quotes, content is text + author
	d.Content = template.HTML(fmt.Sprintf(`"%s" --%s`, item.Quote, item.Author))
	d.Description = item.Quote // For separate usage
	return d
}

func (s *ContentService) formatDate(d *DisplayItem) {
	// Replicate Perl's timezone and formatting logic if needed.
	// Perl: "Sun, 04 Jan 2026 15:04:05 +0000"
	// Go's time.Time is already aware. we just format it.
	d.FormattedDate = d.Timestamp.Format("Mon, 02 Jan 2006 15:04:05 -0700")

	// Date components for grouping
	d.DateDay = d.Timestamp.Format("Mon")
	d.DateMonth = d.Timestamp.Format("Jan")
	d.DateRawDay = d.Timestamp.Format("02")
	d.DateYear = d.Timestamp.Format("2006")
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

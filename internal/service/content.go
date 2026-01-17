package service

import (
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
}

func NewContentService(cfg *config.Config) *ContentService {
	return &ContentService{Config: cfg}
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
		linkFiller = fmt.Sprintf(`<img src="%s" />`, item.URL)
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
			embed := fmt.Sprintf(`<blockquote class="twitter-tweet"><a href="%s"></a></blockquote><script async src="https://platform.twitter.com/widgets.js" charset="utf-8"></script>`, item.URL)
			d.Content = template.HTML(embed)
			isTwitter = true
		}
	}

	// Imgur
	isImgur := false
	if strings.Contains(item.URL, "imgur.com") {
		// Regex for ID extraction
		re := regexp.MustCompile(`imgur\.com\/(?:.*[\/-])?([a-zA-Z0-9]{5,})(?:\..*)?$`)
		matches := re.FindStringSubmatch(item.URL)
		if len(matches) > 1 {
			id := matches[1]
			videoURL := fmt.Sprintf("https://i.imgur.com/%s.mp4", id)
			imgURL := fmt.Sprintf("https://i.imgur.com/%s.jpg", id)

			// We render a video tag by default. If it fails to load (404 for static images, or other errors),
			// the onerror handler swaps it for a standard image tag.
			// This avoids server-side rate limits (HTTP 429) and speeds up response time.
			// Note: We wrap it in the anchor tag in the Go code, but the onerror replaces the VIDEO tag specifically.
			embed := fmt.Sprintf(
				`<a href="%s"><video autoplay loop muted playsinline style="max-width: 500px;" src="%s" onerror="this.onerror=null;this.outerHTML='<img src=\'%s\' style=\'max-width: 500px;\' />'"></video></a>`,
				item.URL, videoURL, imgURL)

			d.Content = template.HTML(embed)
			isImgur = true
		}
	}

	// YouTube logic removed: Handled client-side by OGPreview for "click to play" behavior
	// and to correctly handle unavailable videos (404s).

	if !isYoutube && !isTwitter && !isImgur {
		baseURL := s.Config.BaseURL
		content := fmt.Sprintf(`<a href="http://%s/irclink/?%d">%s</a>`, baseURL, item.ID, linkFiller)
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

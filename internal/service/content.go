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

	// YouTube
	// Supports: youtube.com/watch?v=, embed/, youtu.be/
	videoID := ""
	if strings.Contains(strings.ToLower(item.URL), "youtube.com") || strings.Contains(strings.ToLower(item.URL), "youtu.be") {
		re := regexp.MustCompile(`(?:youtube\.com\/watch\?v=|youtube\.com\/embed\/|youtu\.be\/)([a-zA-Z0-9_-]{11})`)
		matches := re.FindStringSubmatch(item.URL)
		if len(matches) > 1 {
			videoID = matches[1]
		} else {
			// Try query param
			re2 := regexp.MustCompile(`youtube\.com\/watch\?.*[&?]v=([a-zA-Z0-9_-]{11})`)
			matches2 := re2.FindStringSubmatch(item.URL)
			if len(matches2) > 1 {
				videoID = matches2[1]
			}
		}
	}

	if videoID != "" {
		embed := fmt.Sprintf(`<div class="youtube-embed-wrapper"><iframe width="560" height="315" src="https://www.youtube.com/embed/%s?rel=0" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen="allowfullscreen"></iframe></div>`, videoID)
		d.Content = template.HTML(embed)
		isYoutube = true
	}

	if !isYoutube && !isTwitter {
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

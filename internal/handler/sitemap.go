package handler

import (
	"encoding/xml"
	"fmt"
	"net/http"
)

// SitemapURLSet represents the root element of a sitemap XML.
type SitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []SitemapURL `xml:"url"`
}

// SitemapURL represents a single URL entry in a sitemap.
type SitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

// SitemapHandler generates a dynamic sitemap.xml.
func (h *Handler) SitemapHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := h.Config.BaseURL
	urls := []SitemapURL{}

	// Homepage - highest priority, changes frequently
	urls = append(urls, SitemapURL{
		Loc:        baseURL + "/",
		ChangeFreq: "hourly",
		Priority:   "1.0",
	})

	// Static pages
	urls = append(urls, SitemapURL{
		Loc:        baseURL + "/stats",
		ChangeFreq: "daily",
		Priority:   "0.5",
	})

	urls = append(urls, SitemapURL{
		Loc:        baseURL + "/search",
		ChangeFreq: "weekly",
		Priority:   "0.3",
	})

	// Date-based archive pages for last 30 days (~5 pages of 6 days each)
	// Skip page 1 since homepage already covers it
	for page := 2; page <= 5; page++ {
		from, to := pageToDateRange(page)

		// Calculate priority - decreases with age
		priority := fmt.Sprintf("%.1f", 0.9-float64(page-2)*0.1)

		urls = append(urls, SitemapURL{
			Loc:        baseURL + "/?from=" + from.Format("2006-01-02") + "&to=" + to.Format("2006-01-02"),
			LastMod:    to.Format("2006-01-02"),
			ChangeFreq: "weekly",
			Priority:   priority,
		})
	}

	sitemap := SitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(sitemap); err != nil {
		http.Error(w, "Error generating sitemap", http.StatusInternalServerError)
	}
}

// RobotsHandler serves a dynamic robots.txt that includes the sitemap URL.
func (h *Handler) RobotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	fmt.Fprintf(w, "User-agent: *\nDisallow:\n\nSitemap: %s/sitemap.xml\n", h.Config.BaseURL)
}

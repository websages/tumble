package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"tumble/internal/activitypub"
	"tumble/internal/archive"
	"tumble/internal/config"
	"tumble/internal/data"
	"tumble/internal/service"
	"tumble/internal/templates"
	"tumble/internal/version"
)

type Handler struct {
	Store         data.Store
	Service       *service.ContentService
	Renderer      *templates.Renderer
	Config        *config.Config
	ArchiveClient *archive.Client
	ActivityPub   *activitypub.Service
}

func NewHandler(cfg *config.Config, store data.Store, svc *service.ContentService, renderer *templates.Renderer) *Handler {
	return &Handler{
		Config:        cfg,
		Store:         store,
		Service:       svc,
		Renderer:      renderer,
		ArchiveClient: archive.NewClient(5),
		ActivityPub:   activitypub.NewService(cfg, store),
	}
}

func (h *Handler) ServerError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("Internal Server Error", "method", r.Method, "path", r.URL.Path, "query", r.URL.RawQuery, "error", err)

	w.WriteHeader(http.StatusInternalServerError)

	if h.Config.Mode == "development" {
		fmt.Fprintf(w, "Internal Server Error: %s", err.Error())
	} else {
		fmt.Fprint(w, "Internal Server Error")
	}
}

// Input validation constants
const (
	maxPosterLength = 256
	maxSearchLength = 500
)

// validFilterTypes is the allowlist for filter type parameter
var validFilterTypes = map[string]bool{
	"":       true,
	"links":  true,
	"quotes": true,
}

// sanitizePoster truncates poster to max length
func sanitizePoster(poster string) string {
	if len(poster) > maxPosterLength {
		return poster[:maxPosterLength]
	}
	return poster
}

// sanitizeFilterType returns the filter type if valid, empty string otherwise
func sanitizeFilterType(filterType string) string {
	if validFilterTypes[filterType] {
		return filterType
	}
	return ""
}

// sanitizeSearch truncates search query to max length
func sanitizeSearch(query string) string {
	if len(query) > maxSearchLength {
		return query[:maxSearchLength]
	}
	return query
}

// IndexPageData is the data structure for the main template
type IndexPageData struct {
	PageTitle         string
	Hot               template.HTML
	Container         template.HTML
	NavP              template.HTML
	NavN              template.HTML
	GitCommit         string
	GitCommitURL      string
	BaseURL           template.HTML
	Poster            string
	FilterType        string
	IsFallbackContent bool
	CanonicalURL      string
	DevMode           bool
	SiteTitle         string
	SiteDescription   string
	ManagingEditor    string
	WebMaster         string
	Copyright         string
}

// NavigationData holds pagination navigation info
type NavigationData struct {
	PrevPage    int
	NextPage    int
	CurrentPage int
	HasPrev     bool
	HasNext     bool
	PosterParam string
}

// HotLinkItem is a simplified struct for hot links display
type HotLinkItem struct {
	ID       int
	Title    string
	BaseURL  string
	ClickSig string
}

// getHotLinks returns hot link data for templates
func (h *Handler) getHotLinks(ctx context.Context) []HotLinkItem {
	topLinks, err := h.Store.GetTopIRCLinks(ctx, 12, 6, 5)
	if err != nil {
		slog.Error("Failed to get top links", "error", err)
		return nil
	}

	items := make([]HotLinkItem, 0, len(topLinks))
	for _, l := range topLinks {
		items = append(items, HotLinkItem{
			ID:       l.ID,
			Title:    l.Title,
			BaseURL:  h.Config.BaseURL,
			ClickSig: GenerateClickSignature(l.ID, h.Config.ClickSigningKey),
		})
	}
	return items
}

// getHotHTML renders hot links to HTML (for backwards compatibility during transition)
func (h *Handler) getHotHTML(ctx context.Context) template.HTML {
	hotLinks := h.getHotLinks(ctx)
	if len(hotLinks) == 0 {
		return ""
	}

	hotHTML := ""
	for _, link := range hotLinks {
		s, _ := h.Renderer.RenderToString("tumble_item_top5.html", link)
		hotHTML += s
	}
	return template.HTML(hotHTML)
}

// DateGroup represents a group of items for a single date
type DateGroup struct {
	FullDate   string
	DateRawDay string
	DateDay    string
	DateMonth  string
	DateYear   string
	Items      []service.DisplayItem
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parameters
	params := r.URL.Query()
	dtype := params.Get("dtype")
	iParam := params.Get("i")
	fromParam := params.Get("from")
	toParam := params.Get("to")

	// Infer dtype from path if not set
	if dtype == "" {
		if r.URL.Path == "/index.xml" || r.URL.Path == "/index.rss" {
			dtype = "xml"
		}
	}

	var startDays, endDays int
	var i int = 1
	var usingDateRange bool

	// Check for date-based URL parameters first
	if fromParam != "" && toParam != "" {
		fromDate, fromErr := time.Parse("2006-01-02", fromParam)
		toDate, toErr := time.Parse("2006-01-02", toParam)
		if fromErr == nil && toErr == nil {
			// Convert dates to days offset from today
			now := time.Now().Truncate(24 * time.Hour)
			startDays = int(now.Sub(fromDate).Hours() / 24)
			endDays = int(now.Sub(toDate).Hours() / 24)
			usingDateRange = true
		}
	}

	// Fall back to page-based pagination if no valid date range
	if !usingDateRange {
		if iParam != "" {
			val, err := strconv.Atoi(iParam)
			if err == nil && val > 0 {
				i = val
			}
		}
		// Date interval logic based on page number
		startDays = i * 6
		endDays = (i - 1) * 6
	}

	// Fetch Items
	var wg sync.WaitGroup
	var errIrc, errImg, errQuote error
	var ircLinks []data.IRCLink
	var images []data.Image
	var quotes []data.Quote

	poster := sanitizePoster(params.Get("poster"))
	filterType := sanitizeFilterType(params.Get("type"))
	isFallback := false

	if poster != "" {
		// Filtered View: Only links/quotes by 'poster'
		limit := 30
		offset := (i - 1) * 30

		timelineItems, err := h.Store.GetUserTimeline(ctx, poster, filterType, limit, offset)
		if err != nil {
			errIrc = err
		} else {
			for _, item := range timelineItems {
				if item.Type == "link" {
					ircLinks = append(ircLinks, data.IRCLink{
						ID:          item.ID,
						Timestamp:   item.Timestamp,
						User:        poster,
						Title:       item.Title,
						URL:         item.URL,
						ContentType: item.ContentType,
					})
				} else if item.Type == "quote" {
					quotes = append(quotes, data.Quote{
						ID:        item.ID,
						Timestamp: item.Timestamp,
						Author:    poster,
						Quote:     item.Content,
					})
				}
			}
		}
	} else {
		// Standard View
		wg.Add(3)
		go func() {
			defer wg.Done()
			ircLinks, errIrc = h.Store.GetRecentIRCLinks(ctx, startDays, endDays, data.ClientFilter{})
		}()
		go func() {
			defer wg.Done()
			images, errImg = h.Store.GetRecentImages(ctx, startDays, endDays, data.ClientFilter{})
		}()
		go func() {
			defer wg.Done()
			quotes, errQuote = h.Store.GetRecentQuotes(ctx, startDays, endDays, data.ClientFilter{})
		}()
		wg.Wait()
	}

	if errIrc != nil || errImg != nil || errQuote != nil {
		err := fmt.Errorf("irc: %v, img: %v, quote: %v", errIrc, errImg, errQuote)
		h.ServerError(w, r, err)
		return
	}

	// Check for empty state on front page
	if poster == "" && i == 1 && len(ircLinks) == 0 && len(images) == 0 && len(quotes) == 0 {
		slog.Info("No recent content found, fetching global timeline fallback")
		fallbackItems, err := h.Store.GetGlobalTimeline(ctx, 20, 0)
		if err == nil {
			isFallback = true
			for _, item := range fallbackItems {
				switch item.Type {
				case "link":
					ircLinks = append(ircLinks, data.IRCLink{
						ID:        item.ID,
						Timestamp: item.Timestamp,
						User:      item.Author,
						Title:     item.Title,
						URL:       item.URL,
					})
				case "quote":
					quotes = append(quotes, data.Quote{
						ID:        item.ID,
						Timestamp: item.Timestamp,
						Author:    item.Author,
						Quote:     item.Content,
					})
				case "image":
					images = append(images, data.Image{
						ID:        item.ID,
						Timestamp: item.Timestamp,
						Title:     item.Title,
						Link:      item.URL,
						URL:       item.URL,
						MD5Sum:    item.MD5Sum,
					})
				}
			}
		} else {
			slog.Error("Error fetching global timeline fallback", "error", err)
		}
	}

	// Process all items into DisplayItems
	var allItems []service.DisplayItem

	for _, item := range ircLinks {
		allItems = append(allItems, h.Service.ProcessIRCLink(item))
	}
	for _, item := range images {
		allItems = append(allItems, h.Service.ProcessImage(item))
	}
	for _, item := range quotes {
		allItems = append(allItems, h.Service.ProcessQuote(item))
	}

	// Sort items by timestamp descending
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Timestamp.After(allItems[j].Timestamp)
	})

	// Render items to HTML (still using RenderToString for now, but templates handle the logic)
	containerHTML := ""
	lastDate := ""
	sectionOpen := false

	for _, item := range allItems {
		// Select template based on item type and output format
		var tmplName string
		switch item.Type {
		case "ircLink":
			tmplName = "tumble_item_ircLink.html"
			if dtype == "rss" || dtype == "xml" {
				tmplName = "tumble_item_ircLink.xml"
			}
		case "image":
			tmplName = "tumble_item_image.html"
			if dtype == "rss" || dtype == "xml" {
				tmplName = "tumble_item_image.xml"
			}
		case "quote":
			tmplName = "tumble_item_quote.html"
			if dtype == "rss" || dtype == "xml" {
				tmplName = "tumble_item_quote.xml"
			}
		}

		// For HTML output, handle date grouping
		if dtype != "rss" && dtype != "xml" {
			if item.FullDate != lastDate {
				if sectionOpen {
					containerHTML += "</div>"
				}
				containerHTML += fmt.Sprintf(`<div class="date-section" data-date="%s">`, item.FullDate)
				sectionOpen = true

				dateData := map[string]string{
					"Date":  item.DateRawDay,
					"Day":   item.DateDay,
					"Month": item.DateMonth,
					"Year":  item.DateYear,
				}
				dateHTML, err := h.Renderer.RenderToString("tumble_date.html", dateData)
				if err == nil {
					containerHTML += dateHTML
				}
				lastDate = item.FullDate
			}
		}

		html, err := h.Renderer.RenderToString(tmplName, item)
		if err != nil {
			slog.Debug("Render Error", "type", item.Type, "id", item.ID, "error", err)
			continue
		}
		containerHTML += html
	}

	if sectionOpen {
		containerHTML += "</div>"
	}

	// Hot Links (Side bar) - Only for HTML
	hotHTML := template.HTML("")
	if dtype != "rss" && dtype != "xml" {
		hotHTML = h.getHotHTML(ctx)
	}

	// Navigation - using template.HTML for now, will move to template later
	nav := h.buildNavigation(i, poster, filterType)

	// View Data
	pageTitle := ""
	if poster != "" {
		pageTitle = fmt.Sprintf(" &gt; Links by %s", poster)
	}

	// Build canonical URL - use date params if provided, otherwise convert from page number
	var canonicalURL string
	if usingDateRange {
		canonicalURL = h.buildCanonicalURL(i, poster, filterType, fromParam, toParam)
	} else {
		canonicalURL = h.buildCanonicalURL(i, poster, filterType, "", "")
	}

	viewData := IndexPageData{
		PageTitle:         pageTitle,
		Container:         template.HTML(containerHTML),
		Hot:               hotHTML,
		NavP:              template.HTML(nav.prevHTML),
		NavN:              template.HTML(nav.nextHTML),
		BaseURL:           template.HTML(h.Config.BaseURL),
		Poster:            poster,
		FilterType:        filterType,
		GitCommit:         version.CommitHash,
		GitCommitURL:      fmt.Sprintf("https://github.com/websages/tumble/commit/%s", version.CommitHash),
		IsFallbackContent: isFallback,
		CanonicalURL:      canonicalURL,
		DevMode:           h.Config.Mode == "development",
		SiteTitle:         h.Config.SiteTitle,
		SiteDescription:   h.Config.SiteDescription,
		ManagingEditor:    h.Config.ManagingEditor,
		WebMaster:         h.Config.WebMaster,
		Copyright:         h.Config.Copyright,
	}

	templateName := "index.html"
	contentType := "text/html; charset=UTF-8"
	if dtype == "rss" || dtype == "xml" {
		templateName = "index.xml"
		contentType = "text/xml; charset=UTF-8"
	}

	w.Header().Set("Content-Type", contentType)
	if err := h.Renderer.Render(w, templateName, viewData); err != nil {
		slog.Error("Error rendering template", "template", templateName, "error", err)
	}
}

type navResult struct {
	prevHTML string
	nextHTML string
}

func (h *Handler) buildNavigation(page int, poster, filterType string) navResult {
	posterParam := ""
	if poster != "" {
		posterParam = fmt.Sprintf("&poster=%s", poster)
		if filterType != "" {
			posterParam += fmt.Sprintf("&type=%s", filterType)
		}
	}

	var prevHTML, nextHTML string

	if page > 1 {
		prevHTML = fmt.Sprintf(`<a href="?i=%d%s" class="nav-link"><span class="material-symbols-rounded nav-icon">chevron_left</span></a>`, page+1, posterParam)
		nextHTML = fmt.Sprintf(`<a href="?i=%d%s" class="nav-link"><span class="material-symbols-rounded nav-icon">chevron_right</span></a>`, page-1, posterParam)
	} else {
		prevHTML = fmt.Sprintf(`<a href="?i=2%s" class="nav-link"><span class="material-symbols-rounded nav-icon">chevron_left</span></a>`, posterParam)
		nextHTML = ""
	}

	return navResult{prevHTML: prevHTML, nextHTML: nextHTML}
}

// pageToDateRange converts a relative page number to a date range.
// Page 1 is the most recent 6 days, page 2 is days 7-12, etc.
func pageToDateRange(page int) (from, to time.Time) {
	now := time.Now().Truncate(24 * time.Hour)
	startDays := page * 6
	endDays := (page - 1) * 6
	from = now.AddDate(0, 0, -startDays)
	to = now.AddDate(0, 0, -endDays)
	return from, to
}

// buildCanonicalURL constructs the canonical URL for the current page.
// If fromDate and toDate are provided (non-empty), they are used directly.
// Otherwise, the page number is converted to a date range.
func (h *Handler) buildCanonicalURL(page int, poster, filterType, fromDate, toDate string) string {
	base := h.Config.BaseURL

	// User filter pages - canonical is the filter URL itself
	if poster != "" {
		canonical := base + "/?poster=" + url.QueryEscape(poster)
		if filterType != "" {
			canonical += "&type=" + url.QueryEscape(filterType)
		}
		return canonical
	}

	// If date range was explicitly provided, use it as-is (self-referential canonical)
	if fromDate != "" && toDate != "" {
		return base + "/?from=" + fromDate + "&to=" + toDate
	}

	// Homepage (page 1) - canonical is root
	if page <= 1 {
		return base + "/"
	}

	// Older pages - convert to date-based canonical
	from, to := pageToDateRange(page)
	return base + "/?from=" + from.Format("2006-01-02") + "&to=" + to.Format("2006-01-02")
}

// buildStatsCanonicalURL constructs the canonical URL for stats pages.
func (h *Handler) buildStatsCanonicalURL(view, sortBy string, page int) string {
	base := h.Config.BaseURL + "/stats"
	params := url.Values{}

	// Only include non-default values
	if view != "" && view != "users" {
		params.Set("view", view)
	}
	if sortBy != "" && sortBy != "links" {
		params.Set("sort", sortBy)
	}
	if page > 1 {
		params.Set("page", strconv.Itoa(page))
	}

	if len(params) > 0 {
		return base + "?" + params.Encode()
	}
	return base
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := sanitizeSearch(r.URL.Query().Get("search"))

	if query == "" {
		return
	}

	links, err := h.Store.SearchIRCLinks(ctx, query, data.ClientFilter{})
	if err != nil {
		h.ServerError(w, r, err)
		return
	}

	quotes, err := h.Store.SearchQuotes(ctx, query, data.ClientFilter{})
	if err != nil {
		h.ServerError(w, r, err)
		return
	}

	containerHTML := ""
	if len(links) > 0 || len(quotes) > 0 {
		// Process all items into DisplayItems
		var allItems []service.DisplayItem
		for _, item := range links {
			allItems = append(allItems, h.Service.ProcessIRCLink(item))
		}
		for _, item := range quotes {
			allItems = append(allItems, h.Service.ProcessQuote(item))
		}

		// Sort by timestamp descending (newest first)
		sort.Slice(allItems, func(i, j int) bool {
			return allItems[i].Timestamp.After(allItems[j].Timestamp)
		})

		// Render items with date grouping
		lastDate := ""
		sectionOpen := false
		for _, item := range allItems {
			var tmplName string
			switch item.Type {
			case "ircLink":
				tmplName = "tumble_item_ircLink.html"
			case "quote":
				tmplName = "tumble_item_quote.html"
			}

			if item.FullDate != lastDate {
				if sectionOpen {
					containerHTML += "</div>"
				}
				containerHTML += fmt.Sprintf(`<div class="date-section" data-date="%s">`, item.FullDate)
				sectionOpen = true

				dateData := map[string]string{
					"Date":  item.DateRawDay,
					"Day":   item.DateDay,
					"Month": item.DateMonth,
					"Year":  item.DateYear,
				}
				dateHTML, err := h.Renderer.RenderToString("tumble_date.html", dateData)
				if err == nil {
					containerHTML += dateHTML
				}
				lastDate = item.FullDate
			}

			s, err := h.Renderer.RenderToString(tmplName, item)
			if err != nil {
				slog.Debug("Render Error", "type", item.Type, "id", item.ID, "error", err)
				continue
			}
			containerHTML += s
		}
		if sectionOpen {
			containerHTML += "</div>"
		}
	} else {
		// Use the new no_search_results template
		data := map[string]interface{}{
			"Query": query,
		}
		containerHTML, _ = h.Renderer.RenderToString("no_search_results.html", data)
	}

	hotHTML := h.getHotHTML(ctx)

	viewData := IndexPageData{
		PageTitle:    fmt.Sprintf(" &gt; %s", query),
		Container:    template.HTML(containerHTML),
		Hot:          hotHTML,
		BaseURL:      template.HTML(h.Config.BaseURL),
		GitCommit:    version.CommitHash,
		GitCommitURL: fmt.Sprintf("https://github.com/websages/tumble/commit/%s", version.CommitHash),
		CanonicalURL: h.Config.BaseURL + "/search?search=" + url.QueryEscape(query),
		DevMode:      h.Config.Mode == "development",
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	h.Renderer.Render(w, "index.html", viewData)
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	view := r.URL.Query().Get("view")
	if view == "" {
		view = "users"
	}

	page := 1
	pageParam := r.URL.Query().Get("page")
	if pageParam != "" {
		val, err := strconv.Atoi(pageParam)
		if err == nil && val > 0 {
			page = val
		}
	}
	limit := 50
	offset := (page - 1) * limit

	nextPage := page + 1
	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 0
	}

	var data map[string]interface{}

	if view == "links" {
		links, err := h.Store.GetLinksByPopularity(ctx, limit, offset)
		if err != nil {
			h.ServerError(w, r, err)
			return
		}

		type LinkViewItem struct {
			Rank        int
			ID          int
			User        string
			Title       string
			URL         string
			Clicks      int
			Timestamp   time.Time
			ContentType string
			BaseURL     string
		}

		var linksView []LinkViewItem
		for i, link := range links {
			linksView = append(linksView, LinkViewItem{
				Rank:        offset + i + 1,
				ID:          link.ID,
				User:        link.User,
				Title:       link.Title,
				URL:         link.URL,
				Clicks:      link.Clicks,
				Timestamp:   link.Timestamp,
				ContentType: link.ContentType,
				BaseURL:     h.Config.BaseURL,
			})
		}

		hasNext := len(links) == limit

		data = map[string]interface{}{
			"Links":        linksView,
			"PageTitle":    " &gt; Stats &gt; Link Popularity",
			"GitCommit":    version.CommitHash,
			"GitCommitURL": fmt.Sprintf("https://github.com/websages/tumble/commit/%s", version.CommitHash),
			"Page":         page,
			"NextPage":     nextPage,
			"PrevPage":     prevPage,
			"HasNext":      hasNext,
			"View":         view,
			"Hot":          h.getHotHTML(ctx),
			"BaseURL":      h.Config.BaseURL,
			"CanonicalURL": h.buildStatsCanonicalURL(view, "", page),
		}
	} else {
		sortBy := r.URL.Query().Get("sort")
		if sortBy == "" {
			sortBy = "links"
		}

		stats, err := h.Store.GetUserStats(ctx, sortBy, limit, offset)
		if err != nil {
			h.ServerError(w, r, err)
			return
		}

		type StatViewItem struct {
			Rank       int
			User       string
			LinkCount  int
			QuoteCount int
		}

		var statsView []StatViewItem
		for i, s := range stats {
			statsView = append(statsView, StatViewItem{
				Rank:       offset + i + 1,
				User:       s.User,
				LinkCount:  s.LinkCount,
				QuoteCount: s.QuoteCount,
			})
		}

		hasNext := len(stats) == limit

		data = map[string]interface{}{
			"Stats":        statsView,
			"PageTitle":    " &gt; Stats",
			"GitCommit":    version.CommitHash,
			"GitCommitURL": fmt.Sprintf("https://github.com/websages/tumble/commit/%s", version.CommitHash),
			"Page":         page,
			"NextPage":     nextPage,
			"PrevPage":     prevPage,
			"HasNext":      hasNext,
			"Sort":         sortBy,
			"View":         view,
			"Hot":          h.getHotHTML(ctx),
			"BaseURL":      h.Config.BaseURL,
			"CanonicalURL": h.buildStatsCanonicalURL(view, sortBy, page),
		}
	}

	if err := h.Renderer.Render(w, "stats.html", data); err != nil {
		slog.Error("Error rendering stats", "error", err)
	}
}

// StatsJSON returns user statistics as JSON with pagination support
func (h *Handler) StatsJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters
	limit := 50
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if val, err := strconv.Atoi(limitParam); err == nil && val > 0 && val <= 1000 {
			limit = val
		}
	}

	offset := 0
	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if val, err := strconv.Atoi(offsetParam); err == nil && val >= 0 {
			offset = val
		}
	}

	stats, err := h.Store.GetUserStats(ctx, "links", limit, offset)
	if err != nil {
		h.ServerError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		slog.Error("Error encoding stats JSON", "error", err)
	}
}

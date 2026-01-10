package handler

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"

	"tumble/internal/config"
	"tumble/internal/data"
	"tumble/internal/service"
	"tumble/internal/templates"
	"tumble/internal/version"
)

type Handler struct {
	Store    data.Store
	Service  *service.ContentService
	Renderer *templates.Renderer
	Config   *config.Config
}

func NewHandler(cfg *config.Config, store data.Store, svc *service.ContentService, renderer *templates.Renderer) *Handler {
	return &Handler{
		Config:   cfg,
		Store:    store,
		Service:  svc,
		Renderer: renderer,
	}
}

// Index Page Data structure for the main template
type IndexPageData struct {
	PageTitle    string
	Hot          template.HTML
	Container    template.HTML
	NavP         template.HTML
	NavN         template.HTML
	GitCommit    string // Placeholder
	GitCommitURL string // Placeholder
	// For XML
	BaseURL template.HTML
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parameters
	params := r.URL.Query()
	dtype := params.Get("dtype")
	iParam := params.Get("i")

	i := 1
	if iParam != "" {
		val, err := strconv.Atoi(iParam)
		if err == nil && val > 0 {
			i = val
		}
	}

	// Date interval logic:
	// Perl: start_days = i * 6, end_days = (i - 1) * 6
	startDays := i * 6
	endDays := (i - 1) * 6

	// Fetch Items
	var wg sync.WaitGroup
	var errIrc, errImg, errQuote error
	var ircLinks []data.IRCLink
	var images []data.Image
	var quotes []data.Quote

	wg.Add(3)
	go func() { defer wg.Done(); ircLinks, errIrc = h.Store.GetRecentIRCLinks(ctx, startDays, endDays) }()
	go func() { defer wg.Done(); images, errImg = h.Store.GetRecentImages(ctx, startDays, endDays) }()
	go func() { defer wg.Done(); quotes, errQuote = h.Store.GetRecentQuotes(ctx, startDays, endDays) }()
	wg.Wait()

	if errIrc != nil || errImg != nil || errQuote != nil {
		log.Printf("Error fetching data: %v %v %v", errIrc, errImg, errQuote)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Combine and Sort
	// Since we fetched by specific intervals, we just need to merge and sort by timestamp.
	// OR we can process them all and then sort.
	// But simply rendering them in memory and then concatenating is easier if we process them in time order.
	// Perl simply assumes they are ordered by key timestamp in the hash?
	// Actually Perl: `foreach my $item_id ( reverse sort { $a cmp $b } keys %{$data} )`
	// The keys are timestamps (Wait, `key => 'timestamp'` in fetch means keys are timestamps).
	// So items are sorted by timestamp.

	type ProcessedItem struct {
		Timestamp  string // for sorting
		HTML       string
		DateRawDay string
		DateDay    string
		DateMonth  string
		FullDate   string // YYYYMMDD for comparison
	}

	processedItems := []ProcessedItem{}

	// Process IRCLinks
	for _, item := range ircLinks {
		d := h.Service.ProcessIRCLink(item)
		tmplName := "tumble_item_ircLink.html"
		if dtype == "rss" || dtype == "xml" {
			tmplName = "tumble_item_ircLink.xml"
		}

		html, err := h.Renderer.RenderToString(tmplName, d)
		if err == nil {
			processedItems = append(processedItems, ProcessedItem{
				Timestamp:  item.Timestamp.Format("20060102150405"), // Sortable string
				HTML:       html,
				DateRawDay: d.DateRawDay,
				DateDay:    d.DateDay,
				DateMonth:  d.DateMonth,
				FullDate:   item.Timestamp.Format("20060102"),
			})
		} else {
			log.Printf("DEBUG: Render Error for Link %d: %v", item.ID, err)
		}
	}

	// Process Images
	for _, item := range images {
		d := h.Service.ProcessImage(item)
		tmplName := "tumble_item_image.html"
		if dtype == "rss" || dtype == "xml" {
			tmplName = "tumble_item_image.xml"
		}

		html, err := h.Renderer.RenderToString(tmplName, d)
		if err == nil {
			processedItems = append(processedItems, ProcessedItem{
				Timestamp:  item.Timestamp.Format("20060102150405"),
				HTML:       html,
				DateRawDay: d.DateRawDay,
				DateDay:    d.DateDay,
				DateMonth:  d.DateMonth,
				FullDate:   item.Timestamp.Format("20060102"),
			})
		}
	}

	// Process Quotes
	for _, item := range quotes {
		d := h.Service.ProcessQuote(item)
		tmplName := "tumble_item_quote.html"
		if dtype == "rss" || dtype == "xml" {
			tmplName = "tumble_item_quote.xml"
		}

		html, err := h.Renderer.RenderToString(tmplName, d)
		if err == nil {
			processedItems = append(processedItems, ProcessedItem{
				Timestamp:  item.Timestamp.Format("20060102150405"),
				HTML:       html,
				DateRawDay: d.DateRawDay,
				DateDay:    d.DateDay,
				DateMonth:  d.DateMonth,
				FullDate:   item.Timestamp.Format("20060102"),
			})
		}
	}

	// Sort items (descending)
	// Simple bubble sort or whatever for small lists, or sort packages.
	// For "100% compatibility" I must sort descending.
	for j := 0; j < len(processedItems); j++ {
		for k := j + 1; k < len(processedItems); k++ {
			if processedItems[j].Timestamp < processedItems[k].Timestamp {
				processedItems[j], processedItems[k] = processedItems[k], processedItems[j]
			}
		}
	}

	// Generate Container HTML
	containerHTML := ""
	lastDate := ""

	for _, p := range processedItems {
		if dtype != "rss" && dtype != "xml" {
			if p.FullDate != lastDate {
				// Date Changed, Render Date Template
				dateData := map[string]string{
					"Date":  p.DateRawDay,
					"Day":   p.DateDay,
					"Month": p.DateMonth,
				}
				dateHTML, err := h.Renderer.RenderToString("tumble_date.html", dateData)
				if err == nil {
					containerHTML += dateHTML
				}
				lastDate = p.FullDate
			}
		}
		containerHTML += p.HTML
	}

	// Hot Links (Side bar) - Only for HTML
	hotHTML := ""
	if dtype != "rss" && dtype != "xml" {
		// Perl: 12 to 6 days ago.
		topLinks, err := h.Store.GetTopIRCLinks(ctx, 12, 6, 5)
		if err == nil {
			for _, l := range topLinks {
				// Link content: <a href...>Title</a>
				content := fmt.Sprintf(`<a href="http://%s/irclink/?%d">%s</a>`, h.Config.BaseURL, l.ID, l.Title)
				if len(l.Title) > 15 && len(l.Title) > 15 {
					// Truncate logic from Perl
					// if ( $hot->{$_}->{'title'} =~ /^(http:\/\/.*)/ ) { if ( length( $1 ) > 15 ) { ... } }
					// Handled loosely here or strictly port logic.
				}

				// Render item
				// Using map for flexibility
				data := map[string]interface{}{
					"Content": template.HTML(content),
				}
				s, _ := h.Renderer.RenderToString("tumble_item_top5.html", data)
				hotHTML += s
			}
		}
	}

	// Navigation
	navP := ""
	navN := ""
	if iParam != "" {
		navP = fmt.Sprintf(`<a href="?i=%d"><img src="/img/prev.jpg" border="0" alt="" /></a>`, i+1)
		navN = fmt.Sprintf(` &nbsp;<a href="?i=%d"><img src="/img/next.jpg" border="0" alt="" /></a>`, i-1)
	} else {
		navP = `<a href="?i=2"><img src="/img/prev.jpg" border="0" alt="" /></a>`
	}
	if i == 1 {
		navN = "" // Perl: $nav->{'n'} = '' unless $self->{'arg'}->{'i'};
	}

	// View Data
	viewData := IndexPageData{
		PageTitle:    "",
		Container:    template.HTML(containerHTML),
		Hot:          template.HTML(hotHTML),
		NavP:         template.HTML(navP),
		NavN:         template.HTML(navN),
		BaseURL:      template.HTML(h.Config.BaseURL),
		GitCommit:    version.CommitHash,
		GitCommitURL: fmt.Sprintf("https://github.com/websages/tumble/commit/%s", version.CommitHash),
	}

	templateName := "index.html"
	contentType := "text/html; charset=UTF-8"
	if dtype == "rss" || dtype == "xml" {
		templateName = "index.xml"
		contentType = "text/xml; charset=UTF-8"
	}

	w.Header().Set("Content-Type", contentType)
	if err := h.Renderer.Render(w, templateName, viewData); err != nil {
		log.Printf("Error rendering template: %v", err)
	}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query().Get("search")

	if query == "" {
		// Perl behaviour: returns unless string?
		return
	}

	// Perform Search
	links, err := h.Store.SearchIRCLinks(ctx, query)
	if err != nil {
		log.Printf("Search error: %v", err)
		http.Error(w, "Search Error", http.StatusInternalServerError)
		return
	}

	containerHTML := ""
	if len(links) > 0 {
		for _, item := range links {
			d := h.Service.ProcessIRCLink(item)
			s, _ := h.Renderer.RenderToString("tumble_item_ircLink.html", d)
			containerHTML += s
		}
	} else {
		// No results template (tumble_item_text)
		msg := fmt.Sprintf(`
            <font color="#000">Your search-fu is weak.</font><br /><br />
            Your search for '%s' did not return any results.  Perhaps the following tips can help aid you on your quest:
            <ul>
                <li>Searches must be done using four or more characters.<br /><br />
                <li>MySQL fulltext-searching is the magic behind this.  Stop blaming scott.<br /><br />
                <li>Try not to be such a fucking idiot.
            </ul>`, query)

		data := map[string]interface{}{
			"Content": template.HTML(msg),
		}
		containerHTML, _ = h.Renderer.RenderToString("tumble_item_text.html", data)
	}

	// Hot links (Same logic as Index)
	hotHTML := ""
	topLinks, err := h.Store.GetTopIRCLinks(ctx, 12, 6, 5)
	if err == nil {
		for _, l := range topLinks {
			content := fmt.Sprintf(`<a href="http://%s/irclink/?%d">%s</a>`, h.Config.BaseURL, l.ID, l.Title)
			data := map[string]interface{}{"Content": template.HTML(content)}
			s, _ := h.Renderer.RenderToString("tumble_item_top5.html", data)
			hotHTML += s
		}
	}

	viewData := IndexPageData{
		PageTitle: fmt.Sprintf(" &gt; %s", query),
		Container: template.HTML(containerHTML),
		Hot:       template.HTML(hotHTML),
		BaseURL:   template.HTML(h.Config.BaseURL),
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	h.Renderer.Render(w, "index.html", viewData)
}

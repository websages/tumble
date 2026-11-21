package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"
)

// DiagnoseAttempt is one probe of a URL with a given method and user agent.
type DiagnoseAttempt struct {
	Method      string   `json:"method"`
	UserAgent   string   `json:"user_agent"`
	Status      int      `json:"status,omitempty"`
	ContentType string   `json:"content_type,omitempty"`
	FinalURL    string   `json:"final_url,omitempty"`
	Redirects   []string `json:"redirects,omitempty"`
	DurationMS  int64    `json:"duration_ms"`
	Error       string   `json:"error,omitempty"`
}

// DiagnoseResponse reports how a URL answers each probe, what was cached,
// and what a fresh preview fetch now returns.
type DiagnoseResponse struct {
	URL         string            `json:"url"`
	CachedEntry json.RawMessage   `json:"cached_entry,omitempty"`
	CachedAt    *time.Time        `json:"cached_at,omitempty"`
	Attempts    []DiagnoseAttempt `json:"attempts"`
	Preview     map[string]any    `json:"preview"`
}

// APIv1PreviewDiagnoseHandler handles GET /api/v1/preview/diagnose?url=...
// It probes the URL with every scrape user agent (HEAD and GET), then
// clears the cached preview and fetches a fresh one. Requires X-API-Key.
func (h *Handler) APIv1PreviewDiagnoseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}
	if !isAuthorizedAPIKey(r, h.Config.AdminSecret) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "Invalid or missing API key")
		return
	}

	target := r.URL.Query().Get("url")
	u, err := url.Parse(target)
	if target == "" || err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		writeAPIError(w, http.StatusBadRequest, "bad_request", "An absolute http(s) 'url' query parameter is required")
		return
	}

	resp := DiagnoseResponse{URL: target}
	if cached, err := h.Store.GetLinkPreview(r.Context(), target); err == nil && cached != nil {
		resp.CachedEntry = json.RawMessage(cached.Data)
		resp.CachedAt = &cached.UpdatedAt
	}

	for _, ua := range scrapeUserAgents {
		for _, method := range []string{http.MethodHead, http.MethodGet} {
			resp.Attempts = append(resp.Attempts, h.probeURL(target, method, ua))
		}
	}

	// Throw away the cached result and run the real preview pipeline.
	h.Store.DeleteLinkPreview(r.Context(), target)
	rec := httptest.NewRecorder()
	fresh := r.Clone(r.Context())
	fresh.URL = &url.URL{Path: "/api/v1/preview", RawQuery: url.Values{"url": {target}}.Encode()}
	h.OGPreviewHandler(rec, fresh)
	json.Unmarshal(rec.Body.Bytes(), &resp.Preview)

	writeJSON(w, http.StatusOK, resp)
}

// probeURL makes one request and records the status, redirects and timing
// without reading the body.
func (h *Handler) probeURL(target, method string, ua scrapeUserAgent) DiagnoseAttempt {
	attempt := DiagnoseAttempt{Method: method, UserAgent: ua.Name}
	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		attempt.Error = err.Error()
		return attempt
	}
	if ua.Value != "" {
		req.Header.Set("User-Agent", ua.Value)
	}

	client := &http.Client{
		Timeout: h.Config.RequestTimeout,
		CheckRedirect: func(next *http.Request, via []*http.Request) error {
			attempt.Redirects = append(attempt.Redirects, next.URL.String())
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	start := time.Now()
	res, err := client.Do(req)
	attempt.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		attempt.Error = err.Error()
		return attempt
	}
	res.Body.Close()

	attempt.Status = res.StatusCode
	attempt.ContentType = res.Header.Get("Content-Type")
	attempt.FinalURL = res.Request.URL.String()
	return attempt
}

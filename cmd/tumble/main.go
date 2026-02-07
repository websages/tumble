package main

import (
	"context"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"tumble/internal/assets"
	"tumble/internal/config"
	"tumble/internal/data"
	"tumble/internal/handler"
	"tumble/internal/service"
	"tumble/internal/templates"

	"golang.org/x/time/rate"
)

type responseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// trailingSlashMiddleware intercepts 301 redirects that add trailing slashes
// and converts them to 308 (Permanent Redirect) to preserve HTTP methods.
// This is safer for POST/PUT/DELETE requests which would otherwise be converted to GET.
func trailingSlashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap the response writer to intercept redirects
		wrapper := &redirectInterceptor{
			ResponseWriter: w,
			request:        r,
		}
		next.ServeHTTP(wrapper, r)
	})
}

// redirectInterceptor intercepts 301 redirects and converts them to 308
type redirectInterceptor struct {
	http.ResponseWriter
	request     *http.Request
	wroteHeader bool
}

func (ri *redirectInterceptor) WriteHeader(code int) {
	if ri.wroteHeader {
		return
	}
	ri.wroteHeader = true

	// Convert 301 to 308 for non-GET/HEAD requests to preserve HTTP method
	if code == http.StatusMovedPermanently && ri.request.Method != http.MethodGet && ri.request.Method != http.MethodHead {
		code = http.StatusPermanentRedirect // 308
	}
	ri.ResponseWriter.WriteHeader(code)
}

func (ri *redirectInterceptor) Write(b []byte) (int, error) {
	if !ri.wroteHeader {
		ri.WriteHeader(http.StatusOK)
	}
	return ri.ResponseWriter.Write(b)
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always set these headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Only set HSTS for non-localhost requests
		host := r.Host
		if host != "localhost" && !strings.HasPrefix(host, "localhost:") &&
			host != "127.0.0.1" && !strings.HasPrefix(host, "127.0.0.1:") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// ipRateLimiter manages per-IP rate limiters
type ipRateLimiter struct {
	limiters sync.Map
	rate     rate.Limit
	burst    int
}

func newIPRateLimiter(r rate.Limit, burst int) *ipRateLimiter {
	return &ipRateLimiter{
		rate:  r,
		burst: burst,
	}
}

func (i *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	limiter, exists := i.limiters.Load(ip)
	if !exists {
		limiter = rate.NewLimiter(i.rate, i.burst)
		i.limiters.Store(ip, limiter)
	}
	return limiter.(*rate.Limiter)
}

// extractIP gets the client IP, checking X-Forwarded-For for proxied requests
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by reverse proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP (original client)
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr (strip port)
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// Global rate limiters with different limits for different endpoint types
var (
	// General limiter: 60 requests/minute with burst of 10
	generalLimiter = newIPRateLimiter(rate.Limit(1), 10)
	// OG preview limiter: 30 requests/minute with burst of 100
	// High burst allows cold cache page loads (75+ links); rate limiting only applies to cache misses
	ogPreviewLimiter = newIPRateLimiter(rate.Limit(0.5), 100)
	// Search limiter: 20 requests/minute with burst of 3
	searchLimiter = newIPRateLimiter(rate.Limit(0.33), 3)
)

func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		path := r.URL.Path

		// Select appropriate limiter based on endpoint
		// Note: ogpreview is handled separately with cache-aware rate limiting
		var limiter *rate.Limiter
		switch {
		case strings.HasPrefix(path, "/ogpreview"):
			// Skip middleware rate limiting - handled by ogPreviewWithCacheRateLimit
			next.ServeHTTP(w, r)
			return
		case strings.HasPrefix(path, "/search"):
			limiter = searchLimiter.getLimiter(ip)
		default:
			limiter = generalLimiter.getLimiter(ip)
		}

		if !limiter.Allow() {
			slog.Warn("Rate limit exceeded", "ip", ip, "path", path)
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ogPreviewWithCacheRateLimit wraps the OG preview handler with cache-aware rate limiting.
// Cache hits bypass rate limiting entirely; only cache misses consume rate limit tokens.
func ogPreviewWithCacheRateLimit(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to serve from cache first - no rate limit consumed
		if h.TryServeCachedOGPreview(w, r) {
			return
		}

		// Cache miss - apply rate limiting before fetching
		ip := extractIP(r)
		if !ogPreviewLimiter.getLimiter(ip).Allow() {
			slog.Warn("Rate limit exceeded", "ip", ip, "path", r.URL.Path)
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		// Proceed with actual fetch
		h.OGPreviewHandler(w, r)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK} // Default status

		next.ServeHTTP(rw, r)

		slog.Info("Request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"bytes", rw.bytesWritten,
			"duration", time.Since(start),
			"remote_addr", r.RemoteAddr,
		)
	})
}

func main() {
	// Load Config
	// Load Config
	cfgPath := ""
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Error("Fatal: Could not load config", "error", err)
		os.Exit(1)
	}

	// Setup Logging
	var output io.Writer = os.Stdout
	if cfg.Logging.Output != "" && cfg.Logging.Output != "stdout" {
		if cfg.Logging.Output == "stderr" {
			output = os.Stderr
		} else {
			f, err := os.OpenFile(cfg.Logging.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				slog.Error("Failed to open log file", "path", cfg.Logging.Output, "error", err)
				os.Exit(1)
			}
			defer f.Close()
			output = f
		}
	}

	// Redirect standard log library to the same output
	// This captures logs from libraries or legacy code (like irclink.go)
	// preventing them from leaking to stdout/stderr if a file is configured.
	// We only do this if we aren't writing to stdout/stderr to match expected behavior
	// of "if log file is specified, do not output to stdout"
	if cfg.Mode == "development" || cfg.Mode == "dev" {
		// If we are in dev mode and have a specific file output, we force standard log to that file too
		// If output is already stdout/stderr, this is a no-op effectively
		log.SetOutput(output)
	} else {
		// In production/other modes, we also likely want to capture standard logs into our structured log stream
		// or at least to the same destination.
		log.SetOutput(output)
	}

	var level slog.Level
	// Default level based on Mode if not explicitly set
	if cfg.Logging.Level == "" {
		if cfg.Mode == "development" {
			level = slog.LevelDebug
		} else {
			level = slog.LevelInfo
		}
	} else {
		switch cfg.Logging.Level {
		case "debug", "verbose":
			level = slog.LevelDebug
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			level = slog.LevelInfo
		}
	}

	var logHandler slog.Handler
	if cfg.Mode == "production" {
		logHandler = slog.NewJSONHandler(output, &slog.HandlerOptions{Level: level})
	} else {
		logHandler = slog.NewTextHandler(output, &slog.HandlerOptions{Level: level})
	}

	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	// Init DB
	store, err := data.NewStore(cfg)
	if err != nil {
		slog.Error("Fatal: Could not connect to DB", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Auto-Migrate
	if err := store.Bootstrap(context.TODO()); err != nil {
		slog.Error("Fatal: Database bootstrap failed", "error", err)
		os.Exit(1)
	}

	// Init Service
	svc := service.NewContentService(cfg, store)

	// Init Renderer
	renderer, err := templates.NewRenderer(cfg)
	if err != nil {
		slog.Error("Fatal: Could not init renderer", "error", err)
		os.Exit(1)
	}

	// Init Handler
	h := handler.NewHandler(cfg, store, svc, renderer)

	// Router
	mux := http.NewServeMux()

	// Main Routes
	mux.HandleFunc("/", h.Index)
	mux.HandleFunc("/index.cgi", h.Index)
	mux.HandleFunc("/stats", h.Stats)
	mux.HandleFunc("/search", h.Search)
	mux.HandleFunc("/search.cgi", h.Search)
	mux.HandleFunc("/link/", h.IRCLinkHandler)    // Primary endpoint for links
	mux.HandleFunc("/irclink/", h.IRCLinkHandler) // Legacy endpoint (backwards compatibility)

	mux.HandleFunc("/ogpreview", ogPreviewWithCacheRateLimit(h))
	mux.HandleFunc("/ogpreview.cgi", ogPreviewWithCacheRateLimit(h))
	mux.HandleFunc("/api/caching/invalidate", h.InvalidateCacheHandler)
	mux.HandleFunc("/buttons/", h.ButtonHandler)           // Handle /buttons/ with ButtonHandler (landing + result)
	mux.HandleFunc("/buttons/button.cgi", h.ButtonHandler) // Legacy explicit path

	// v0 Routes (Aliased)
	mux.HandleFunc("/v0/", h.Index)
	mux.HandleFunc("/v0/index.cgi", h.Index)
	mux.HandleFunc("/v0/search.cgi", h.Search)
	mux.HandleFunc("/v0/link/", h.IRCLinkHandler)    // Primary v0 endpoint
	mux.HandleFunc("/v0/irclink/", h.IRCLinkHandler) // Legacy v0 endpoint
	mux.HandleFunc("/v0/ogpreview.cgi", ogPreviewWithCacheRateLimit(h))
	mux.HandleFunc("/v0/quote/", h.QuoteHandler)

	// Quote Handler (Legacy)
	mux.HandleFunc("/quote/", h.QuoteHandler)
	mux.HandleFunc("/quote/index.cgi", h.QuoteHandler)

	// SEO Routes
	mux.HandleFunc("/sitemap.xml", h.SitemapHandler)
	mux.HandleFunc("/robots.txt", h.RobotsHandler)

	// Static Assets
	// Serve from embedded FS
	// "/css/" -> internal/assets/css
	fileServer := http.FileServer(http.FS(assets.StaticFS))
	mux.Handle("/css/", fileServer)
	mux.Handle("/img/", fileServer)
	// mux.Handle("/buttons/", fileServer) // Removed in favor of ButtonHandler check
	mux.Handle("/favicon.ico", fileServer)
	// Legacy static files
	mux.Handle("/apple-touch-icon.png", fileServer)
	subFS, _ := fs.Sub(assets.StaticFS, "2202")
	mux.Handle("/2202/", http.StripPrefix("/2202/", http.FileServer(http.FS(subFS))))
	mux.Handle("/roast/", fileServer)

	// API Documentation
	mux.HandleFunc("/api/docs", h.DocsHandler)
	mux.HandleFunc("/api/openapi.json", h.OpenAPISpecHandler)

	// Start
	addr := ":" + cfg.Port
	if cfg.Port == "" {
		addr = ":8080"
	}
	slog.Info("Starting tumble server", "addr", addr)
	if err := http.ListenAndServe(addr, trailingSlashMiddleware(rateLimitMiddleware(securityHeadersMiddleware(loggingMiddleware(mux))))); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

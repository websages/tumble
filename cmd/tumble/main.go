package main

import (
	"context"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"tumble/internal/assets"
	"tumble/internal/config"
	"tumble/internal/data"
	"tumble/internal/handler"
	"tumble/internal/scheduler"
	"tumble/internal/service"
	"tumble/internal/templates"
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

func loggingMiddleware(mode string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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
				"remote_addr", clientIP(r, mode),
			)
		})
	}
}

func clientIP(r *http.Request, mode string) string {
	if mode == "development" || mode == "dev" {
		return hostFromAddr(r.RemoteAddr)
	}

	// In production, honor proxy headers from Caddy or other reverse proxies.
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		if ip := pickIPFromXFF(xff); ip != "" {
			return ip
		}
	}

	xri := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if xri != "" {
		return hostFromAddr(xri)
	}

	return hostFromAddr(r.RemoteAddr)
}

func hostFromAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return normalizeIP(strings.Trim(addr, "[]"))
	}
	return normalizeIP(host)
}

func pickIPFromXFF(xff string) string {
	parts := strings.Split(xff, ",")
	firstNonEmpty := ""
	for _, part := range parts {
		ip := normalizeIP(strings.TrimSpace(part))
		if ip == "" {
			continue
		}
		if firstNonEmpty == "" {
			firstNonEmpty = ip
		}
		parsed := net.ParseIP(ip)
		if parsed != nil && parsed.To4() != nil {
			return parsed.To4().String()
		}
	}
	return firstNonEmpty
}

func normalizeIP(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}
	if parsed.IsLoopback() && parsed.To4() == nil {
		return "127.0.0.1"
	}
	if v4 := parsed.To4(); v4 != nil {
		return v4.String()
	}
	return parsed.String()
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

	// Init Scheduler
	sched := scheduler.New(store)
	if err := sched.Start(context.Background()); err != nil {
		slog.Error("Failed to start scheduler", "error", err)
		os.Exit(1)
	}
	defer sched.Stop()

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
	mux.HandleFunc("/stats.json", h.StatsJSON)
	mux.HandleFunc("/search", h.Search)
	mux.HandleFunc("/search.cgi", h.Search)
	mux.HandleFunc("/link/", h.IRCLinkHandler)    // Primary endpoint for links
	mux.HandleFunc("/irclink/", h.IRCLinkHandler) // Legacy endpoint (backwards compatibility)

	mux.HandleFunc("/ogpreview", h.OGPreviewHandler)
	mux.HandleFunc("/ogpreview.cgi", h.OGPreviewHandler)
	mux.HandleFunc("/api/caching/invalidate", h.InvalidateCacheHandler)
	mux.HandleFunc("/api/kitten/fetch", h.FetchKittenHandler)
	mux.HandleFunc("/buttons/", h.ButtonHandler)           // Handle /buttons/ with ButtonHandler (landing + result)
	mux.HandleFunc("/buttons/button.cgi", h.ButtonHandler) // Legacy explicit path

	// v0 Routes (Aliased)
	mux.HandleFunc("/v0/", h.Index)
	mux.HandleFunc("/v0/index.cgi", h.Index)
	mux.HandleFunc("/v0/search.cgi", h.Search)
	mux.HandleFunc("/v0/link/", h.IRCLinkHandler)    // Primary v0 endpoint
	mux.HandleFunc("/v0/irclink/", h.IRCLinkHandler) // Legacy v0 endpoint
	mux.HandleFunc("/v0/ogpreview.cgi", h.OGPreviewHandler)
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

	// API v1 Routes
	mux.HandleFunc("/api/v1/links", h.APIv1LinksHandler)
	mux.HandleFunc("/api/v1/links/", h.APIv1LinksHandler)
	mux.HandleFunc("/api/v1/quotes", h.APIv1QuotesHandler)
	mux.HandleFunc("/api/v1/quotes/", h.APIv1QuotesHandler)
	mux.HandleFunc("/api/v1/stats", h.APIv1StatsHandler)
	mux.HandleFunc("/api/v1/users/", h.APIv1UsersHandler)
	mux.HandleFunc("/api/v1/search", h.APIv1SearchHandler)
	mux.HandleFunc("/api/v1/cache", h.APIv1CacheHandler)
	mux.HandleFunc("/api/v1/kittens/", h.APIv1KittensDailyHandler)

	// Public redirect shortlink
	mux.HandleFunc("/r/", h.APIv1RedirectHandler)

	// Start
	addr := ":" + cfg.Port
	if cfg.Port == "" {
		addr = ":8080"
	}
	slog.Info("Starting tumble server", "addr", addr)
	if err := http.ListenAndServe(addr, trailingSlashMiddleware(securityHeadersMiddleware(loggingMiddleware(cfg.Mode)(mux)))); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

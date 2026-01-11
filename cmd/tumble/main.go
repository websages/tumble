package main

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"time"
	"tumble/internal/assets"
	"tumble/internal/config"
	"tumble/internal/data"
	"tumble/internal/handler"
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
	cfgPath := "conf/config.yaml" // Default or flag
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Warn("Warning: Could not load config, trying fallback", "path", cfgPath, "error", err)
		// Proceed with defaults or fail? Perl requires config.yaml in htdocs usually.
		// We'll assume we need one.
		// Try htdocs/config.yaml
		cfg, err = config.Load("htdocs/config.yaml")
		if err != nil {
			slog.Error("Fatal: Could not load config", "error", err)
			os.Exit(1)
		}
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

	var level slog.Level
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

	logger := slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	// Init DB
	store, err := data.NewStore(cfg.Driver, cfg.DSN())
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
	svc := service.NewContentService(cfg)

	// Init Renderer
	renderer, err := templates.NewRenderer()
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
	mux.HandleFunc("/search.cgi", h.Search)
	mux.HandleFunc("/irclink/", h.IRCLinkHandler) // Handles /irclink/?id and posts
	mux.HandleFunc("/ogpreview.cgi", h.OGPreviewHandler)
	mux.HandleFunc("/buttons/button.cgi", h.ButtonHandler)

	// v0 Routes (Aliased)
	mux.HandleFunc("/v0/", h.Index)
	mux.HandleFunc("/v0/index.cgi", h.Index)
	mux.HandleFunc("/v0/search.cgi", h.Search)
	mux.HandleFunc("/v0/irclink/", h.IRCLinkHandler)
	mux.HandleFunc("/v0/ogpreview.cgi", h.OGPreviewHandler)
	mux.HandleFunc("/v0/quote/", h.QuoteHandler)

	// Quote Handler (Legacy)
	mux.HandleFunc("/quote/", h.QuoteHandler)
	mux.HandleFunc("/quote/index.cgi", h.QuoteHandler)

	// Static Assets
	// Serve from embedded FS
	// "/css/" -> internal/assets/css
	fileServer := http.FileServer(http.FS(assets.StaticFS))
	mux.Handle("/css/", fileServer)
	mux.Handle("/img/", fileServer)
	mux.Handle("/buttons/", fileServer)
	mux.Handle("/favicon.ico", fileServer)
	// Legacy static files
	mux.Handle("/robots.txt", fileServer)
	mux.Handle("/apple-touch-icon.png", fileServer)
	subFS, _ := fs.Sub(assets.StaticFS, "2202")
	mux.Handle("/2202/", http.StripPrefix("/2202/", http.FileServer(http.FS(subFS))))
	mux.Handle("/roast/", fileServer)

	// API Documentation
	mux.HandleFunc("/api/docs", h.DocsHandler)
	mux.HandleFunc("/api/openapi.json", h.OpenAPISpecHandler)

	// Start
	addr := ":8080" // Default or from config? Perl was CGI so port wasn't in config.
	slog.Info("Starting tumble server", "addr", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

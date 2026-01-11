package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"tumble/internal/assets"
	"tumble/internal/config"
	"tumble/internal/data"
	"tumble/internal/handler"
	"tumble/internal/service"
	"tumble/internal/templates"
)

func main() {
	// Load Config
	cfgPath := "conf/config.yaml" // Default or flag
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Printf("Warning: Could not load config from %s: %v", cfgPath, err)
		// Proceed with defaults or fail? Perl requires config.yaml in htdocs usually.
		// We'll assume we need one.
		// Try htdocs/config.yaml
		cfg, err = config.Load("htdocs/config.yaml")
		if err != nil {
			log.Fatalf("Fatal: Could not load config: %v", err)
		}
	}

	// Init DB
	store, err := data.NewStore(cfg.Driver, cfg.DSN())
	if err != nil {
		log.Fatalf("Fatal: Could not connect to DB: %v", err)
	}
	defer store.Close()

	// Auto-Migrate
	if err := store.Bootstrap(context.TODO()); err != nil {
		log.Fatalf("Fatal: Database bootstrap failed: %v", err)
	}

	// Init Service
	svc := service.NewContentService(cfg)

	// Init Renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		log.Fatalf("Fatal: Could not init renderer: %v", err)
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

	// API Documentation
	mux.HandleFunc("/api/docs", h.DocsHandler)
	mux.HandleFunc("/api/openapi.json", h.OpenAPISpecHandler)

	// Start
	addr := ":8080" // Default or from config? Perl was CGI so port wasn't in config.
	log.Printf("Starting tumble server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

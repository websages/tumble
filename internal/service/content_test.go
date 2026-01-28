package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"tumble/internal/config"
	"tumble/internal/data"
)

func TestProcessIRCLink_Flickr(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://tumble.test"}
	svc := NewContentService(cfg, nil)

	tests := []struct {
		name     string
		item     data.IRCLink
		wantURL  string
		wantType string // "flickr" or "default" (internal link)
	}{
		{
			name: "Flickr Static URL",
			item: data.IRCLink{
				ID:          1,
				Title:       "A Photo",
				URL:         "http://farm3.staticflickr.com/2362/2362225867_0a3b0b7e05.jpg",
				ContentType: "image/jpeg",
				User:        "photog",
				Timestamp:   time.Now(),
			},
			wantURL:  "https://www.flickr.com/photo.gne?id=2362225867",
			wantType: "flickr",
		},
		{
			name: "Normal Image",
			item: data.IRCLink{
				ID:          2,
				Title:       "Just an Image",
				URL:         "http://example.com/image.jpg",
				ContentType: "image/jpeg",
				User:        "user",
				Timestamp:   time.Now(),
			},
			wantURL:  "http://tumble.test/irclink/?2",
			wantType: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.ProcessIRCLink(tt.item)
			html := string(got.Content)

			if tt.wantType == "flickr" {
				if !strings.Contains(html, tt.wantURL) {
					t.Errorf("ProcessIRCLink() html = %v, want to contain %v", html, tt.wantURL)
				}

				// Verify it's an anchor tag
				if !strings.HasPrefix(html, "<a href=") {
					t.Errorf("ProcessIRCLink() html should start with anchor tag, got %v", html)
				}
				// Verify target blank
				if !strings.Contains(html, `target="_blank"`) {
					t.Errorf("ProcessIRCLink() html should contain target=_blank, got %v", html)
				}
			} else {
				if !strings.Contains(html, tt.wantURL) {
					t.Errorf("ProcessIRCLink() html = %v, want to contain %v", html, tt.wantURL)
				}
				// Verify target blank for default links too
				if !strings.Contains(html, `target="_blank"`) {
					t.Errorf("ProcessIRCLink() html should contain target=_blank, got %v", html)
				}
			}
		})
	}
}

func TestProcessImage_Flickr(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://tumble.test"}
	svc := NewContentService(cfg, nil)

	tests := []struct {
		name     string
		item     data.Image
		wantURL  string
		wantType string // "flickr" or "default"
	}{
		{
			name: "Flickr Static URL",
			item: data.Image{
				ID:        1,
				Title:     "A Photo",
				URL:       "http://farm3.staticflickr.com/2362/2362225867_0a3b0b7e05.jpg",
				Timestamp: time.Now(),
			},
			wantURL:  "https://www.flickr.com/photo.gne?id=2362225867",
			wantType: "flickr",
		},
		{
			name: "Normal Image",
			item: data.Image{
				ID:        2,
				Title:     "Just an Image",
				URL:       "http://example.com/image.jpg",
				Timestamp: time.Now(),
			},
			wantURL:  "http://example.com/image.jpg",
			wantType: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.ProcessImage(tt.item)
			html := string(got.Content)

			if tt.wantType == "flickr" {
				if !strings.Contains(html, tt.wantURL) {
					t.Errorf("ProcessImage() html = %v, want to contain %v", html, tt.wantURL)
				}
				// Verify it's an anchor tag
				if !strings.HasPrefix(html, "<a href=") {
					t.Errorf("ProcessImage() html should start with anchor tag, got %v", html)
				}
				// Verify target blank
				if !strings.Contains(html, `target="_blank"`) {
					t.Errorf("ProcessImage() html should contain target=_blank, got %v", html)
				}
			} else {
				if !strings.Contains(html, tt.wantURL) {
					t.Errorf("ProcessImage() html = %v, want to contain %v", html, tt.wantURL)
				}
				// Verify default is just img tag
				if !strings.HasPrefix(html, "<img src=") {
					t.Errorf("ProcessImage() html should start with img tag, got %v", html)
				}
			}
		})
	}
}

func TestProcessIRCLink_Imgur(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://tumble.test"}
	svc := NewContentService(cfg, nil)

	tests := []struct {
		name               string
		item               data.IRCLink
		wantType           string // "single", "gallery"
		wantIRCLinkHandler bool   // Should route through /irclink/?
		wantImgurCDN       bool   // Should use i.imgur.com
		wantErrorHandler   bool   // Should have onerror handler
		wantSuppressOG     bool
		wantGalleryCard    bool
	}{
		{
			name: "Single Imgur Image - Standard URL",
			item: data.IRCLink{
				ID:          42,
				Title:       "Cool Picture",
				URL:         "https://imgur.com/abc123",
				ContentType: "text/html",
				User:        "testuser",
				Timestamp:   time.Now(),
			},
			wantType:           "single",
			wantIRCLinkHandler: true,
			wantImgurCDN:       true,
			wantErrorHandler:   false, // Extensionless URLs default to .mp4 video (no error handler)
			wantSuppressOG:     true,
		},
		{
			name: "Single Imgur Image - With Extension",
			item: data.IRCLink{
				ID:          43,
				Title:       "Another Picture",
				URL:         "https://imgur.com/xyz789.jpg",
				ContentType: "image/jpeg",
				User:        "testuser",
				Timestamp:   time.Now(),
			},
			wantType:           "single",
			wantIRCLinkHandler: true,
			wantImgurCDN:       true,
			wantErrorHandler:   true,
			wantSuppressOG:     true,
		},
		{
			name: "Single Imgur Image - Direct i.imgur.com",
			item: data.IRCLink{
				ID:          44,
				Title:       "Direct CDN",
				URL:         "https://i.imgur.com/def456.png",
				ContentType: "image/png",
				User:        "testuser",
				Timestamp:   time.Now(),
			},
			wantType:           "single",
			wantIRCLinkHandler: true,
			wantImgurCDN:       true,
			wantErrorHandler:   true,
			wantSuppressOG:     true,
		},
		{
			name: "Imgur Gallery - /gallery/ URL (no preview)",
			item: data.IRCLink{
				ID:          45,
				Title:       "Gallery Title",
				URL:         "https://imgur.com/gallery/abcGallery",
				ContentType: "text/html",
				User:        "testuser",
				Timestamp:   time.Now(),
			},
			wantType:           "gallery",
			wantIRCLinkHandler: true,
			wantSuppressOG:     true,
			wantGalleryCard:    false, // No LinkPreview, so should show 404 error
		},
		{
			name: "Imgur Gallery - /a/ URL (no preview)",
			item: data.IRCLink{
				ID:          46,
				Title:       "Album Title",
				URL:         "https://imgur.com/a/xyzAlbum",
				ContentType: "text/html",
				User:        "testuser",
				Timestamp:   time.Now(),
			},
			wantType:           "gallery",
			wantIRCLinkHandler: true,
			wantSuppressOG:     true,
			wantGalleryCard:    false, // No LinkPreview, so should show 404 error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.ProcessIRCLink(tt.item)
			html := string(got.Content)

			// Verify SuppressOG
			if got.SuppressOG != tt.wantSuppressOG {
				t.Errorf("ProcessIRCLink() SuppressOG = %v, want %v", got.SuppressOG, tt.wantSuppressOG)
			}

			// CRITICAL: Verify IRC link handler routing (click tracking)
			if tt.wantIRCLinkHandler {
				expectedIRCLink := fmt.Sprintf("%s/irclink/?%d", cfg.BaseURL, tt.item.ID)
				if !strings.Contains(html, expectedIRCLink) {
					t.Errorf("ProcessIRCLink() html should contain IRC link handler %v, got %v", expectedIRCLink, html)
				}

				// CRITICAL: Ensure we NEVER link directly to imgur.com (bypassing click tracking)
				// Gallery cards should not have direct imgur.com links
				if strings.Contains(html, `href="https://imgur.com`) || strings.Contains(html, `href="http://imgur.com`) {
					t.Errorf("ProcessIRCLink() html should NOT contain direct imgur.com links, got %v", html)
				}
			}

			// Verify Imgur CDN usage for single images
			if tt.wantImgurCDN {
				if !strings.Contains(html, "i.imgur.com") {
					t.Errorf("ProcessIRCLink() html should contain i.imgur.com CDN URL")
				}
			}

			// Verify error handler for single images
			if tt.wantErrorHandler {
				if !strings.Contains(html, "onerror=") {
					t.Errorf("ProcessIRCLink() html should contain onerror handler")
				}
				// Verify it uses the standard error pattern
				if !strings.Contains(html, "http-error-badge") {
					t.Errorf("ProcessIRCLink() html should use http-error-badge class for errors")
				}
				if !strings.Contains(html, "missing-link") {
					t.Errorf("ProcessIRCLink() html should use missing-link class for errors")
				}
			}

			// Verify gallery card structure (only if we expect a card)
			if tt.wantGalleryCard {
				if !strings.Contains(html, "imgur-gallery-card") {
					t.Errorf("ProcessIRCLink() html should contain imgur-gallery-card class")
				}
				if !strings.Contains(html, "Imgur Gallery") {
					t.Errorf("ProcessIRCLink() html should contain 'Imgur Gallery' text")
				}
				// Verify gallery has preview image attempt
				if !strings.Contains(html, "<img src=") {
					t.Errorf("ProcessIRCLink() gallery should attempt to show preview image")
				}
			}

			// Verify galleries WITHOUT previews show error badge (not gallery card)
			if tt.wantType == "gallery" && !tt.wantGalleryCard {
				if !strings.Contains(html, "http-error-badge") {
					t.Errorf("ProcessIRCLink() gallery without preview should show http-error-badge, got: %v", html)
				}
				if !strings.Contains(html, "missing-link") {
					t.Errorf("ProcessIRCLink() gallery without preview should show missing-link class")
				}
				// Should NOT contain gallery card elements
				if strings.Contains(html, "imgur-gallery-card") {
					t.Errorf("ProcessIRCLink() gallery without preview should NOT show gallery card")
				}
			}

			// Verify target="_blank" for all Imgur links
			if !strings.Contains(html, `target="_blank"`) {
				t.Errorf("ProcessIRCLink() html should open in new tab with target=_blank")
			}
		})
	}
}

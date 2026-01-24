package service

import (
	"strings"
	"testing"
	"time"

	"tumble/internal/config"
	"tumble/internal/data"
)

func TestProcessIRCLink_Flickr(t *testing.T) {
	cfg := &config.Config{BaseURL: "tumble.test"}
	svc := NewContentService(cfg)

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
	cfg := &config.Config{BaseURL: "tumble.test"}
	svc := NewContentService(cfg)

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

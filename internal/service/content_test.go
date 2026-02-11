package service

import (
	"testing"
	"time"

	"tumble/internal/config"
	"tumble/internal/data"
)

func TestProcessIRCLink_Flickr(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://tumble.test"}
	svc := NewContentService(cfg, nil)

	tests := []struct {
		name          string
		item          data.IRCLink
		wantEmbedType EmbedType
		wantPhotoPage string
		wantMediaURL  string
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
			wantEmbedType: EmbedTypeFlickr,
			wantPhotoPage: "https://www.flickr.com/photo.gne?id=2362225867",
			wantMediaURL:  "http://farm3.staticflickr.com/2362/2362225867_0a3b0b7e05.jpg",
		},
		{
			name: "Normal Image (direct link, not Flickr)",
			item: data.IRCLink{
				ID:          2,
				Title:       "Just an Image",
				URL:         "http://example.com/image.jpg",
				ContentType: "image/jpeg",
				User:        "user",
				Timestamp:   time.Now(),
			},
			wantEmbedType: EmbedTypeImage,
			wantMediaURL:  "http://example.com/image.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.ProcessIRCLink(tt.item)

			if got.EmbedType != tt.wantEmbedType {
				t.Errorf("ProcessIRCLink() EmbedType = %v, want %v", got.EmbedType, tt.wantEmbedType)
			}

			if tt.wantPhotoPage != "" && got.PhotoPageURL != tt.wantPhotoPage {
				t.Errorf("ProcessIRCLink() PhotoPageURL = %v, want %v", got.PhotoPageURL, tt.wantPhotoPage)
			}

			if tt.wantMediaURL != "" && got.MediaURL != tt.wantMediaURL {
				t.Errorf("ProcessIRCLink() MediaURL = %v, want %v", got.MediaURL, tt.wantMediaURL)
			}
		})
	}
}

func TestProcessImage_Flickr(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://tumble.test"}
	svc := NewContentService(cfg, nil)

	tests := []struct {
		name          string
		item          data.Image
		wantEmbedType EmbedType
		wantPhotoPage string
		wantMediaURL  string
	}{
		{
			name: "Flickr Static URL",
			item: data.Image{
				ID:        1,
				Title:     "A Photo",
				URL:       "http://farm3.staticflickr.com/2362/2362225867_0a3b0b7e05.jpg",
				Timestamp: time.Now(),
			},
			wantEmbedType: EmbedTypeFlickr,
			wantPhotoPage: "https://www.flickr.com/photo.gne?id=2362225867",
			wantMediaURL:  "http://farm3.staticflickr.com/2362/2362225867_0a3b0b7e05.jpg",
		},
		{
			name: "Normal Image",
			item: data.Image{
				ID:        2,
				Title:     "Just an Image",
				URL:       "http://example.com/image.jpg",
				Timestamp: time.Now(),
			},
			wantEmbedType: EmbedTypeImage,
			wantMediaURL:  "http://example.com/image.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.ProcessImage(tt.item)

			if got.EmbedType != tt.wantEmbedType {
				t.Errorf("ProcessImage() EmbedType = %v, want %v", got.EmbedType, tt.wantEmbedType)
			}

			if tt.wantPhotoPage != "" && got.PhotoPageURL != tt.wantPhotoPage {
				t.Errorf("ProcessImage() PhotoPageURL = %v, want %v", got.PhotoPageURL, tt.wantPhotoPage)
			}

			if got.MediaURL != tt.wantMediaURL {
				t.Errorf("ProcessImage() MediaURL = %v, want %v", got.MediaURL, tt.wantMediaURL)
			}
		})
	}
}

func TestProcessIRCLink_Imgur(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://tumble.test"}
	svc := NewContentService(cfg, nil)

	tests := []struct {
		name           string
		item           data.IRCLink
		wantEmbedType  EmbedType
		wantMediaURL   string
		wantIsAnimated bool
		wantSuppressOG bool
		wantIsBroken   bool
	}{
		{
			name: "Single Imgur Image - Standard URL (defaults to mp4)",
			item: data.IRCLink{
				ID:          42,
				Title:       "Cool Picture",
				URL:         "https://imgur.com/abc1234",
				ContentType: "text/html",
				User:        "testuser",
				Timestamp:   time.Now(),
			},
			wantEmbedType:  EmbedTypeImgurSingle,
			wantMediaURL:   "https://i.imgur.com/abc1234.mp4",
			wantIsAnimated: true,
			wantSuppressOG: true,
		},
		{
			name: "Single Imgur Image - With JPG Extension",
			item: data.IRCLink{
				ID:          43,
				Title:       "Another Picture",
				URL:         "https://imgur.com/xyz78901.jpg",
				ContentType: "image/jpeg",
				User:        "testuser",
				Timestamp:   time.Now(),
			},
			wantEmbedType:  EmbedTypeImgurSingle,
			wantMediaURL:   "https://i.imgur.com/xyz78901.jpg",
			wantIsAnimated: false,
			wantSuppressOG: true,
		},
		{
			name: "Imgur Gallery - /gallery/ URL (no store, so broken)",
			item: data.IRCLink{
				ID:          45,
				Title:       "Gallery Title",
				URL:         "https://imgur.com/gallery/abcGallery",
				ContentType: "text/html",
				User:        "testuser",
				Timestamp:   time.Now(),
			},
			wantEmbedType:  EmbedTypeImgurGallery,
			wantSuppressOG: true,
			wantIsBroken:   true, // No store, so no thumbnail
		},
		{
			name: "Imgur Album - /a/ URL (no store, so broken)",
			item: data.IRCLink{
				ID:          46,
				Title:       "Album Title",
				URL:         "https://imgur.com/a/xyzAlbum",
				ContentType: "text/html",
				User:        "testuser",
				Timestamp:   time.Now(),
			},
			wantEmbedType:  EmbedTypeImgurGallery,
			wantSuppressOG: true,
			wantIsBroken:   true, // No store, so no thumbnail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.ProcessIRCLink(tt.item)

			if got.EmbedType != tt.wantEmbedType {
				t.Errorf("ProcessIRCLink() EmbedType = %v, want %v", got.EmbedType, tt.wantEmbedType)
			}

			if got.SuppressOG != tt.wantSuppressOG {
				t.Errorf("ProcessIRCLink() SuppressOG = %v, want %v", got.SuppressOG, tt.wantSuppressOG)
			}

			if tt.wantMediaURL != "" && got.MediaURL != tt.wantMediaURL {
				t.Errorf("ProcessIRCLink() MediaURL = %v, want %v", got.MediaURL, tt.wantMediaURL)
			}

			if got.IsAnimated != tt.wantIsAnimated {
				t.Errorf("ProcessIRCLink() IsAnimated = %v, want %v", got.IsAnimated, tt.wantIsAnimated)
			}

			if got.IsBroken != tt.wantIsBroken {
				t.Errorf("ProcessIRCLink() IsBroken = %v, want %v", got.IsBroken, tt.wantIsBroken)
			}

			// Verify BaseURL is always set correctly
			if got.BaseURL != cfg.BaseURL {
				t.Errorf("ProcessIRCLink() BaseURL = %v, want %v", got.BaseURL, cfg.BaseURL)
			}
		})
	}
}

func TestProcessIRCLink_Twitter(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://tumble.test"}
	svc := NewContentService(cfg, nil)

	tests := []struct {
		name          string
		item          data.IRCLink
		wantEmbedType EmbedType
		wantEmbedURL  string
	}{
		{
			name: "Twitter.com URL",
			item: data.IRCLink{
				ID:          1,
				Title:       "A Tweet",
				URL:         "https://twitter.com/user/status/1234567890",
				ContentType: "text/html",
				User:        "poster",
				Timestamp:   time.Now(),
			},
			wantEmbedType: EmbedTypeTwitter,
			wantEmbedURL:  "https://twitter.com/user/status/1234567890",
		},
		{
			name: "X.com URL (should convert to twitter.com for embed)",
			item: data.IRCLink{
				ID:          2,
				Title:       "A Tweet",
				URL:         "https://x.com/user/status/1234567890",
				ContentType: "text/html",
				User:        "poster",
				Timestamp:   time.Now(),
			},
			wantEmbedType: EmbedTypeTwitter,
			wantEmbedURL:  "https://twitter.com/user/status/1234567890",
		},
		{
			name: "Non-tweet Twitter URL",
			item: data.IRCLink{
				ID:          3,
				Title:       "Twitter Profile",
				URL:         "https://twitter.com/someuser",
				ContentType: "text/html",
				User:        "poster",
				Timestamp:   time.Now(),
			},
			wantEmbedType: EmbedTypeGeneric, // Not a tweet, so generic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.ProcessIRCLink(tt.item)

			if got.EmbedType != tt.wantEmbedType {
				t.Errorf("ProcessIRCLink() EmbedType = %v, want %v", got.EmbedType, tt.wantEmbedType)
			}

			if tt.wantEmbedURL != "" && got.EmbedURL != tt.wantEmbedURL {
				t.Errorf("ProcessIRCLink() EmbedURL = %v, want %v", got.EmbedURL, tt.wantEmbedURL)
			}
		})
	}
}

func TestProcessQuote(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://tumble.test"}
	svc := NewContentService(cfg, nil)

	item := data.Quote{
		ID:        1,
		Author:    "Someone Famous",
		Quote:     "This is a great quote",
		Timestamp: time.Now(),
	}

	got := svc.ProcessQuote(item)

	if got.EmbedType != EmbedTypeQuote {
		t.Errorf("ProcessQuote() EmbedType = %v, want %v", got.EmbedType, EmbedTypeQuote)
	}

	if got.Quote != item.Quote {
		t.Errorf("ProcessQuote() Quote = %v, want %v", got.Quote, item.Quote)
	}

	if got.Author != item.Author {
		t.Errorf("ProcessQuote() Author = %v, want %v", got.Author, item.Author)
	}

	if got.Description != item.Quote {
		t.Errorf("ProcessQuote() Description = %v, want %v", got.Description, item.Quote)
	}
}

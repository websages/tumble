package handler

import "testing"

func TestExtractMapsTitle(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "place with plus-encoded name",
			url:  "https://www.google.com/maps/place/Eiffel+Tower/@48.8583701,2.2944813,17z",
			want: "Eiffel Tower",
		},
		{
			name: "place with percent-encoded name",
			url:  "https://www.google.com/maps/place/Caf%C3%A9+Central/@48.2082,16.3738,17z",
			want: "Café Central",
		},
		{
			name: "place with comma in name",
			url:  "https://www.google.com/maps/place/Portland%2C+OR/@45.5,-122.6,12z",
			want: "Portland, OR",
		},
		{
			name: "search query",
			url:  "https://www.google.com/maps/search/pizza+near+me/@40.7,-74,12z",
			want: "pizza near me",
		},
		{
			name: "directions",
			url:  "https://www.google.com/maps/dir/New+York/Boston",
			want: "New York → Boston",
		},
		{
			name: "directions with coordinates",
			url:  "https://www.google.com/maps/dir/Paris/London/@49.5,-0.5,7z",
			want: "Paris → London",
		},
		{
			name: "short URL returns empty",
			url:  "https://maps.app.goo.gl/abc123",
			want: "",
		},
		{
			name: "bare maps URL returns empty",
			url:  "https://www.google.com/maps/@40.7,-74,12z",
			want: "",
		},
		{
			name: "maps.google.com place URL",
			url:  "https://maps.google.com/maps/place/Central+Park/@40.7828,-73.9653,15z",
			want: "Central Park",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMapsTitle(tt.url)
			if got != tt.want {
				t.Errorf("extractMapsTitle(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestDecodeMapsName(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
		want    string
	}{
		{"plus to space", "Eiffel+Tower", "Eiffel Tower"},
		{"percent encoded", "Caf%C3%A9+Central", "Café Central"},
		{"no encoding needed", "Portland", "Portland"},
		{"multiple pluses", "New+York+City", "New York City"},
		{"percent comma", "Portland%2C+OR", "Portland, OR"},
		{"leading trailing spaces", "+trimmed+", "trimmed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeMapsName(tt.encoded)
			if got != tt.want {
				t.Errorf("decodeMapsName(%q) = %q, want %q", tt.encoded, got, tt.want)
			}
		})
	}
}

func TestIsGoogleMapsShortURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://maps.app.goo.gl/abc123", true},
		{"https://goo.gl/maps/xyz789", true},
		{"https://www.google.com/maps/place/Eiffel+Tower", false},
		{"https://maps.google.com/maps/place/Foo", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := isGoogleMapsShortURL(tt.url)
			if got != tt.want {
				t.Errorf("isGoogleMapsShortURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

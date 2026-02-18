package data

import "testing"

func TestClientFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		filter   ClientFilter
		expected bool
	}{
		{
			name:     "all nil fields",
			filter:   ClientFilter{},
			expected: true,
		},
		{
			name:     "only ClientType set",
			filter:   ClientFilter{ClientType: strPtr("irc")},
			expected: false,
		},
		{
			name:     "only ClientNetwork set",
			filter:   ClientFilter{ClientNetwork: strPtr("libera")},
			expected: false,
		},
		{
			name:     "only ClientChannel set",
			filter:   ClientFilter{ClientChannel: strPtr("#general")},
			expected: false,
		},
		{
			name:     "all fields set",
			filter:   ClientFilter{ClientType: strPtr("irc"), ClientNetwork: strPtr("libera"), ClientChannel: strPtr("#general")},
			expected: false,
		},
		{
			name:     "two fields set",
			filter:   ClientFilter{ClientType: strPtr("slack"), ClientNetwork: strPtr("workspace1")},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.IsEmpty()
			if got != tt.expected {
				t.Errorf("ClientFilter.IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClientFilter_Validate(t *testing.T) {
	tests := []struct {
		name      string
		filter    ClientFilter
		wantError bool
	}{
		{
			name:   "empty filter is valid",
			filter: ClientFilter{},
		},
		{
			name:   "type only is valid",
			filter: ClientFilter{ClientType: strPtr("irc")},
		},
		{
			name:   "type and network is valid",
			filter: ClientFilter{ClientType: strPtr("irc"), ClientNetwork: strPtr("libera")},
		},
		{
			name:   "all three is valid",
			filter: ClientFilter{ClientType: strPtr("irc"), ClientNetwork: strPtr("libera"), ClientChannel: strPtr("#general")},
		},
		{
			name:      "network without type is invalid",
			filter:    ClientFilter{ClientNetwork: strPtr("libera")},
			wantError: true,
		},
		{
			name:      "channel without type is invalid",
			filter:    ClientFilter{ClientChannel: strPtr("#general")},
			wantError: true,
		},
		{
			name:      "channel without network is invalid",
			filter:    ClientFilter{ClientType: strPtr("irc"), ClientChannel: strPtr("#general")},
			wantError: true,
		},
		{
			name:      "channel and network without type is invalid",
			filter:    ClientFilter{ClientNetwork: strPtr("libera"), ClientChannel: strPtr("#general")},
			wantError: true,
		},
		{
			name:      "invalid client_type is rejected",
			filter:    ClientFilter{ClientType: strPtr("foobar")},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

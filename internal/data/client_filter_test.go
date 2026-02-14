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

func strPtr(s string) *string {
	return &s
}

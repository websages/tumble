package data

import "testing"

func TestSourceFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		filter   SourceFilter
		expected bool
	}{
		{
			name:     "all nil fields",
			filter:   SourceFilter{},
			expected: true,
		},
		{
			name:     "only SourceType set",
			filter:   SourceFilter{SourceType: strPtr("irc")},
			expected: false,
		},
		{
			name:     "only SourceNetwork set",
			filter:   SourceFilter{SourceNetwork: strPtr("libera")},
			expected: false,
		},
		{
			name:     "only SourceChannel set",
			filter:   SourceFilter{SourceChannel: strPtr("#general")},
			expected: false,
		},
		{
			name:     "all fields set",
			filter:   SourceFilter{SourceType: strPtr("irc"), SourceNetwork: strPtr("libera"), SourceChannel: strPtr("#general")},
			expected: false,
		},
		{
			name:     "two fields set",
			filter:   SourceFilter{SourceType: strPtr("slack"), SourceNetwork: strPtr("workspace1")},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.IsEmpty()
			if got != tt.expected {
				t.Errorf("SourceFilter.IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "7 days",
			input:    "7d",
			expected: 7 * 24 * time.Hour,
		},
		{
			name:     "1 day",
			input:    "1d",
			expected: 24 * time.Hour,
		},
		{
			name:     "2 weeks",
			input:    "2w",
			expected: 14 * 24 * time.Hour,
		},
		{
			name:     "3 months (approximated as 90 days)",
			input:    "3m",
			expected: 90 * 24 * time.Hour,
		},
		{
			name:     "1 year (approximated as 365 days)",
			input:    "1y",
			expected: 365 * 24 * time.Hour,
		},
		{
			name:     "30 days",
			input:    "30d",
			expected: 30 * 24 * time.Hour,
		},
		{
			name:    "invalid format - no number",
			input:   "d",
			wantErr: true,
		},
		{
			name:    "invalid format - no unit",
			input:   "7",
			wantErr: true,
		},
		{
			name:    "invalid unit",
			input:   "7x",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "negative number",
			input:   "-7d",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestSinceToUnixTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		since    string
		wantErr  bool
		checkFn  func(t *testing.T, result int64)
	}{
		{
			name:    "7 days ago",
			since:   "7d",
			wantErr: false,
			checkFn: func(t *testing.T, result int64) {
				expected := now.Add(-7 * 24 * time.Hour).Unix()
				// Allow 2 second tolerance for test execution time
				assert.InDelta(t, expected, result, 2)
			},
		},
		{
			name:    "1 week ago",
			since:   "1w",
			wantErr: false,
			checkFn: func(t *testing.T, result int64) {
				expected := now.Add(-7 * 24 * time.Hour).Unix()
				assert.InDelta(t, expected, result, 2)
			},
		},
		{
			name:    "invalid duration",
			since:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SinceToUnixTime(tt.since)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.checkFn != nil {
					tt.checkFn(t, result)
				}
			}
		})
	}
}

func TestBuildQueryOptions(t *testing.T) {
	tests := []struct {
		name        string
		limit       int
		offset      int
		unread      bool
		since       string
		tag         string
		expectError bool
		checkOpts   func(t *testing.T, opts QueryOptions)
	}{
		{
			name:   "basic pagination",
			limit:  20,
			offset: 40,
			checkOpts: func(t *testing.T, opts QueryOptions) {
				assert.Equal(t, 20, opts.Limit)
				assert.Equal(t, 40, opts.Offset)
				assert.False(t, opts.UnreadOnly)
				assert.Nil(t, opts.SinceTime)
			},
		},
		{
			name:   "unread filter",
			unread: true,
			checkOpts: func(t *testing.T, opts QueryOptions) {
				assert.True(t, opts.UnreadOnly)
			},
		},
		{
			name:  "since filter",
			since: "7d",
			checkOpts: func(t *testing.T, opts QueryOptions) {
				require.NotNil(t, opts.SinceTime)
				// Should be approximately 7 days ago
				expected := time.Now().Add(-7 * 24 * time.Hour).Unix()
				assert.InDelta(t, expected, *opts.SinceTime, 2)
			},
		},
		{
			name:  "tag filter",
			tag:   "golang",
			checkOpts: func(t *testing.T, opts QueryOptions) {
				assert.Equal(t, "golang", opts.Tag)
			},
		},
		{
			name:   "combined filters",
			limit:  10,
			offset: 0,
			unread: true,
			since:  "2w",
			tag:    "rust",
			checkOpts: func(t *testing.T, opts QueryOptions) {
				assert.Equal(t, 10, opts.Limit)
				assert.Equal(t, 0, opts.Offset)
				assert.True(t, opts.UnreadOnly)
				assert.Equal(t, "rust", opts.Tag)
				require.NotNil(t, opts.SinceTime)
			},
		},
		{
			name:        "invalid since format",
			since:       "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := BuildQueryOptions(tt.limit, tt.offset, tt.unread, tt.since, tt.tag)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.checkOpts != nil {
					tt.checkOpts(t, opts)
				}
			}
		})
	}
}

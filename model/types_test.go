package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFeed_Validation(t *testing.T) {
	tests := []struct {
		name    string
		feed    Feed
		wantErr bool
	}{
		{
			name: "valid feed",
			feed: Feed{
				URL:      "https://example.com/rss",
				Title:    "Example Feed",
				Category: "tech",
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			feed: Feed{
				Title:    "Example Feed",
				Category: "tech",
			},
			wantErr: true,
		},
		{
			name: "empty URL",
			feed: Feed{
				URL:      "",
				Title:    "Example Feed",
				Category: "tech",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.feed.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEntry_IsUnread(t *testing.T) {
	tests := []struct {
		name   string
		entry  Entry
		expect bool
	}{
		{
			name:   "unread entry",
			entry:  Entry{IsRead: false},
			expect: true,
		},
		{
			name:   "read entry",
			entry:  Entry{IsRead: true},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entry.IsUnread()
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestEntry_Age(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		entry     Entry
		expectAge time.Duration
	}{
		{
			name: "1 hour old",
			entry: Entry{
				Published: now.Add(-1 * time.Hour),
			},
			expectAge: time.Hour,
		},
		{
			name: "1 day old",
			entry: Entry{
				Published: now.Add(-24 * time.Hour),
			},
			expectAge: 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entry.Age()
			// Allow 1 second tolerance
			delta := got - tt.expectAge
			if delta < 0 {
				delta = -delta
			}
			assert.Less(t, delta, time.Second)
		})
	}
}

func TestEntry_HasTag(t *testing.T) {
	entry := Entry{
		Tags: []string{"golang", "programming", "tech"},
	}

	tests := []struct {
		tag    string
		expect bool
	}{
		{"golang", true},
		{"programming", true},
		{"tech", true},
		{"rust", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := entry.HasTag(tt.tag)
			assert.Equal(t, tt.expect, got)
		})
	}
}

// Package model defines the core data structures for feed-cli.
package model

import (
	"errors"
	"time"
)

// Feed represents an RSS/Atom feed source.
type Feed struct {
	ID           int64      `json:"id"`
	URL          string     `json:"url"`
	Title        string     `json:"title"`
	Category     string     `json:"category,omitempty"`
	LastUpdated  *time.Time `json:"last_updated,omitempty"`
	ETag         string     `json:"etag,omitempty"`
	LastModified string     `json:"last_modified,omitempty"`
}

// Validate checks if the feed has required fields.
func (f *Feed) Validate() error {
	if f.URL == "" {
		return errors.New("feed URL is required")
	}
	return nil
}

// Entry represents a single RSS/Atom entry/article.
type Entry struct {
	ID        int64     `json:"id"`
	FeedID    int64     `json:"feed_id"`
	GUID      string    `json:"guid"`
	Title     string    `json:"title"`
	Link      string    `json:"link"`
	Content   string    `json:"content"`
	Published time.Time `json:"published"`
	IsRead    bool      `json:"is_read"`
	Tags      []string  `json:"tags,omitempty"`
}

// IsUnread returns true if the entry hasn't been read.
func (e *Entry) IsUnread() bool {
	return !e.IsRead
}

// Age returns how long ago the entry was published.
func (e *Entry) Age() time.Duration {
	return time.Since(e.Published)
}

// HasTag checks if the entry has the specified tag.
func (e *Entry) HasTag(tag string) bool {
	for _, t := range e.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Tag represents a label that can be applied to entries.
type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

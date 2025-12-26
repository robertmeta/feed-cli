// Package feed provides RSS/Atom feed fetching and parsing for feed-cli.
package feed

import (
	"fmt"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/robertmeta/feed-cli/model"
)

// Fetcher handles fetching and parsing RSS/Atom feeds.
type Fetcher struct {
	parser *gofeed.Parser
}

// NewFetcher creates a new Fetcher.
func NewFetcher() *Fetcher {
	return &Fetcher{
		parser: gofeed.NewParser(),
	}
}

// Fetch retrieves and parses a feed from a URL.
func (f *Fetcher) Fetch(url string) (*model.Feed, []*model.Entry, error) {
	parsedFeed, err := f.parser.ParseURL(url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch feed from %s: %w", url, err)
	}

	feed, entries := f.convert(parsedFeed, url)
	return feed, entries, nil
}

// Parse parses feed content from a string.
func (f *Fetcher) Parse(content string) (*model.Feed, []*model.Entry, error) {
	if content == "" {
		return nil, nil, fmt.Errorf("feed content is empty")
	}

	parsedFeed, err := f.parser.ParseString(content)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	feed, entries := f.convert(parsedFeed, "")
	return feed, entries, nil
}

// convert converts a gofeed.Feed to our model types.
func (f *Fetcher) convert(gf *gofeed.Feed, url string) (*model.Feed, []*model.Entry) {
	// Convert feed metadata
	feed := &model.Feed{
		Title: gf.Title,
		URL:   url,
	}

	// Use feed link if URL not provided
	if feed.URL == "" && gf.Link != "" {
		feed.URL = gf.Link
	}

	// Convert entries
	var entries []*model.Entry
	for _, item := range gf.Items {
		entry := f.convertItem(item)
		entries = append(entries, entry)
	}

	return feed, entries
}

// convertItem converts a gofeed.Item to a model.Entry.
func (f *Fetcher) convertItem(item *gofeed.Item) *model.Entry {
	entry := &model.Entry{
		GUID:   item.GUID,
		Title:  item.Title,
		Link:   item.Link,
		IsRead: false, // New entries default to unread
	}

	// Use link as GUID if GUID is missing
	if entry.GUID == "" {
		entry.GUID = item.Link
	}

	// Get content (prefer full content over description)
	if item.Content != "" {
		entry.Content = item.Content
	} else if item.Description != "" {
		entry.Content = item.Description
	}

	// Parse published date
	if item.PublishedParsed != nil {
		entry.Published = *item.PublishedParsed
	} else if item.UpdatedParsed != nil {
		entry.Published = *item.UpdatedParsed
	} else {
		// Fallback to current time if no date found
		entry.Published = time.Now()
	}

	return entry
}

// FetchWithCache retrieves a feed with HTTP caching support (ETag, Last-Modified).
// Returns the feed, entries, whether it was modified (true = new content, false = not modified), and any error.
func (f *Fetcher) FetchWithCache(url string, etag string, lastModified string) (*model.Feed, []*model.Entry, bool, error) {
	// For now, just fetch normally
	// TODO: Implement HTTP conditional GET with If-None-Match and If-Modified-Since headers
	feed, entries, err := f.Fetch(url)
	if err != nil {
		return nil, nil, false, err
	}

	// Always return modified=true for now (no caching yet)
	return feed, entries, true, nil
}

// ExtractCategories extracts categories/tags from feed entries.
func ExtractCategories(content string) []string {
	// Simple category extraction from content
	// This is a placeholder - can be enhanced later
	categories := []string{}

	content = strings.ToLower(content)

	// Check for common tech keywords
	keywords := map[string]string{
		"golang": "golang",
		"go ":    "golang",
		"rust":   "rust",
		"python": "python",
		"javascript": "javascript",
		"typescript": "typescript",
	}

	for keyword, category := range keywords {
		if strings.Contains(content, keyword) {
			categories = append(categories, category)
		}
	}

	return categories
}

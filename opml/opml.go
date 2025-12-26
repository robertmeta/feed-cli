// Package opml provides OPML import and export functionality for feed-cli.
package opml

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/robertmeta/feed-cli/model"
)

// OPML represents the root OPML structure.
type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    Head     `xml:"head"`
	Body    Body     `xml:"body"`
}

// Head contains metadata about the OPML document.
type Head struct {
	Title       string `xml:"title,omitempty"`
	DateCreated string `xml:"dateCreated,omitempty"`
}

// Body contains the outline elements (feeds).
type Body struct {
	Outlines []Outline `xml:"outline"`
}

// Outline represents a feed or category in OPML.
type Outline struct {
	Text     string    `xml:"text,attr,omitempty"`
	Title    string    `xml:"title,attr,omitempty"`
	Type     string    `xml:"type,attr,omitempty"`
	XMLUrl   string    `xml:"xmlUrl,attr,omitempty"`
	Category string    `xml:"category,attr,omitempty"`
	Outlines []Outline `xml:"outline,omitempty"`
}

// Parse reads an OPML file and extracts feeds.
func Parse(r io.Reader) ([]*model.Feed, error) {
	var opml OPML
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&opml); err != nil {
		return nil, fmt.Errorf("failed to parse OPML: %w", err)
	}

	var feeds []*model.Feed
	feeds = extractFeeds(opml.Body.Outlines, "")

	return feeds, nil
}

// extractFeeds recursively extracts feeds from outlines.
// parentCategory is used for nested outlines that don't specify their own category.
func extractFeeds(outlines []Outline, parentCategory string) []*model.Feed {
	var feeds []*model.Feed

	for _, outline := range outlines {
		// If this outline has an xmlUrl, it's a feed
		if outline.XMLUrl != "" {
			feed := &model.Feed{
				URL:   outline.XMLUrl,
				Title: outline.Title,
			}

			// Use explicit category if provided, otherwise inherit from parent
			if outline.Category != "" {
				feed.Category = outline.Category
			} else if parentCategory != "" {
				feed.Category = parentCategory
			}

			// Fallback to text if title is empty
			if feed.Title == "" {
				feed.Title = outline.Text
			}

			feeds = append(feeds, feed)
		}

		// Recursively process nested outlines
		if len(outline.Outlines) > 0 {
			// Use outline text as category for children if they don't have one
			categoryForChildren := outline.Text
			if categoryForChildren == "" {
				categoryForChildren = parentCategory
			}

			childFeeds := extractFeeds(outline.Outlines, categoryForChildren)
			feeds = append(feeds, childFeeds...)
		}
	}

	return feeds
}

// Generate creates an OPML file from a list of feeds.
func Generate(w io.Writer, feeds []*model.Feed) error {
	// Group feeds by category
	categories := make(map[string][]*model.Feed)
	var uncategorized []*model.Feed

	for _, feed := range feeds {
		if feed.Category == "" {
			uncategorized = append(uncategorized, feed)
		} else {
			categories[feed.Category] = append(categories[feed.Category], feed)
		}
	}

	// Build OPML structure
	opml := OPML{
		Version: "2.0",
		Head: Head{
			Title:       "feed-cli Subscriptions",
			DateCreated: time.Now().Format(time.RFC1123),
		},
		Body: Body{
			Outlines: []Outline{},
		},
	}

	// Add categorized feeds
	for category, categoryFeeds := range categories {
		categoryOutline := Outline{
			Text:     category,
			Title:    category,
			Outlines: []Outline{},
		}

		for _, feed := range categoryFeeds {
			feedOutline := Outline{
				Type:     "rss",
				Text:     feed.Title,
				Title:    feed.Title,
				XMLUrl:   feed.URL,
				Category: feed.Category,
			}
			categoryOutline.Outlines = append(categoryOutline.Outlines, feedOutline)
		}

		opml.Body.Outlines = append(opml.Body.Outlines, categoryOutline)
	}

	// Add uncategorized feeds directly to body
	for _, feed := range uncategorized {
		feedOutline := Outline{
			Type:   "rss",
			Text:   feed.Title,
			Title:  feed.Title,
			XMLUrl: feed.URL,
		}
		opml.Body.Outlines = append(opml.Body.Outlines, feedOutline)
	}

	// Write XML with indentation
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")

	// Write XML declaration
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf("failed to write XML header: %w", err)
	}

	// Encode OPML
	if err := encoder.Encode(opml); err != nil {
		return fmt.Errorf("failed to encode OPML: %w", err)
	}

	// Add final newline
	if _, err := w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write final newline: %w", err)
	}

	return nil
}

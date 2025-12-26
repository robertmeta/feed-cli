package feed

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetcher_ParseRSS2(t *testing.T) {
	// Read RSS 2.0 fixture
	data, err := os.ReadFile("../testdata/rss2.xml")
	require.NoError(t, err)

	fetcher := NewFetcher()
	feed, entries, err := fetcher.Parse(string(data))
	require.NoError(t, err)

	// Verify feed metadata
	assert.Equal(t, "Test RSS Feed", feed.Title)
	assert.NotEmpty(t, feed.URL)

	// Verify entries
	require.Len(t, entries, 3, "Should parse 3 entries from RSS feed")

	// Check first entry
	assert.Equal(t, "First Test Entry", entries[0].Title)
	assert.Equal(t, "https://example.com/entry-1", entries[0].Link)
	assert.Equal(t, "entry-1", entries[0].GUID)
	assert.Contains(t, entries[0].Content, "first test entry")
	assert.False(t, entries[0].Published.IsZero(), "Published date should be set")

	// Check second entry
	assert.Equal(t, "Second Test Entry", entries[1].Title)
	assert.Equal(t, "entry-2", entries[1].GUID)

	// Check third entry
	assert.Equal(t, "Third Test Entry", entries[2].Title)
}

func TestFetcher_ParseAtom(t *testing.T) {
	// Read Atom fixture
	data, err := os.ReadFile("../testdata/atom.xml")
	require.NoError(t, err)

	fetcher := NewFetcher()
	feed, entries, err := fetcher.Parse(string(data))
	require.NoError(t, err)

	// Verify feed metadata
	assert.Equal(t, "Test Atom Feed", feed.Title)

	// Verify entries
	require.Len(t, entries, 2, "Should parse 2 entries from Atom feed")

	// Check first entry
	assert.Equal(t, "First Atom Entry", entries[0].Title)
	assert.Equal(t, "https://example.com/atom-entry-1", entries[0].Link)
	assert.Equal(t, "atom-entry-1", entries[0].GUID)
	assert.Contains(t, entries[0].Content, "HTML content")
	assert.False(t, entries[0].Published.IsZero())

	// Check second entry
	assert.Equal(t, "Second Atom Entry", entries[1].Title)
}

func TestFetcher_ParseInvalidFeed(t *testing.T) {
	fetcher := NewFetcher()

	// Test with invalid XML
	_, _, err := fetcher.Parse("<invalid>xml</broken>")
	assert.Error(t, err, "Should error on invalid XML")

	// Test with empty string
	_, _, err = fetcher.Parse("")
	assert.Error(t, err, "Should error on empty string")

	// Test with non-feed XML
	_, _, err = fetcher.Parse("<?xml version='1.0'?><root><item>not a feed</item></root>")
	assert.Error(t, err, "Should error on non-feed XML")
}

func TestFetcher_FetchURL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	fetcher := NewFetcher()

	// Test with a known-good RSS feed (Hacker News)
	feed, entries, err := fetcher.Fetch("https://news.ycombinator.com/rss")
	require.NoError(t, err)
	assert.NotEmpty(t, feed.Title)
	assert.NotEmpty(t, entries, "Should fetch at least some entries")

	// Verify entries have required fields
	for _, e := range entries {
		assert.NotEmpty(t, e.GUID, "Entry should have GUID")
		assert.NotEmpty(t, e.Title, "Entry should have title")
		assert.NotEmpty(t, e.Link, "Entry should have link")
	}
}

func TestFetcher_FetchInvalidURL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	fetcher := NewFetcher()

	// Test with invalid URL
	_, _, err := fetcher.Fetch("not-a-valid-url")
	assert.Error(t, err)

	// Test with non-existent domain
	_, _, err = fetcher.Fetch("https://this-domain-definitely-does-not-exist-12345.com/rss")
	assert.Error(t, err)

	// Test with URL that returns non-feed content
	_, _, err = fetcher.Fetch("https://example.com")
	assert.Error(t, err)
}

func TestFetcher_EntriesDefaultToUnread(t *testing.T) {
	data, err := os.ReadFile("../testdata/rss2.xml")
	require.NoError(t, err)

	fetcher := NewFetcher()
	_, entries, err := fetcher.Parse(string(data))
	require.NoError(t, err)

	// All entries should default to unread
	for _, e := range entries {
		assert.False(t, e.IsRead, "Newly fetched entries should be unread by default")
	}
}

func TestFetcher_HandlesEmptyContent(t *testing.T) {
	// Feed with entry that has no description/content
	minimalRSS := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Minimal Feed</title>
    <item>
      <title>Entry with no content</title>
      <link>https://example.com/minimal</link>
      <guid>minimal-1</guid>
    </item>
  </channel>
</rss>`

	fetcher := NewFetcher()
	_, entries, err := fetcher.Parse(minimalRSS)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	// Should handle missing content gracefully
	assert.Equal(t, "Entry with no content", entries[0].Title)
	assert.Equal(t, "", entries[0].Content) // Empty content is OK
}

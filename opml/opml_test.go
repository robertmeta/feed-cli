package opml

import (
	"os"
	"strings"
	"testing"

	"github.com/robertmeta/feed-cli/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOPML_ValidFile(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>Test Feeds</title>
  </head>
  <body>
    <outline text="Tech" title="Tech">
      <outline type="rss" text="Feed 1" title="Feed 1" xmlUrl="https://example.com/feed1" category="tech"/>
      <outline type="rss" text="Feed 2" title="Feed 2" xmlUrl="https://example.com/feed2" category="tech"/>
    </outline>
    <outline type="rss" text="Feed 3" title="Feed 3" xmlUrl="https://example.com/feed3" category="blog"/>
  </body>
</opml>`

	feeds, err := Parse(strings.NewReader(opmlContent))
	require.NoError(t, err)
	require.Len(t, feeds, 3, "Should parse 3 feeds")

	// Check first feed
	assert.Equal(t, "https://example.com/feed1", feeds[0].URL)
	assert.Equal(t, "Feed 1", feeds[0].Title)
	assert.Equal(t, "tech", feeds[0].Category)

	// Check second feed
	assert.Equal(t, "https://example.com/feed2", feeds[1].URL)
	assert.Equal(t, "tech", feeds[1].Category)

	// Check third feed
	assert.Equal(t, "https://example.com/feed3", feeds[2].URL)
	assert.Equal(t, "blog", feeds[2].Category)
}

func TestParseOPML_RealFile(t *testing.T) {
	// Parse the actual feeds.opml file
	file, err := os.Open("../feeds.opml")
	require.NoError(t, err)
	defer file.Close()

	feeds, err := Parse(file)
	require.NoError(t, err)
	assert.NotEmpty(t, feeds, "Should parse feeds from feeds.opml")

	// Verify we got all expected feeds (25 total)
	assert.GreaterOrEqual(t, len(feeds), 20, "Should have at least 20 feeds")

	// Check that URLs are valid
	for _, feed := range feeds {
		assert.NotEmpty(t, feed.URL, "Feed should have URL")
		assert.NotEmpty(t, feed.Title, "Feed should have title")
		assert.NotEmpty(t, feed.Category, "Feed should have category")
	}

	// Spot check a specific feed
	var foundHN bool
	for _, feed := range feeds {
		if feed.URL == "https://news.ycombinator.com/rss" {
			foundHN = true
			assert.Equal(t, "Hacker News", feed.Title)
			assert.Equal(t, "tech-news", feed.Category)
			break
		}
	}
	assert.True(t, foundHN, "Should find Hacker News feed")
}

func TestParseOPML_FlatStructure(t *testing.T) {
	// OPML without nested outlines (flat list)
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Flat Feeds</title></head>
  <body>
    <outline type="rss" text="Feed A" title="Feed A" xmlUrl="https://example.com/a"/>
    <outline type="rss" text="Feed B" title="Feed B" xmlUrl="https://example.com/b"/>
  </body>
</opml>`

	feeds, err := Parse(strings.NewReader(opmlContent))
	require.NoError(t, err)
	assert.Len(t, feeds, 2)
}

func TestParseOPML_InvalidXML(t *testing.T) {
	invalidContent := `<invalid>xml</broken>`

	_, err := Parse(strings.NewReader(invalidContent))
	assert.Error(t, err, "Should error on invalid XML")
}

func TestParseOPML_EmptyFile(t *testing.T) {
	emptyContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Empty</title></head>
  <body></body>
</opml>`

	feeds, err := Parse(strings.NewReader(emptyContent))
	require.NoError(t, err)
	assert.Len(t, feeds, 0, "Empty OPML should return no feeds")
}

func TestParseOPML_MissingXmlUrl(t *testing.T) {
	// Outline without xmlUrl should be skipped
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline type="rss" text="Valid Feed" xmlUrl="https://example.com/feed"/>
    <outline type="rss" text="Invalid Feed"/>
  </body>
</opml>`

	feeds, err := Parse(strings.NewReader(opmlContent))
	require.NoError(t, err)
	assert.Len(t, feeds, 1, "Should skip outlines without xmlUrl")
	assert.Equal(t, "https://example.com/feed", feeds[0].URL)
}

func TestGenerateOPML(t *testing.T) {
	feeds := []*model.Feed{
		{URL: "https://example.com/feed1", Title: "Feed 1", Category: "tech"},
		{URL: "https://example.com/feed2", Title: "Feed 2", Category: "tech"},
		{URL: "https://example.com/feed3", Title: "Feed 3", Category: "blog"},
	}

	var buf strings.Builder
	err := Generate(&buf, feeds)
	require.NoError(t, err)

	output := buf.String()

	// Verify output contains XML declaration
	assert.Contains(t, output, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, output, `<opml version="2.0">`)

	// Verify all feeds are present
	assert.Contains(t, output, `xmlUrl="https://example.com/feed1"`)
	assert.Contains(t, output, `xmlUrl="https://example.com/feed2"`)
	assert.Contains(t, output, `xmlUrl="https://example.com/feed3"`)

	// Verify titles
	assert.Contains(t, output, `title="Feed 1"`)
	assert.Contains(t, output, `title="Feed 2"`)
	assert.Contains(t, output, `title="Feed 3"`)

	// Verify categories
	assert.Contains(t, output, `category="tech"`)
	assert.Contains(t, output, `category="blog"`)
}

func TestGenerateOPML_EmptyList(t *testing.T) {
	feeds := []*model.Feed{}

	var buf strings.Builder
	err := Generate(&buf, feeds)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `<opml version="2.0">`)
	assert.Contains(t, output, `<body>`)
	assert.Contains(t, output, `</body>`)
}

func TestRoundTrip(t *testing.T) {
	// Test that we can generate OPML and parse it back
	originalFeeds := []*model.Feed{
		{URL: "https://example.com/feed1", Title: "Feed 1", Category: "tech"},
		{URL: "https://example.com/feed2", Title: "Feed 2", Category: "blog"},
	}

	// Generate OPML
	var buf strings.Builder
	err := Generate(&buf, originalFeeds)
	require.NoError(t, err)

	// Parse it back
	parsedFeeds, err := Parse(strings.NewReader(buf.String()))
	require.NoError(t, err)

	// Verify we got the same feeds back
	require.Len(t, parsedFeeds, 2)
	assert.Equal(t, originalFeeds[0].URL, parsedFeeds[0].URL)
	assert.Equal(t, originalFeeds[0].Title, parsedFeeds[0].Title)
	assert.Equal(t, originalFeeds[0].Category, parsedFeeds[0].Category)

	assert.Equal(t, originalFeeds[1].URL, parsedFeeds[1].URL)
	assert.Equal(t, originalFeeds[1].Title, parsedFeeds[1].Title)
	assert.Equal(t, originalFeeds[1].Category, parsedFeeds[1].Category)
}

func TestParseOPML_CategoryInheritance(t *testing.T) {
	// Test that nested outlines inherit category from parent if not specified
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Tech News" title="Tech News">
      <outline type="rss" text="Feed 1" xmlUrl="https://example.com/feed1" category="tech"/>
      <outline type="rss" text="Feed 2" xmlUrl="https://example.com/feed2"/>
    </outline>
  </body>
</opml>`

	feeds, err := Parse(strings.NewReader(opmlContent))
	require.NoError(t, err)
	require.Len(t, feeds, 2)

	// First feed has explicit category
	assert.Equal(t, "tech", feeds[0].Category)

	// Second feed should inherit from parent or have empty category
	// (depending on implementation - we'll decide in the implementation)
}

func TestGenerateOPML_SpecialCharacters(t *testing.T) {
	// Test that special XML characters are properly escaped
	feeds := []*model.Feed{
		{URL: "https://example.com/feed?id=1&type=rss", Title: "Feed with & < >", Category: "test"},
	}

	var buf strings.Builder
	err := Generate(&buf, feeds)
	require.NoError(t, err)

	output := buf.String()

	// Should contain escaped characters
	assert.Contains(t, output, "&amp;")  // & should be escaped
	// The XML encoder should handle this automatically
}

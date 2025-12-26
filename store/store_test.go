package store

import (
	"testing"
	"time"

	"github.com/robertmeta/feed-cli/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	// Test creating a new in-memory database
	s, err := New(":memory:")
	require.NoError(t, err)
	require.NotNil(t, s)
	defer s.Close()
}

func TestStore_SaveAndGetFeed(t *testing.T) {
	s, err := New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	feed := &model.Feed{
		URL:      "https://example.com/rss",
		Title:    "Example Feed",
		Category: "tech",
	}

	// Save feed
	err = s.SaveFeed(feed)
	require.NoError(t, err)
	assert.NotZero(t, feed.ID, "Feed ID should be set after save")

	// Get feed by ID
	got, err := s.GetFeed(feed.ID)
	require.NoError(t, err)
	assert.Equal(t, feed.URL, got.URL)
	assert.Equal(t, feed.Title, got.Title)
	assert.Equal(t, feed.Category, got.Category)
}

func TestStore_GetAllFeeds(t *testing.T) {
	s, err := New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	// Save multiple feeds
	feeds := []*model.Feed{
		{URL: "https://example1.com/rss", Title: "Feed 1", Category: "tech"},
		{URL: "https://example2.com/rss", Title: "Feed 2", Category: "news"},
		{URL: "https://example3.com/rss", Title: "Feed 3", Category: "tech"},
	}

	for _, f := range feeds {
		err := s.SaveFeed(f)
		require.NoError(t, err)
	}

	// Get all feeds
	all, err := s.GetAllFeeds()
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestStore_DeleteFeed(t *testing.T) {
	s, err := New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	feed := &model.Feed{
		URL:   "https://example.com/rss",
		Title: "Example Feed",
	}

	err = s.SaveFeed(feed)
	require.NoError(t, err)

	// Delete feed
	err = s.DeleteFeed(feed.ID)
	require.NoError(t, err)

	// Verify it's gone
	_, err = s.GetFeed(feed.ID)
	assert.Error(t, err, "Should error when getting deleted feed")
}

func TestStore_SaveAndGetEntry(t *testing.T) {
	s, err := New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	// First create a feed
	feed := &model.Feed{
		URL:   "https://example.com/rss",
		Title: "Example Feed",
	}
	err = s.SaveFeed(feed)
	require.NoError(t, err)

	// Create an entry
	entry := &model.Entry{
		FeedID:    feed.ID,
		GUID:      "entry-1",
		Title:     "Test Entry",
		Link:      "https://example.com/entry-1",
		Content:   "Test content",
		Published: time.Now(),
		IsRead:    false,
	}

	// Save entry
	err = s.SaveEntry(entry)
	require.NoError(t, err)
	assert.NotZero(t, entry.ID, "Entry ID should be set after save")

	// Get entry by ID
	got, err := s.GetEntry(entry.ID)
	require.NoError(t, err)
	assert.Equal(t, entry.GUID, got.GUID)
	assert.Equal(t, entry.Title, got.Title)
	assert.Equal(t, entry.Link, got.Link)
	assert.Equal(t, entry.IsRead, got.IsRead)
}

func TestStore_GetEntries_Pagination(t *testing.T) {
	s, err := New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	// Create a feed
	feed := &model.Feed{
		URL:   "https://example.com/rss",
		Title: "Example Feed",
	}
	err = s.SaveFeed(feed)
	require.NoError(t, err)

	// Create 50 entries
	baseTime := time.Now()
	for i := 0; i < 50; i++ {
		entry := &model.Entry{
			FeedID:    feed.ID,
			GUID:      string(rune('a' + i)),
			Title:     "Entry " + string(rune('a'+i)),
			Link:      "https://example.com/entry-" + string(rune('a'+i)),
			Published: baseTime.Add(-time.Duration(i) * time.Hour), // Older entries
			IsRead:    false,
		}
		err = s.SaveEntry(entry)
		require.NoError(t, err)
	}

	// Test pagination: Get first 10 entries (offset 0, limit 10)
	entries, err := s.GetEntries(QueryOptions{
		Limit:  10,
		Offset: 0,
	})
	require.NoError(t, err)
	assert.Len(t, entries, 10, "Should get 10 entries")

	// Test offset: Get next 10 entries (offset 10, limit 10)
	entries2, err := s.GetEntries(QueryOptions{
		Limit:  10,
		Offset: 10,
	})
	require.NoError(t, err)
	assert.Len(t, entries2, 10, "Should get next 10 entries")

	// Verify entries are different
	assert.NotEqual(t, entries[0].ID, entries2[0].ID, "Offset should return different entries")

	// Test getting last page (offset 45, limit 10) - should only get 5
	entries3, err := s.GetEntries(QueryOptions{
		Limit:  10,
		Offset: 45,
	})
	require.NoError(t, err)
	assert.Len(t, entries3, 5, "Should get remaining 5 entries")
}

func TestStore_GetEntries_UnreadFilter(t *testing.T) {
	s, err := New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	// Create a feed
	feed := &model.Feed{
		URL:   "https://example.com/rss",
		Title: "Example Feed",
	}
	err = s.SaveFeed(feed)
	require.NoError(t, err)

	// Create entries - some read, some unread
	for i := 0; i < 10; i++ {
		entry := &model.Entry{
			FeedID:    feed.ID,
			GUID:      string(rune('a' + i)),
			Title:     "Entry " + string(rune('a'+i)),
			Link:      "https://example.com/entry",
			Published: time.Now(),
			IsRead:    i%2 == 0, // Every other entry is read
		}
		err = s.SaveEntry(entry)
		require.NoError(t, err)
	}

	// Get only unread entries
	unread, err := s.GetEntries(QueryOptions{
		UnreadOnly: true,
	})
	require.NoError(t, err)
	assert.Len(t, unread, 5, "Should get 5 unread entries")

	// Verify all are unread
	for _, e := range unread {
		assert.False(t, e.IsRead, "All entries should be unread")
	}
}

func TestStore_MarkEntryRead(t *testing.T) {
	s, err := New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	// Create feed and entry
	feed := &model.Feed{URL: "https://example.com/rss", Title: "Test"}
	err = s.SaveFeed(feed)
	require.NoError(t, err)

	entry := &model.Entry{
		FeedID:    feed.ID,
		GUID:      "test-guid",
		Title:     "Test Entry",
		Published: time.Now(),
		IsRead:    false,
	}
	err = s.SaveEntry(entry)
	require.NoError(t, err)

	// Mark as read
	err = s.MarkEntryRead(entry.ID, true)
	require.NoError(t, err)

	// Verify
	got, err := s.GetEntry(entry.ID)
	require.NoError(t, err)
	assert.True(t, got.IsRead)

	// Mark as unread
	err = s.MarkEntryRead(entry.ID, false)
	require.NoError(t, err)

	got, err = s.GetEntry(entry.ID)
	require.NoError(t, err)
	assert.False(t, got.IsRead)
}

func TestStore_UniqueConstraints(t *testing.T) {
	s, err := New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	// Create feed
	feed := &model.Feed{
		URL:   "https://example.com/rss",
		Title: "Test Feed",
	}
	err = s.SaveFeed(feed)
	require.NoError(t, err)

	// Try to create duplicate feed URL
	duplicate := &model.Feed{
		URL:   "https://example.com/rss", // Same URL
		Title: "Different Title",
	}
	err = s.SaveFeed(duplicate)
	assert.Error(t, err, "Should error on duplicate feed URL")

	// Create entry
	entry := &model.Entry{
		FeedID:    feed.ID,
		GUID:      "unique-guid",
		Title:     "Entry",
		Published: time.Now(),
	}
	err = s.SaveEntry(entry)
	require.NoError(t, err)

	// Try to create duplicate GUID for same feed
	duplicateEntry := &model.Entry{
		FeedID:    feed.ID,
		GUID:      "unique-guid", // Same GUID
		Title:     "Different Entry",
		Published: time.Now(),
	}
	err = s.SaveEntry(duplicateEntry)
	assert.Error(t, err, "Should error on duplicate GUID in same feed")
}

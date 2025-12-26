// Package store provides SQLite database operations for feed-cli.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/robertmeta/feed-cli/model"
	_ "modernc.org/sqlite"
)

// Store manages the SQLite database.
type Store struct {
	db *sql.DB
}

// QueryOptions specifies how to query entries.
type QueryOptions struct {
	Limit      int
	Offset     int
	UnreadOnly bool
	Tag        string
	SinceTime  *int64 // Unix timestamp
}

// New creates a new Store with the given database path.
// Use ":memory:" for an in-memory database (useful for testing).
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}

	// Initialize schema
	if err := store.createSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return store, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// createSchema creates the database tables and indexes.
func (s *Store) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT UNIQUE NOT NULL,
		title TEXT,
		category TEXT,
		last_updated INTEGER,
		etag TEXT,
		last_modified TEXT
	);

	CREATE TABLE IF NOT EXISTS entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		feed_id INTEGER NOT NULL,
		guid TEXT NOT NULL,
		title TEXT,
		link TEXT,
		content TEXT,
		published INTEGER NOT NULL,
		is_read INTEGER DEFAULT 0,
		FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
		UNIQUE(feed_id, guid)
	);

	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS entry_tags (
		entry_id INTEGER NOT NULL,
		tag_id INTEGER NOT NULL,
		PRIMARY KEY (entry_id, tag_id),
		FOREIGN KEY (entry_id) REFERENCES entries(id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_entries_published ON entries(published DESC);
	CREATE INDEX IF NOT EXISTS idx_entries_is_read ON entries(is_read);
	CREATE INDEX IF NOT EXISTS idx_entries_feed_id ON entries(feed_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveFeed saves a feed to the database.
// If the feed has an ID of 0, it will be inserted. Otherwise, it will be updated.
func (s *Store) SaveFeed(f *model.Feed) error {
	if f.ID == 0 {
		// Insert
		result, err := s.db.Exec(
			"INSERT INTO feeds (url, title, category, etag, last_modified) VALUES (?, ?, ?, ?, ?)",
			f.URL, f.Title, f.Category, f.ETag, f.LastModified,
		)
		if err != nil {
			return fmt.Errorf("failed to insert feed: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		f.ID = id
		return nil
	}

	// Update
	_, err := s.db.Exec(
		"UPDATE feeds SET url = ?, title = ?, category = ?, etag = ?, last_modified = ? WHERE id = ?",
		f.URL, f.Title, f.Category, f.ETag, f.LastModified, f.ID,
	)
	return err
}

// GetFeed retrieves a feed by ID.
func (s *Store) GetFeed(id int64) (*model.Feed, error) {
	feed := &model.Feed{}
	err := s.db.QueryRow(
		"SELECT id, url, title, category, etag, last_modified FROM feeds WHERE id = ?",
		id,
	).Scan(&feed.ID, &feed.URL, &feed.Title, &feed.Category, &feed.ETag, &feed.LastModified)

	if err == sql.ErrNoRows {
		return nil, errors.New("feed not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	return feed, nil
}

// GetAllFeeds retrieves all feeds.
func (s *Store) GetAllFeeds() ([]*model.Feed, error) {
	rows, err := s.db.Query("SELECT id, url, title, category, etag, last_modified FROM feeds")
	if err != nil {
		return nil, fmt.Errorf("failed to query feeds: %w", err)
	}
	defer rows.Close()

	var feeds []*model.Feed
	for rows.Next() {
		feed := &model.Feed{}
		err := rows.Scan(&feed.ID, &feed.URL, &feed.Title, &feed.Category, &feed.ETag, &feed.LastModified)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feed: %w", err)
		}
		feeds = append(feeds, feed)
	}

	return feeds, rows.Err()
}

// DeleteFeed deletes a feed by ID.
func (s *Store) DeleteFeed(id int64) error {
	_, err := s.db.Exec("DELETE FROM feeds WHERE id = ?", id)
	return err
}

// SaveEntry saves an entry to the database.
func (s *Store) SaveEntry(e *model.Entry) error {
	if e.ID == 0 {
		// Insert
		result, err := s.db.Exec(
			"INSERT INTO entries (feed_id, guid, title, link, content, published, is_read) VALUES (?, ?, ?, ?, ?, ?, ?)",
			e.FeedID, e.GUID, e.Title, e.Link, e.Content, e.Published.Unix(), boolToInt(e.IsRead),
		)
		if err != nil {
			return fmt.Errorf("failed to insert entry: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		e.ID = id
		return nil
	}

	// Update
	_, err := s.db.Exec(
		"UPDATE entries SET feed_id = ?, guid = ?, title = ?, link = ?, content = ?, published = ?, is_read = ? WHERE id = ?",
		e.FeedID, e.GUID, e.Title, e.Link, e.Content, e.Published.Unix(), boolToInt(e.IsRead), e.ID,
	)
	return err
}

// GetEntry retrieves an entry by ID.
func (s *Store) GetEntry(id int64) (*model.Entry, error) {
	entry := &model.Entry{}
	var publishedUnix int64
	var isReadInt int

	err := s.db.QueryRow(
		"SELECT id, feed_id, guid, title, link, content, published, is_read FROM entries WHERE id = ?",
		id,
	).Scan(&entry.ID, &entry.FeedID, &entry.GUID, &entry.Title, &entry.Link, &entry.Content, &publishedUnix, &isReadInt)

	if err == sql.ErrNoRows {
		return nil, errors.New("entry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}

	entry.Published = unixToTime(publishedUnix)
	entry.IsRead = intToBool(isReadInt)

	return entry, nil
}

// GetEntries retrieves entries with optional filtering, pagination.
func (s *Store) GetEntries(opts QueryOptions) ([]*model.Entry, error) {
	query := "SELECT id, feed_id, guid, title, link, content, published, is_read FROM entries WHERE 1=1"
	args := []interface{}{}

	// Apply filters
	if opts.UnreadOnly {
		query += " AND is_read = 0"
	}

	if opts.SinceTime != nil {
		query += " AND published >= ?"
		args = append(args, *opts.SinceTime)
	}

	// Order by published date (newest first)
	query += " ORDER BY published DESC"

	// Apply pagination
	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	var entries []*model.Entry
	for rows.Next() {
		entry := &model.Entry{}
		var publishedUnix int64
		var isReadInt int

		err := rows.Scan(&entry.ID, &entry.FeedID, &entry.GUID, &entry.Title, &entry.Link, &entry.Content, &publishedUnix, &isReadInt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}

		entry.Published = unixToTime(publishedUnix)
		entry.IsRead = intToBool(isReadInt)
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// MarkEntryRead marks an entry as read or unread.
func (s *Store) MarkEntryRead(id int64, isRead bool) error {
	_, err := s.db.Exec("UPDATE entries SET is_read = ? WHERE id = ?", boolToInt(isRead), id)
	return err
}

// Helper functions for boolean<->int conversion (SQLite doesn't have BOOLEAN type)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}

// Helper to convert Unix timestamp to time.Time
func unixToTime(unix int64) time.Time {
	return time.Unix(unix, 0)
}

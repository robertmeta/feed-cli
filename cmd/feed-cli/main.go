package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/robertmeta/feed-cli/feed"
	"github.com/robertmeta/feed-cli/model"
	"github.com/robertmeta/feed-cli/opml"
	"github.com/robertmeta/feed-cli/store"
	"github.com/urfave/cli/v2"
)

const (
	ExitSuccess      = 0
	ExitGeneralError = 1
	ExitUsageError   = 2
	ExitDataError    = 3
)

func main() {
	app := &cli.App{
		Name:    "feed-cli",
		Usage:   "A scriptable RSS/Atom feed reader",
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "db",
				Aliases: []string{"d"},
				Value:   getDefaultDBPath(),
				Usage:   "Database file path",
				EnvVars: []string{"FEED_CLI_DB"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "add",
				Usage:     "Add a new feed",
				ArgsUsage: "<url>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "category",
						Aliases: []string{"c"},
						Usage:   "Feed category",
					},
				},
				Action: addFeed,
			},
			{
				Name:   "feeds",
				Usage:  "List all feeds",
				Action: listFeeds,
			},
			{
				Name:  "update",
				Usage: "Update feeds (fetch new entries)",
				Flags: []cli.Flag{
					&cli.Int64Flag{
						Name:    "feed-id",
						Aliases: []string{"f"},
						Usage:   "Update specific feed by ID (if not set, updates all)",
					},
				},
				Action: updateFeeds,
			},
			{
				Name:  "list",
				Usage: "List entries",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "limit",
						Aliases: []string{"l"},
						Value:   50,
						Usage:   "Maximum number of entries to return",
					},
					&cli.IntFlag{
						Name:    "offset",
						Aliases: []string{"o"},
						Value:   0,
						Usage:   "Offset for pagination",
					},
					&cli.BoolFlag{
						Name:    "unread",
						Aliases: []string{"u"},
						Usage:   "Show only unread entries",
					},
					&cli.StringFlag{
						Name:    "since",
						Aliases: []string{"s"},
						Usage:   "Show entries since duration (e.g., 7d, 2w, 3m, 1y)",
					},
					&cli.StringFlag{
						Name:    "tag",
						Aliases: []string{"t"},
						Usage:   "Filter by tag",
					},
				},
				Action: listEntries,
			},
			{
				Name:      "show",
				Usage:     "Show entry details",
				ArgsUsage: "<entry-id>",
				Action:    showEntry,
			},
			{
				Name:      "mark-read",
				Usage:     "Mark entries as read",
				ArgsUsage: "<entry-id>...",
				Action:    markRead,
			},
			{
				Name:   "mark-all-read",
				Usage:  "Mark all entries as read",
				Action: markAllRead,
			},
			{
				Name:      "remove",
				Usage:     "Remove a feed",
				ArgsUsage: "<feed-id>",
				Action:    removeFeed,
			},
			{
				Name:      "import",
				Usage:     "Import feeds from OPML file",
				ArgsUsage: "<opml-file>",
				Action:    importOPML,
			},
			{
				Name:  "export",
				Usage: "Export feeds to OPML file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output file (default: stdout)",
					},
				},
				Action: exportOPML,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitGeneralError)
	}
}

func getDefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "feed-cli.db"
	}
	return filepath.Join(home, ".config", "feed-cli", "feed-cli.db")
}

func getStore(c *cli.Context) (*store.Store, error) {
	dbPath := c.String("db")

	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	s, err := store.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return s, nil
}

func outputJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func addFeed(c *cli.Context) error {
	if c.NArg() < 1 {
		return cli.Exit("Usage: feed-cli add <url>", ExitUsageError)
	}

	url := c.Args().Get(0)
	category := c.String("category")

	s, err := getStore(c)
	if err != nil {
		return cli.Exit(err.Error(), ExitDataError)
	}
	defer s.Close()

	newFeed := &model.Feed{
		URL:      url,
		Category: category,
	}

	// Validate feed
	if err := newFeed.Validate(); err != nil {
		return cli.Exit(err.Error(), ExitDataError)
	}

	// Fetch feed to get title
	fetcher := feed.NewFetcher()
	parsedFeed, _, err := fetcher.Fetch(url)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Failed to fetch feed: %v", err), ExitDataError)
	}

	newFeed.Title = parsedFeed.Title

	// Save feed
	if err := s.SaveFeed(newFeed); err != nil {
		return cli.Exit(fmt.Sprintf("Failed to save feed: %v", err), ExitDataError)
	}

	return outputJSON(map[string]interface{}{
		"success": true,
		"feed":    newFeed,
	})
}

func listFeeds(c *cli.Context) error {
	s, err := getStore(c)
	if err != nil {
		return cli.Exit(err.Error(), ExitDataError)
	}
	defer s.Close()

	feeds, err := s.GetAllFeeds()
	if err != nil {
		return cli.Exit(fmt.Sprintf("Failed to get feeds: %v", err), ExitDataError)
	}

	return outputJSON(feeds)
}

func updateFeeds(c *cli.Context) error {
	s, err := getStore(c)
	if err != nil {
		return cli.Exit(err.Error(), ExitDataError)
	}
	defer s.Close()

	feedID := c.Int64("feed-id")
	fetcher := feed.NewFetcher()

	var feedsToUpdate []*model.Feed

	if feedID > 0 {
		// Update specific feed
		f, err := s.GetFeed(feedID)
		if err != nil {
			return cli.Exit(fmt.Sprintf("Failed to get feed: %v", err), ExitDataError)
		}
		feedsToUpdate = append(feedsToUpdate, f)
	} else {
		// Update all feeds
		feedsToUpdate, err = s.GetAllFeeds()
		if err != nil {
			return cli.Exit(fmt.Sprintf("Failed to get feeds: %v", err), ExitDataError)
		}
	}

	// Concurrent fetching with up to 50 parallel requests
	results := make(map[string]interface{})
	totalNewEntries := 0

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 50) // Limit to 50 concurrent fetches

	for _, f := range feedsToUpdate {
		wg.Add(1)
		go func(feedToUpdate *model.Feed) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }() // Release semaphore

			_, entries, err := fetcher.Fetch(feedToUpdate.URL)
			if err != nil {
				mu.Lock()
				results[feedToUpdate.URL] = map[string]interface{}{
					"error": err.Error(),
				}
				mu.Unlock()
				return
			}

			// Save entries
			newEntries := 0
			for _, entry := range entries {
				entry.FeedID = feedToUpdate.ID
				if err := s.SaveEntry(entry); err != nil {
					// Ignore duplicate entries (already exists)
					continue
				}
				newEntries++
			}

			mu.Lock()
			totalNewEntries += newEntries
			results[feedToUpdate.URL] = map[string]interface{}{
				"new_entries":   newEntries,
				"total_entries": len(entries),
			}
			mu.Unlock()
		}(f)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return outputJSON(map[string]interface{}{
		"updated_feeds":     len(feedsToUpdate),
		"total_new_entries": totalNewEntries,
		"results":           results,
	})
}

func listEntries(c *cli.Context) error {
	s, err := getStore(c)
	if err != nil {
		return cli.Exit(err.Error(), ExitDataError)
	}
	defer s.Close()

	opts, err := store.BuildQueryOptions(
		c.Int("limit"),
		c.Int("offset"),
		c.Bool("unread"),
		c.String("since"),
		c.String("tag"),
	)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Invalid query options: %v", err), ExitUsageError)
	}

	entries, err := s.GetEntries(opts)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Failed to get entries: %v", err), ExitDataError)
	}

	return outputJSON(map[string]interface{}{
		"count":   len(entries),
		"limit":   opts.Limit,
		"offset":  opts.Offset,
		"entries": entries,
	})
}

func showEntry(c *cli.Context) error {
	if c.NArg() < 1 {
		return cli.Exit("Usage: feed-cli show <entry-id>", ExitUsageError)
	}

	entryID := c.Args().Get(0)
	var id int64
	if _, err := fmt.Sscanf(entryID, "%d", &id); err != nil {
		return cli.Exit("Invalid entry ID", ExitUsageError)
	}

	s, err := getStore(c)
	if err != nil {
		return cli.Exit(err.Error(), ExitDataError)
	}
	defer s.Close()

	entry, err := s.GetEntry(id)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Failed to get entry: %v", err), ExitDataError)
	}

	return outputJSON(entry)
}

func markRead(c *cli.Context) error {
	if c.NArg() < 1 {
		return cli.Exit("Usage: feed-cli mark-read <entry-id>...", ExitUsageError)
	}

	s, err := getStore(c)
	if err != nil {
		return cli.Exit(err.Error(), ExitDataError)
	}
	defer s.Close()

	marked := 0
	for i := 0; i < c.NArg(); i++ {
		var id int64
		if _, err := fmt.Sscanf(c.Args().Get(i), "%d", &id); err != nil {
			continue
		}

		if err := s.MarkEntryRead(id, true); err != nil {
			continue
		}
		marked++
	}

	return outputJSON(map[string]interface{}{
		"marked_read": marked,
	})
}

func markAllRead(c *cli.Context) error {
	s, err := getStore(c)
	if err != nil {
		return cli.Exit(err.Error(), ExitDataError)
	}
	defer s.Close()

	// Get all unread entries
	opts := store.QueryOptions{UnreadOnly: true}
	entries, err := s.GetEntries(opts)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Failed to get entries: %v", err), ExitDataError)
	}

	// Mark all as read
	for _, entry := range entries {
		s.MarkEntryRead(entry.ID, true)
	}

	return outputJSON(map[string]interface{}{
		"marked_read": len(entries),
	})
}

func removeFeed(c *cli.Context) error {
	if c.NArg() < 1 {
		return cli.Exit("Usage: feed-cli remove <feed-id>", ExitUsageError)
	}

	var feedID int64
	if _, err := fmt.Sscanf(c.Args().Get(0), "%d", &feedID); err != nil {
		return cli.Exit("Invalid feed ID", ExitUsageError)
	}

	s, err := getStore(c)
	if err != nil {
		return cli.Exit(err.Error(), ExitDataError)
	}
	defer s.Close()

	if err := s.DeleteFeed(feedID); err != nil {
		return cli.Exit(fmt.Sprintf("Failed to delete feed: %v", err), ExitDataError)
	}

	return outputJSON(map[string]interface{}{
		"success": true,
		"feed_id": feedID,
	})
}

func importOPML(c *cli.Context) error {
	if c.NArg() < 1 {
		return cli.Exit("Usage: feed-cli import <opml-file>", ExitUsageError)
	}

	opmlPath := c.Args().Get(0)

	// Open OPML file
	file, err := os.Open(opmlPath)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Failed to open OPML file: %v", err), ExitDataError)
	}
	defer file.Close()

	// Parse OPML
	feeds, err := opml.Parse(file)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Failed to parse OPML: %v", err), ExitDataError)
	}

	// Open database
	s, err := getStore(c)
	if err != nil {
		return cli.Exit(err.Error(), ExitDataError)
	}
	defer s.Close()

	// Import feeds
	imported := 0
	skipped := 0
	var errors []string

	for _, newFeed := range feeds {
		if err := s.SaveFeed(newFeed); err != nil {
			// Feed might already exist (duplicate URL)
			skipped++
			errors = append(errors, fmt.Sprintf("%s: %v", newFeed.URL, err))
			continue
		}
		imported++
	}

	return outputJSON(map[string]interface{}{
		"success":  true,
		"imported": imported,
		"skipped":  skipped,
		"total":    len(feeds),
		"errors":   errors,
	})
}

func exportOPML(c *cli.Context) error {
	s, err := getStore(c)
	if err != nil {
		return cli.Exit(err.Error(), ExitDataError)
	}
	defer s.Close()

	// Get all feeds
	feeds, err := s.GetAllFeeds()
	if err != nil {
		return cli.Exit(fmt.Sprintf("Failed to get feeds: %v", err), ExitDataError)
	}

	// Determine output destination
	outputPath := c.String("output")
	var writer io.Writer

	if outputPath == "" {
		// Output to stdout
		writer = os.Stdout
	} else {
		// Output to file
		file, err := os.Create(outputPath)
		if err != nil {
			return cli.Exit(fmt.Sprintf("Failed to create output file: %v", err), ExitDataError)
		}
		defer file.Close()
		writer = file
	}

	// Generate OPML
	if err := opml.Generate(writer, feeds); err != nil {
		return cli.Exit(fmt.Sprintf("Failed to generate OPML: %v", err), ExitDataError)
	}

	// If outputting to file, also return JSON status
	if outputPath != "" {
		return outputJSON(map[string]interface{}{
			"success": true,
			"file":    outputPath,
			"count":   len(feeds),
		})
	}

	return nil
}

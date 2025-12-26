# feed-cli

A minimal, scriptable RSS/Atom feed reader for the command line. Designed for automation, scripting, and AI systems with JSON-first output.

## Features

- ✅ **Scriptable** - JSON output by default, perfect for piping to `jq` or other tools
- ✅ **Fast** - CGo-free SQLite for easy cross-compilation
- ✅ **Pagination** - Full `--offset` and `--limit` support
- ✅ **Date filtering** - Simple `--since 7d` syntax
- ✅ **Read tracking** - Mark entries as read/unread
- ✅ **Categories** - Organize feeds by category
- ✅ **TDD** - Comprehensive test coverage

## Installation

```bash
go install github.com/robertmeta/feed-cli/cmd/feed-cli@latest
```

Or build from source:

```bash
git clone https://github.com/robertmeta/feed-cli
cd feed-cli
make build
# Binary will be in bin/feed-cli
```

## Quick Start

```bash
# Add a feed
feed-cli add https://news.ycombinator.com/rss --category tech

# List all feeds
feed-cli feeds | jq '.'

# Update feeds (fetch new entries)
feed-cli update

# List latest entries
feed-cli list --limit 10 | jq '.entries[].title'

# List unread entries from last week
feed-cli list --unread --since 7d | jq '.'
```

## Usage

### Feed Management

```bash
# Add a feed
feed-cli add <url> [--category <category>]

# List all feeds
feed-cli feeds

# Update all feeds
feed-cli update

# Update specific feed
feed-cli update --feed-id 1

# Remove a feed
feed-cli remove <feed-id>
```

### Browsing Entries

```bash
# List entries with pagination
feed-cli list --limit 20 --offset 0

# Show only unread entries
feed-cli list --unread

# Filter by date (7 days, 2 weeks, 3 months, 1 year)
feed-cli list --since 7d
feed-cli list --since 2w
feed-cli list --since 3m
feed-cli list --since 1y

# Combine filters
feed-cli list --unread --since 2w --limit 50

# Show full entry details
feed-cli show <entry-id>
```

### Read Tracking

```bash
# Mark entries as read
feed-cli mark-read <entry-id> [<entry-id>...]

# Mark all entries as read
feed-cli mark-all-read
```

### Database Location

By default, the database is stored at `~/.config/feed-cli/feed-cli.db`.

Override with:
- `--db` flag: `feed-cli --db /path/to/db.db list`
- `FEED_CLI_DB` environment variable

## JSON Output & jq Examples

All commands output JSON by default, making feed-cli perfect for scripting.

### Pretty Print

```bash
# Pretty print all feeds
feed-cli feeds | jq '.'

# Pretty print entries
feed-cli list | jq '.'
```

### Extracting Data

```bash
# Get all feed titles
feed-cli feeds | jq -r '.[].title'

# Get all feed URLs
feed-cli feeds | jq -r '.[].url'

# Count unread entries
feed-cli list --unread | jq '.count'

# Extract entry URLs
feed-cli list --since 7d | jq -r '.entries[].link'

# Get titles of unread entries
feed-cli list --unread | jq -r '.entries[].title'

# Get entry titles and links as table
feed-cli list --limit 5 | jq -r '.entries[] | "\(.title)\t\(.link)"'
```

### Filtering with jq

```bash
# Find entries with "golang" in title
feed-cli list | jq '.entries[] | select(.title | test("golang"; "i"))'

# Get entries from a specific feed ID
feed-cli list | jq '.entries[] | select(.feed_id == 1)'

# Count entries per feed
feed-cli list | jq '[.entries[] | .feed_id] | group_by(.) | map({feed_id: .[0], count: length})'
```

### Opening in Browser

```bash
# Open first 5 unread entries in browser
feed-cli list --unread --limit 5 | jq -r '.entries[].link' | xargs -n1 open

# Open specific entry
feed-cli show 123 | jq -r '.link' | xargs open
```

### Scripting Examples

**Daily digest script:**
```bash
#!/bin/bash
# daily-digest.sh - Email yourself unread entries

UNREAD=$(feed-cli list --unread --since 1d | jq -r '.entries[] | "- \(.title): \(.link)"')

if [ -n "$UNREAD" ]; then
    echo "$UNREAD" | mail -s "Daily Feed Digest" you@example.com
    feed-cli mark-all-read
fi
```

**Feed stats:**
```bash
#!/bin/bash
# feed-stats.sh - Show feed statistics

echo "=== Feed Statistics ==="
echo "Total feeds: $(feed-cli feeds | jq 'length')"
echo "Total entries: $(feed-cli list --limit 999999 | jq '.count')"
echo "Unread entries: $(feed-cli list --unread | jq '.count')"
echo ""
echo "=== Top Feeds ==="
feed-cli list | jq -r '[.entries[] | .feed_id] | group_by(.) | map({feed_id: .[0], count: length}) | sort_by(.count) | reverse | .[:5][] | "\(.count) entries from feed \(.feed_id)"'
```

**Import from feeds.yaml:**
```bash
#!/bin/bash
# import-feeds.sh - Import feeds from YAML file

yq eval '.feeds[].url' feeds.yaml | while read url; do
    category=$(yq eval ".feeds[] | select(.url == \"$url\") | .category" feeds.yaml)
    feed-cli add "$url" --category "$category"
done
```

## Examples

### Workflow Example

```bash
# Morning routine: Check and read feeds

# 1. Update all feeds
feed-cli update

# 2. See what's new
feed-cli list --unread --since 1d | jq -r '.entries[] | "\(.title)"'

# 3. Open interesting articles
feed-cli list --unread --limit 5 | jq -r '.entries[].link' | xargs -n1 open

# 4. Mark as read
feed-cli mark-all-read
```

### Integration with Other Tools

```bash
# Export to CSV
feed-cli list | jq -r '.entries[] | [.id, .title, .link, .published] | @csv' > entries.csv

# Filter with fzf
feed-cli list | jq -r '.entries[] | "\(.id)\t\(.title)"' | fzf | cut -f1 | xargs feed-cli show

# Send to Slack webhook
feed-cli list --unread --limit 3 | jq '.entries[] | {text: .title, url: .link}' | \
  xargs -I{} curl -X POST -H 'Content-type: application/json' --data '{}' YOUR_WEBHOOK_URL
```

## Development

### Building

```bash
make build          # Build binary
make test           # Run all tests
make test-unit      # Run unit tests only
make test-coverage  # Generate coverage report
make lint           # Run linter
make clean          # Clean build artifacts
```

### Testing

feed-cli follows Test-Driven Development (TDD):

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./model -v
go test ./store -v
go test ./feed -v

# Run with coverage
go test -cover ./...
```

### Project Structure

```
feed-cli/
├── cmd/feed-cli/    # CLI entry point
├── model/           # Data structures
├── store/           # SQLite database operations
├── feed/            # RSS/Atom fetching
├── opml/            # OPML import/export (future)
└── testdata/        # Test fixtures
```

## Architecture

feed-cli uses a simple, flat package structure:

- **model/** - Core data types (Feed, Entry, Tag)
- **store/** - SQLite database layer with query builder
- **feed/** - RSS/Atom parser (wraps gofeed)
- **cmd/feed-cli/** - CLI commands and JSON output

No complex abstractions, just clean separation of concerns.

## Dependencies

- **gofeed** - RSS/Atom parsing
- **modernc.org/sqlite** - CGo-free SQLite
- **urfave/cli/v2** - CLI framework
- **testify** - Testing assertions

## Database Schema

Simple relational schema with proper indexes:

```sql
feeds
  ├─ id, url (unique), title, category
  └─ etag, last_modified (for HTTP caching)

entries
  ├─ id, feed_id (FK), guid, title, link
  ├─ content, published, is_read
  └─ UNIQUE(feed_id, guid) -- prevent duplicates

tags (future)
  └─ id, name

entry_tags (future)
  └─ entry_id (FK), tag_id (FK)
```

## Roadmap

- [x] Core feed management
- [x] Read/unread tracking
- [x] Pagination
- [x] Date filtering
- [x] JSON output
- [ ] OPML import/export
- [ ] Tag support
- [ ] Full-text search
- [ ] Web interface (optional)
- [ ] HTTP caching (ETags, Last-Modified)

## License

MIT License - see [LICENSE](LICENSE) file

## Contributing

Pull requests welcome! Please:

1. Write tests for new features
2. Follow existing code style
3. Update README if adding commands

## Credits

Built with:
- [gofeed](https://github.com/mmcdole/gofeed) by mmcdole
- [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) by cznic
- [urfave/cli](https://github.com/urfave/cli) by urfave

---

**feed-cli** - Simple, scriptable RSS for the command line.

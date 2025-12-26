package store

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// durationPattern matches duration strings like "7d", "2w", "3m", "1y"
var durationPattern = regexp.MustCompile(`^(\d+)([dwmy])$`)

// ParseDuration parses a duration string like "7d", "2w", "3m", "1y".
// Returns the duration or an error if the format is invalid.
//
// Supported units:
//   - d: days
//   - w: weeks (7 days)
//   - m: months (30 days, approximation)
//   - y: years (365 days, approximation)
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("duration string is empty")
	}

	matches := durationPattern.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid duration format: %s (expected format: <number><unit>, e.g., 7d, 2w, 3m, 1y)", s)
	}

	// Parse the number
	num, err := strconv.Atoi(matches[1])
	if err != nil || num < 0 {
		return 0, fmt.Errorf("invalid number in duration: %s", matches[1])
	}

	unit := matches[2]

	// Convert to time.Duration
	var duration time.Duration
	switch unit {
	case "d": // days
		duration = time.Duration(num) * 24 * time.Hour
	case "w": // weeks
		duration = time.Duration(num) * 7 * 24 * time.Hour
	case "m": // months (approximate as 30 days)
		duration = time.Duration(num) * 30 * 24 * time.Hour
	case "y": // years (approximate as 365 days)
		duration = time.Duration(num) * 365 * 24 * time.Hour
	default:
		return 0, fmt.Errorf("invalid duration unit: %s (expected d, w, m, or y)", unit)
	}

	return duration, nil
}

// SinceToUnixTime converts a "since" duration string (e.g., "7d") to a Unix timestamp.
// Returns the Unix timestamp representing the time point that is <duration> ago from now.
func SinceToUnixTime(since string) (int64, error) {
	duration, err := ParseDuration(since)
	if err != nil {
		return 0, err
	}

	sinceTime := time.Now().Add(-duration)
	return sinceTime.Unix(), nil
}

// BuildQueryOptions constructs QueryOptions from CLI flags.
func BuildQueryOptions(limit, offset int, unread bool, since, tag string) (QueryOptions, error) {
	opts := QueryOptions{
		Limit:      limit,
		Offset:     offset,
		UnreadOnly: unread,
		Tag:        tag,
	}

	// Parse since duration if provided
	if since != "" {
		sinceUnix, err := SinceToUnixTime(since)
		if err != nil {
			return opts, fmt.Errorf("failed to parse --since flag: %w", err)
		}
		opts.SinceTime = &sinceUnix
	}

	return opts, nil
}

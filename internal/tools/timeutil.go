package tools

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ParseTime accepts:
//   - "" -> zero (caller decides default)
//   - RFC3339 absolute, e.g. "2026-06-03T12:00:00Z"
//   - "now"
//   - "now-15m", "now-1h", "now-2h30m" (any time.ParseDuration suffix)
func ParseTime(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if s == "now" {
		return now, nil
	}
	if strings.HasPrefix(s, "now-") {
		d, err := time.ParseDuration(strings.TrimPrefix(s, "now-"))
		if err != nil {
			return time.Time{}, fmt.Errorf("parse relative time %q: %w", s, err)
		}
		return now.Add(-d), nil
	}
	if strings.HasPrefix(s, "now+") {
		d, err := time.ParseDuration(strings.TrimPrefix(s, "now+"))
		if err != nil {
			return time.Time{}, fmt.Errorf("parse relative time %q: %w", s, err)
		}
		return now.Add(d), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q (want RFC3339 or now[-Xs]): %w", s, err)
	}
	return t, nil
}

// Window parses a start/end pair with a sensible default of "last 15 minutes" if both empty.
func Window(start, end string) (*timestamppb.Timestamp, *timestamppb.Timestamp, error) {
	now := time.Now()
	s, err := ParseTime(start, now)
	if err != nil {
		return nil, nil, err
	}
	e, err := ParseTime(end, now)
	if err != nil {
		return nil, nil, err
	}
	if s.IsZero() && e.IsZero() {
		s = now.Add(-15 * time.Minute)
		e = now
	} else if s.IsZero() {
		s = e.Add(-15 * time.Minute)
	} else if e.IsZero() {
		e = now
	}
	return timestamppb.New(s), timestamppb.New(e), nil
}

package tools

import (
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		in   string
		want time.Time
		err  bool
	}{
		{"", time.Time{}, false},
		{"now", now, false},
		{"now-15m", now.Add(-15 * time.Minute), false},
		{"now-1h30m", now.Add(-90 * time.Minute), false},
		{"now+5m", now.Add(5 * time.Minute), false},
		{"2026-06-03T11:00:00Z", time.Date(2026, 6, 3, 11, 0, 0, 0, time.UTC), false},
		{"garbage", time.Time{}, true},
		{"now-bogus", time.Time{}, true},
	}
	for _, c := range cases {
		got, err := ParseTime(c.in, now)
		if (err != nil) != c.err {
			t.Errorf("ParseTime(%q) err=%v, wantErr=%v", c.in, err, c.err)
			continue
		}
		if !c.err && !got.Equal(c.want) {
			t.Errorf("ParseTime(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestWindowDefaults(t *testing.T) {
	s, e, err := Window("", "")
	if err != nil {
		t.Fatal(err)
	}
	if d := e.AsTime().Sub(s.AsTime()); d < 14*time.Minute || d > 16*time.Minute {
		t.Errorf("default window = %v, want ~15m", d)
	}
}

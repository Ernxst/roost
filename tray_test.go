package main

import (
	"testing"
	"time"

	"roost/internal/store"
)

func TestByline(t *testing.T) {
	for _, c := range []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{"ada"}, "From @ada"},
		{[]string{"ada", "grace"}, "From @ada and @grace"},
		{[]string{"ada", "grace", "alan"}, "From @ada, @grace and @alan"},
		{[]string{"ada", "grace", "alan", "kay"}, "From @ada, @grace, @alan and 1 other"},
		{[]string{"ada", "grace", "alan", "kay", "joan"}, "From @ada, @grace, @alan and 2 others"},
	} {
		if got := byline(c.in); got != c.want {
			t.Errorf("byline(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDueSync(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	recent := &store.Sync{FinishedAt: now.Add(-10 * time.Minute)}
	stale := &store.Sync{FinishedAt: now.Add(-2 * time.Hour)}
	for _, c := range []struct {
		name      string
		last      *store.Sync
		lastFull  time.Time
		every     int
		due, full bool
	}{
		{"off", stale, time.Time{}, 0, false, false},
		{"recent", recent, now.Add(-30 * 24 * time.Hour), 60, false, false},
		{"never synced", nil, time.Time{}, 60, true, false},
		{"never read in full", stale, time.Time{}, 60, true, false},
		{"read in full this week", stale, now.Add(-6 * 24 * time.Hour), 60, true, false},
		{"read in full a week ago", stale, now.Add(-7 * 24 * time.Hour), 60, true, true},
	} {
		due, full := dueSync(c.last, c.lastFull, c.every, now)
		if due != c.due || full != c.full {
			t.Errorf("%s: due %v, full %v; want %v, %v", c.name, due, full, c.due, c.full)
		}
	}
}

func TestSyncNotice(t *testing.T) {
	for _, c := range []struct {
		rec         store.Sync
		authors     []string
		title, body string
	}{
		{store.Sync{}, nil, "", ""},
		{store.Sync{Added: 2}, []string{"ada", "grace"}, "2 new bookmarks", "From @ada and @grace"},
		{store.Sync{Removed: 1}, nil, "1 bookmark gone from X", "roost kept its copy."},
		{store.Sync{Removed: 3}, nil, "3 bookmarks gone from X", "roost kept their copies."},
		{store.Sync{Added: 1, Removed: 2}, []string{"ada"}, "1 new bookmark", "From @ada\n2 bookmarks are gone from X; roost kept their copies."},
	} {
		title, body := syncNotice(c.rec, c.authors)
		if title != c.title || body != c.body {
			t.Errorf("syncNotice(%+v) = %q, %q; want %q, %q", c.rec, title, body, c.title, c.body)
		}
	}
}

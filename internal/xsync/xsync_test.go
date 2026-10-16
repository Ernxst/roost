package xsync

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"roost/internal/store"
)

// fakePage serves one timeline page per scroll, after an optional login.
type fakePage struct {
	pages      []string
	next       int
	loginTicks int
	shown      bool
	logins     int
}

func (p *fakePage) URL() (string, error) {
	if p.loginTicks > 0 {
		p.loginTicks--
		return "https://x.com/i/flow/login", nil
	}
	return "https://x.com/i/history", nil
}

func (p *fakePage) Drain() ([]string, error) {
	if p.next >= len(p.pages) {
		return nil, nil
	}
	p.next++
	return []string{p.pages[p.next-1]}, nil
}

func (p *fakePage) Scroll() error { return nil }
func (p *fakePage) ShowLogin()    { p.shown = true; p.logins++ }
func (p *fakePage) HideLogin()    { p.shown = false }

// timeline builds response bodies for posts with the given IDs, newest
// first, in pages of two, ending with an empty page.
func timeline(ids ...int) []string {
	var pages []string
	for i := 0; i < len(ids); i += 2 {
		var entries []string
		for _, id := range ids[i:min(i+2, len(ids))] {
			entries = append(entries, fmt.Sprintf(`{"entryId":"tweet-%d","sortIndex":"%d","content":{"itemContent":{"tweet_results":{"result":{
				"rest_id":"%d","core":{"user_results":{"result":{"core":{"screen_name":"ada","name":"Ada"}}}},
				"legacy":{"created_at":"Thu Oct 15 06:45:33 +0000 2026","full_text":"post %d"}}}}}}`, id, id, id, id))
		}
		pages = append(pages, `{"data":{"timeline":{"instructions":[{"entries":[`+strings.Join(entries, ",")+`]}]}}}`)
	}
	return append(pages, `{"data":{"timeline":{"instructions":[{"entries":[{"entryId":"cursor-bottom-0"}]}]}}}`)
}

func opts(full bool) Options {
	return Options{Full: full, Tick: time.Millisecond, Stall: 50 * time.Millisecond}
}

func open(t *testing.T) *store.Store {
	s, err := store.Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestFullThenIncremental(t *testing.T) {
	lib := open(t)
	page := &fakePage{pages: timeline(5, 4, 3, 2, 1), loginTicks: 3}
	rec, err := Run(context.Background(), page, lib, opts(true))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Added != 5 || !rec.ReachedEnd || page.logins != 1 || page.shown {
		t.Fatalf("first sync = %+v, logins %d, shown %v", rec, page.logins, page.shown)
	}

	// Two new bookmarks, then the ones already held: the incremental sync
	// stops on the page holding the first known one.
	page = &fakePage{pages: timeline(7, 6, 5, 4, 3, 2, 1)}
	rec, err = Run(context.Background(), page, lib, opts(false))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Added != 2 || rec.Seen != 4 || page.next != 2 {
		t.Fatalf("incremental = %+v after %d pages; want 2 added from 2 pages", rec, page.next)
	}

	// Post 3 was removed on X; a full sync marks it.
	page = &fakePage{pages: timeline(7, 6, 5, 4, 2, 1)}
	rec, err = Run(context.Background(), page, lib, opts(true))
	if err != nil || rec.Removed != 1 {
		t.Fatalf("full = %+v, %v; want 1 removed", rec, err)
	}
	st, _ := lib.Stats()
	if st.Total != 6 || st.Removed != 1 || st.LastSync == nil || st.LastSync.Removed != 1 {
		t.Fatalf("stats = %+v", st)
	}
}

func TestFullSyncThatStallsRemovesNothing(t *testing.T) {
	lib := open(t)
	Run(context.Background(), &fakePage{pages: timeline(3, 2, 1)}, lib, opts(true))

	pages := timeline(3, 2)
	page := &fakePage{pages: pages[:1]} // X stops answering before the end
	rec, err := Run(context.Background(), page, lib, opts(true))
	if err != nil || rec.ReachedEnd || rec.Removed != 0 {
		t.Fatalf("stalled full sync = %+v, %v; want no removals", rec, err)
	}
}

func TestBackgroundFullSyncRefusesLargeRemoval(t *testing.T) {
	lib := open(t)
	if _, err := Run(context.Background(), &fakePage{pages: timeline(10, 9, 8, 7, 6, 5, 4, 3, 2, 1)}, lib, opts(true)); err != nil {
		t.Fatal(err)
	}
	background := opts(true)
	background.Background = true
	rec, err := Run(context.Background(), &fakePage{pages: timeline(10, 9, 8, 7, 6, 5, 4)}, lib, background)
	if err == nil || !strings.Contains(err.Error(), "over 20%") || rec.Removed != 0 {
		t.Fatalf("background full sync = %+v, %v; want a refusal with no removals", rec, err)
	}
	stats, err := lib.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 10 || stats.Removed != 0 {
		t.Fatalf("stats after refused sync = %+v; want all ten bookmarks current", stats)
	}
}

func TestNoResponsesFails(t *testing.T) {
	_, err := Run(context.Background(), &fakePage{}, open(t), opts(false))
	if err == nil {
		t.Fatal("a sync that received nothing reported success")
	}
}

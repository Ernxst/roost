package store

import (
	"path/filepath"
	"testing"
	"time"

	"roost/internal/xparse"
)

func entry(id string, idx int64, text string, media bool) xparse.Entry {
	t := xparse.Tweet{ID: id, AuthorHandle: "ada", AuthorName: "Ada Lovelace", Text: text, CreatedAt: time.Unix(1, 0), Raw: []byte("{}")}
	if media {
		t.Media = []xparse.Media{{Kind: "photo", Thumb: "x.jpg"}}
	}
	return xparse.Entry{SortIndex: idx, Tweet: t}
}

func ids(t *testing.T, s *Store, q Query) []string {
	t.Helper()
	res, err := s.List(q)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, b := range res.Items {
		out = append(out, b.Tweet.ID)
	}
	if res.Total != len(out) {
		t.Fatalf("total %d but %d items", res.Total, len(out))
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSyncLifecycle(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	day1 := time.Unix(1000, 0)
	added, err := s.Ingest([]xparse.Entry{entry("1", 10, "gopher tunnels", false), entry("2", 20, "webgpu shaders", true)}, day1)
	if err != nil || added != 2 {
		t.Fatalf("first ingest added %d, err %v", added, err)
	}
	if got := ids(t, s, Query{}); !equal(got, []string{"2", "1"}) {
		t.Fatalf("order = %v, want newest bookmark first", got)
	}

	// A later sync sees post 3 as new and post 2 again; post 1 is gone from X.
	day2 := time.Unix(2000, 0)
	added, err = s.Ingest([]xparse.Entry{entry("3", 30, "rust lifetimes", false), entry("2", 20, "webgpu shaders", true)}, day2)
	if err != nil || added != 1 {
		t.Fatalf("second ingest added %d, err %v; want 1", added, err)
	}
	removed, err := s.MarkRemoved(day2, day2)
	if err != nil || removed != 1 {
		t.Fatalf("removed %d, err %v; want 1", removed, err)
	}
	if got := ids(t, s, Query{}); !equal(got, []string{"3", "2"}) {
		t.Fatalf("current = %v", got)
	}
	if got := ids(t, s, Query{Removed: true}); !equal(got, []string{"1"}) {
		t.Fatalf("removed = %v", got)
	}

	// Bookmarking post 1 again counts it as added and restores it.
	added, _ = s.Ingest([]xparse.Entry{entry("1", 40, "gopher tunnels", false)}, time.Unix(3000, 0))
	if added != 1 {
		t.Fatalf("re-bookmark added %d, want 1", added)
	}
	if got := ids(t, s, Query{}); !equal(got, []string{"1", "3", "2"}) {
		t.Fatalf("after re-bookmark = %v", got)
	}

	if got := ids(t, s, Query{Search: "shad"}); !equal(got, []string{"2"}) {
		t.Fatalf("prefix search = %v", got)
	}
	if got := ids(t, s, Query{Search: "lovelace rust"}); !equal(got, []string{"3"}) {
		t.Fatalf("author and text search = %v", got)
	}
	if got := ids(t, s, Query{MediaOnly: true}); !equal(got, []string{"2"}) {
		t.Fatalf("media only = %v", got)
	}
}

func TestCollections(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Unix(1000, 0)
	s.Ingest([]xparse.Entry{entry("1", 10, "a", false), entry("2", 20, "b", false)}, now)

	c, err := s.CreateCollection("  Reading ", 30, now)
	if err != nil || c.Name != "Reading" {
		t.Fatalf("create = %+v, %v", c, err)
	}
	if _, err := s.CreateCollection("reading", 40, now); err == nil {
		t.Fatal("duplicate name (ignoring case) was accepted")
	}
	if err := s.SetInCollection("1", c.ID, true, now); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, s, Query{Collection: c.ID}); !equal(got, []string{"1"}) {
		t.Fatalf("in collection = %v", got)
	}
	if got := ids(t, s, Query{Collection: Unsorted}); !equal(got, []string{"2"}) {
		t.Fatalf("unsorted = %v", got)
	}
	cols, _ := s.Collections()
	if len(cols) != 1 || cols[0].Count != 1 {
		t.Fatalf("collections = %+v", cols)
	}
	res, _ := s.List(Query{Collection: c.ID})
	if len(res.Items[0].Collections) != 1 || res.Items[0].Collections[0] != c.ID {
		t.Fatalf("bookmark collections = %v", res.Items[0].Collections)
	}

	if err := s.DeleteCollection(c.ID); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, s, Query{Collection: Unsorted}); !equal(got, []string{"2", "1"}) {
		t.Fatalf("unsorted after delete = %v", got)
	}
}

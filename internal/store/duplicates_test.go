package store

import (
	"path/filepath"
	"testing"
	"time"

	"roost/internal/xparse"
)

func TestNormaliseURL(t *testing.T) {
	for in, want := range map[string]string{
		"https://www.example.com/a/?utm_source=x&id=2#top": "example.com/a?id=2",
		"http://example.com/a":                              "example.com/a",
		"https://youtu.be/abc?si=xyz":                       "youtube.com/watch?v=abc",
		"https://example.com/":                              "example.com",
		"not a url":                                         "",
	} {
		if got := NormaliseURL(in); got != want {
			t.Errorf("NormaliseURL(%q) = %q, want %q", in, got, want)
		}
	}
	if id := statusID("https://twitter.com/ada/status/123?s=20"); id != "123" {
		t.Errorf("statusID = %q", id)
	}
}

func TestDuplicates(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := entry("1", 10, "a", true)
	b := entry("2", 11, "b", false)
	b.Tweet.Links = []xparse.Link{{Expanded: "https://x.com/ada/status/1"}, {Expanded: "https://example.com/p?utm_medium=x"}}
	c := entry("3", 12, "c", false)
	c.Tweet.Links = []xparse.Link{{Expanded: "https://www.example.com/p/"}}
	d := entry("4", 13, "d", true)
	s.Ingest([]xparse.Entry{a, b, c, d}, time.Unix(1, 0))

	groups, err := s.FindDuplicates()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, g := range groups {
		got[g.Reason+" "+g.Key] = len(g.Bookmarks)
	}
	want := map[string]int{"post 1": 2, "link example.com/p": 2, "image x.jpg": 2}
	if len(got) != len(want) {
		t.Fatalf("groups = %v", got)
	}
	for k, n := range want {
		if got[k] != n {
			t.Errorf("groups = %v, want %v", got, want)
		}
	}
	if wall, _ := s.Wall(); len(wall) != 2 {
		t.Errorf("wall = %d bookmarks", len(wall))
	}
}

package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"roost/internal/store"
	"roost/internal/xparse"
)

func TestWrite(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Unix(1000, 0)
	tw := xparse.Tweet{ID: "1", URL: "https://x.com/ada/status/1", AuthorHandle: "ada", AuthorName: "Ada", Text: "Engines &amp; looms: https://t.co/abc", CreatedAt: now,
		Links: []xparse.Link{{Short: "https://t.co/abc", Expanded: "https://example.com/looms", Display: "example.com/looms"}},
		Media: []xparse.Media{{Kind: "photo", Thumb: "https://pbs.twimg.com/a.jpg"}}, Raw: []byte("{}")}
	s.Ingest([]xparse.Entry{{SortIndex: 1, Tweet: tw}}, now)
	c, _ := s.CreateCollection("Reading", 10, "", now)
	s.SetInCollection("1", c.ID, true, now)
	s.SetNote("1", "for the essay")
	h, _ := s.AddHighlight("1", "", "Engines", now)
	s.SetHighlightNote(h.ID, "the hook")

	img := filepath.Join(t.TempDir(), "abc.jpg")
	os.WriteFile(img, []byte("jpeg"), 0o644)
	dir := t.TempDir()
	res, err := Write(dir, s, func(u string) (string, bool) { return img, u == "https://pbs.twimg.com/a.jpg?name=small" })
	if err != nil {
		t.Fatal(err)
	}
	if res.Notes != 2 || res.Media != 1 {
		t.Errorf("result = %+v", res)
	}
	note, err := os.ReadFile(filepath.Join(dir, "roost", "Bookmarks", "@ada – Engines & looms.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`author: "@ada"`, `- "[[Reading]]"`, "Engines & looms: https://example.com/looms", "![[abc.jpg]]", "## Note\n\nfor the essay", "> Engines\n\nthe hook\n", "## Links"} {
		if !strings.Contains(string(note), want) {
			t.Errorf("note lacks %q:\n%s", want, note)
		}
	}
	col, _ := os.ReadFile(filepath.Join(dir, "roost", "Collections", "Reading.md"))
	if !strings.Contains(string(col), "[[@ada – Engines & looms]]") {
		t.Errorf("collection note = %s", col)
	}
	if _, err := os.Stat(filepath.Join(dir, "roost", "attachments", "abc.jpg")); err != nil {
		t.Error(err)
	}
}

func TestCollection(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Unix(1000, 0)
	tw := xparse.Tweet{ID: "1", URL: "https://x.com/ada/status/1", AuthorHandle: "ada", AuthorName: "Ada", Text: "Engines &amp; looms", CreatedAt: now,
		Media: []xparse.Media{{Kind: "photo", Thumb: "https://pbs.twimg.com/a.jpg"}}, Raw: []byte("{}")}
	s.Ingest([]xparse.Entry{{SortIndex: 1, Tweet: tw}}, now)
	c, _ := s.CreateCollection("Reading", 10, "", now)
	s.SetInCollection("1", c.ID, true, now)
	s.SetNote("1", "for the essay")
	h, _ := s.AddHighlight("1", "", "Engines", now)
	s.SetHighlightNote(h.ID, "the hook")

	res, _ := s.List(store.Query{Collection: c.ID})
	var md strings.Builder
	if err := Collection(&md, c, res.Items, s.Highlights); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Reading\n\n1 post from X, kept in roost.", "**Ada** ([@ada](https://x.com/ada))", "Engines & looms", "![](https://pbs.twimg.com/a.jpg?name=small)", "*Note:* for the essay", "> Engines\n\nthe hook", "[View on X](https://x.com/ada/status/1)"} {
		if !strings.Contains(md.String(), want) {
			t.Errorf("collection lacks %q:\n%s", want, md.String())
		}
	}
}

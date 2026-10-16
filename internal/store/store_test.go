package store

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"roost/internal/links"
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

	c, err := s.CreateCollection("  Reading ", 30, "", now)
	if err != nil || c.Name != "Reading" {
		t.Fatalf("create = %+v, %v", c, err)
	}
	if _, err := s.CreateCollection("reading", 40, "", now); err == nil {
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

func TestSearchOperators(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Unix(1000, 0)
	day := func(y, m, d int) time.Time { return time.Date(y, time.Month(m), d, 12, 0, 0, 0, time.Local) }
	mk := func(id, handle, text string, at time.Time, media ...xparse.Media) xparse.Entry {
		idx, _ := strconv.ParseInt(id, 10, 64)
		return xparse.Entry{SortIndex: idx, Tweet: xparse.Tweet{ID: id, AuthorHandle: handle, AuthorName: handle, Text: text, CreatedAt: at, Media: media, Raw: []byte("{}")}}
	}
	looms := mk("1", "ada", "engines and looms", day(2024, 5, 1))
	looms.Tweet.Links = []xparse.Link{{Expanded: "https://www.GitHub.com/ada/looms"}}
	looms.Tweet.Raw = []byte(`{"legacy":{"lang":"en","favorite_count":1200}}`)
	video := mk("2", "bob", "a video of looms", day(2025, 2, 1), xparse.Media{Kind: "video"})
	video.Tweet.Links = []xparse.Link{{Expanded: "https://gist.github.com/bob/1"}, {Expanded: "https://notgithub.com/"}}
	video.Tweet.Raw = []byte(`{"legacy":{"lang":"fr","favorite_count":40}}`)
	s.Ingest([]xparse.Entry{
		looms,
		video,
		mk("3", "ada", "photos of engines", day(2025, 6, 1), xparse.Media{Kind: "photo"}),
	}, now)
	reading, _ := s.CreateCollection("Reading list", 10, "", now)
	s.SetInCollection("1", reading.ID, true, now)
	s.SetNote("3", "check this")

	cases := map[string][]string{
		"from:@ada":                          {"3", "1"},
		"looms -video":                       {"1"},
		"has:video":                          {"2"},
		"has:media -has:video":               {"3"},
		"has:note":                           {"3"},
		"check":                              {"3"},
		`in:"reading list"`:                  {"1"},
		"in:none":                            {"3", "2"},
		"-in:any engines":                    {"3"},
		"before:2025-01-01":                  {"1"},
		"after:2025-01-31 before:2025-06-30": {"3", "2"},
		"has:nonsense":                       {},
		"domain:github.com":                  {"2", "1"},
		"domain:https://gist.github.com/x":   {"2"},
		"-domain:gist.github.com":            {"3", "1"},
		"domain:hub.com":                     {},
		"lang:EN":                            {"1"},
		"likes:1000":                         {"1"},
		"likes:0":                            {"3", "2", "1"},
		"likes:many":                         {},
	}
	for q, want := range cases {
		if got := ids(t, s, Query{Search: q}); !equal(got, want) {
			t.Errorf("search %q = %v, want %v", q, got, want)
		}
	}

	smart, err := s.CreateCollection("Ada's engines", 20, "from:ada engines", now)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(t, s, Query{Collection: smart.ID}); !equal(got, []string{"3", "1"}) {
		t.Errorf("smart collection = %v", got)
	}
	if got := ids(t, s, Query{Collection: smart.ID, Search: "photos"}); !equal(got, []string{"3"}) {
		t.Errorf("search within smart collection = %v", got)
	}
	if err := s.SetInCollection("2", smart.ID, true, now); err == nil {
		t.Error("filed a post into a smart collection")
	}
	cols, _ := s.Collections()
	if len(cols) != 2 || cols[1].ID != smart.ID || cols[1].Count != 2 {
		t.Errorf("collections = %+v, want the smart one last with 2 posts", cols)
	}
}

func TestOpenedAsideAndRevisit(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.Local)
	old := entry("1", 10, "from a year ago", false)
	old.Tweet.CreatedAt = time.Date(2025, 10, 16, 20, 0, 0, 0, time.Local)
	s.Ingest([]xparse.Entry{old, entry("2", 20, "b", false), entry("3", 30, "c", false)}, now)

	s.MarkOpened("2", now)
	s.SetAside("3", true, now)
	if got := ids(t, s, Query{Search: "is:unopened"}); !equal(got, []string{"3", "1"}) {
		t.Errorf("is:unopened = %v", got)
	}
	if got := ids(t, s, Query{Collection: Unsorted, Search: "-is:aside"}); !equal(got, []string{"2", "1"}) {
		t.Errorf("sorting queue = %v", got)
	}
	s.SetAside("3", false, now)
	if got := ids(t, s, Query{Search: "is:aside"}); len(got) != 0 {
		t.Errorf("is:aside after putting back = %v", got)
	}

	r, err := s.Revisit(now, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.OnThisDay) != 1 || r.OnThisDay[0].Tweet.ID != "1" {
		t.Errorf("on this day = %+v", r.OnThisDay)
	}
	if len(r.Unopened) != 2 || len(r.Random) != 3 {
		t.Errorf("unopened %d, random %d; want 2 and 3", len(r.Unopened), len(r.Random))
	}
}

func TestSettingsAndMediaURLs(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if v, _ := s.Settings(); v.SyncEvery != 60 || !v.Notify {
		t.Errorf("default settings = %+v", v)
	}
	if err := s.SaveSettings(Settings{CacheVideos: true}); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.Settings(); !v.CacheVideos || v.SyncEvery != 0 {
		t.Errorf("saved settings = %+v", v)
	}

	e := entry("1", 10, "a", false)
	e.Tweet.AuthorAvatar = "https://pbs.twimg.com/a.jpg"
	e.Tweet.Media = []xparse.Media{{Kind: "video", Thumb: "https://pbs.twimg.com/v.jpg", Video: "https://video.twimg.com/v.mp4"}}
	s.Ingest([]xparse.Entry{e}, time.Unix(1, 0))
	urls, err := s.MediaURLs(false)
	if err != nil || len(urls) != 3 {
		t.Fatalf("urls = %v, %v", urls, err)
	}
	if urls, _ := s.MediaURLs(true); len(urls) != 4 {
		t.Errorf("with videos = %v", urls)
	}

	s.Ingest([]xparse.Entry{entry("2", 11, "b", false), entry("3", 12, "b", false)}, time.Unix(5, 0))
	if got, _ := s.NewAuthors(time.Unix(5, 0)); len(got) != 1 || got[0] != "ada" {
		t.Errorf("new authors = %v", got)
	}
	if got, _ := s.NewAuthors(time.Unix(6, 0)); len(got) != 0 {
		t.Errorf("new authors later = %v", got)
	}
}

func TestThreads(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Unix(1000, 0)
	s.Ingest([]xparse.Entry{
		entry("1", 10, "how shaders work 🧵", false),
		entry("2", 20, "a lone post", false),
	}, now)
	cands, err := s.ThreadCandidates(10)
	if err != nil || len(cands) != 1 || cands[0].ID != "1" {
		t.Fatalf("candidates = %+v, %v", cands, err)
	}
	posts := []xparse.Tweet{{ID: "1", Text: "how shaders work 🧵"}, {ID: "9", Text: "first, the rasteriser"}}
	if err := s.SaveThread("1", posts, now); err != nil {
		t.Fatal(err)
	}
	if cands, _ := s.ThreadCandidates(10); len(cands) != 0 {
		t.Errorf("checked post is still a candidate: %+v", cands)
	}
	if got := ids(t, s, Query{Search: "rasteriser"}); !equal(got, []string{"1"}) {
		t.Errorf("search in thread = %v", got)
	}
	if got := ids(t, s, Query{Search: "has:thread"}); !equal(got, []string{"1"}) {
		t.Errorf("has:thread = %v", got)
	}
	res, _ := s.List(Query{Search: "shaders"})
	if res.Items[0].ThreadLength != 2 {
		t.Errorf("thread length = %d", res.Items[0].ThreadLength)
	}
	if got, _ := s.Thread("1"); len(got) != 2 {
		t.Errorf("thread = %+v", got)
	}
}

func TestLinks(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Unix(1000, 0)
	e := entry("1", 10, "worth reading", false)
	e.Tweet.Links = []xparse.Link{{Short: "https://t.co/a", Expanded: "https://blog.example.com/shaders"}, {Short: "https://t.co/b", Expanded: "https://x.com/a/status/2"}}
	s.Ingest([]xparse.Entry{e, entry("2", 20, "no links", false)}, now)

	cands, err := s.LinkCandidates(10)
	if err != nil || len(cands) != 1 || cands[0] != "https://blog.example.com/shaders" {
		t.Fatalf("candidates = %v, %v", cands, err)
	}
	if err := s.SaveLink(links.Page{URL: cands[0], Title: "How shaders work", Text: "the rasteriser fills triangles"}, now); err != nil {
		t.Fatal(err)
	}
	if cands, _ := s.LinkCandidates(10); len(cands) != 0 {
		t.Errorf("saved link still a candidate: %v", cands)
	}
	if got := ids(t, s, Query{Search: "triangles"}); !equal(got, []string{"1"}) {
		t.Errorf("search in linked page = %v", got)
	}
	if got := ids(t, s, Query{Search: "has:article"}); !equal(got, []string{"1"}) {
		t.Errorf("has:article = %v", got)
	}
	ls, err := s.Links("1")
	if err != nil || len(ls) != 2 || !ls[0].HasText || ls[0].Page.Title != "How shaders work" || ls[0].Page.Text != "" || ls[1].Page != nil {
		t.Errorf("links = %+v, %v", ls, err)
	}
}

func TestHighlights(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Unix(1000, 0)
	s.Ingest([]xparse.Entry{entry("1", 10, "shaders run per pixel", false), entry("2", 20, "b", false)}, now)
	if _, err := s.AddHighlight("1", "", "   ", now); err == nil {
		t.Error("empty highlight accepted")
	}
	h, err := s.AddHighlight("1", "", " run per pixel ", now)
	if err != nil || h.Quote != "run per pixel" {
		t.Fatalf("add = %+v, %v", h, err)
	}
	s.AddHighlight("1", "https://blog.example.com", "triangles", now.Add(time.Second))
	if got := ids(t, s, Query{Search: "has:highlight"}); !equal(got, []string{"1"}) {
		t.Errorf("has:highlight = %v", got)
	}
	all, _ := s.AllHighlights()
	if len(all) != 2 || all[0].Quote != "triangles" || all[0].Tweet == nil || all[0].Tweet.ID != "1" {
		t.Errorf("all = %+v", all)
	}
	if got := ids(t, s, Query{Search: "triang"}); !equal(got, []string{"1"}) {
		t.Errorf("search for a highlighted word = %v", got)
	}
	s.DeleteHighlight(h.ID)
	if hs, _ := s.Highlights("1"); len(hs) != 1 {
		t.Errorf("after delete = %+v", hs)
	}
	if err := s.SetHighlightNote(all[0].ID, " compare with rasterising "); err != nil {
		t.Fatal(err)
	}
	if hs, _ := s.Highlights("1"); len(hs) != 1 || hs[0].Note != "compare with rasterising" {
		t.Errorf("highlights = %+v", hs)
	}
	if got := ids(t, s, Query{Search: "rasteris"}); !equal(got, []string{"1"}) {
		t.Errorf("search for a word in a highlight's note = %v", got)
	}
	s.DeleteHighlight(all[0].ID)
	if got := ids(t, s, Query{Search: "triangles"}); len(got) != 0 {
		t.Errorf("search still finds a deleted highlight: %v", got)
	}
}

func TestSuggest(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Unix(1000, 0)
	s.Ingest([]xparse.Entry{
		entry("1", 10, "webgpu compute shaders on the gpu", false),
		entry("2", 20, "writing fragment shaders for the gpu", false),
		entry("3", 30, "sourdough bread hydration", false),
		entry("4", 40, "a new gpu shaders tutorial", false),
	}, now)
	gpu, _ := s.CreateCollection("GPU", 10, "", now)
	baking, _ := s.CreateCollection("Baking", 20, "", now)
	s.SetInCollection("1", gpu.ID, true, now)
	s.SetInCollection("2", gpu.ID, true, now)
	s.SetInCollection("3", baking.ID, true, now)

	got, err := s.Suggest("4", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[0].CollectionID != gpu.ID {
		t.Fatalf("suggestions = %+v, want GPU first", got)
	}
	if got, _ := s.Suggest("1", 3); len(got) != 1 || got[0].CollectionID != baking.ID {
		// Post 1 is by the same author as the baking post, so Baking comes
		// up on author affinity alone; GPU is left out because post 1 is in it.
		t.Errorf("suggestions for a filed post = %+v", got)
	}
}

func TestTranslationsAreParsedAgainAndSearchable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lib.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// A post saved by a parser that ignored translations: its data lacks
	// one, but the raw object X sent has it.
	e := entry("1", 10, "見て", false)
	e.Tweet.Raw = []byte(`{"rest_id":"1","core":{"user_results":{"result":{"core":{"screen_name":"ada","name":"Ada Lovelace"}}}},
		"legacy":{"created_at":"Mon Jun 28 04:13:47 +0000 2021","full_text":"見て"},
		"grok_translated_post_with_availability":{"is_available":true,"data":{"source_language":"ja","destination_language":"en","translation":"Look"}}}`)
	if _, err := s.Ingest([]xparse.Entry{e}, time.Unix(1, 0)); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, s, Query{Search: "look"}); len(got) != 0 {
		t.Fatalf("found %v by a translation the post did not have yet", got)
	}
	if _, err := s.db.Exec(`DELETE FROM settings WHERE key = 'parser_version'`); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	res, err := s.List(Query{Search: "look"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 || res.Items[0].Tweet.Translation == nil || res.Items[0].Tweet.Translation.From != "ja" {
		t.Fatalf("reopened library = %+v, want the post found by its translation", res.Items)
	}
}

func TestLastFullSync(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if at, err := s.LastFullSync(); err != nil || !at.IsZero() {
		t.Fatalf("empty library: %v, %v", at, err)
	}
	s.RecordSync(Sync{Full: true, FinishedAt: time.Unix(100, 0), ReachedEnd: true})
	s.RecordSync(Sync{Full: false, FinishedAt: time.Unix(200, 0), ReachedEnd: true})
	s.RecordSync(Sync{Full: true, FinishedAt: time.Unix(300, 0)})
	s.RecordSync(Sync{Full: true, FinishedAt: time.Unix(400, 0), ReachedEnd: true, Error: "signed out"})
	if at, _ := s.LastFullSync(); !at.Equal(time.Unix(100, 0)) {
		t.Errorf("last full sync = %v, want the only complete one", at)
	}
}

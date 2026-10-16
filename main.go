package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"roost/internal/links"
	"roost/internal/media"
	"roost/internal/store"
	"roost/internal/xparse"
	"roost/internal/xsync"

	"github.com/egoist/mygo"
)

// SyncProgress reports each stage of a sync to the page.
var SyncProgress = mygo.NewEvent[xsync.Progress]("syncProgress")

// ThreadsFound reports how many threads a background pass found, so the page
// can show them.
var ThreadsFound = mygo.NewEvent[int]("threadsFound")

// Library is the bookmark library as the frontend sees it.
type Library struct {
	mu      sync.Mutex
	lib     *store.Store
	main    *mygo.Window
	syncing bool
	media   *media.Cache
	caching sync.Mutex
	// threading serialises thread capture, which shares one hidden page.
	threading sync.Mutex
	linking   sync.Mutex
}

func (l *Library) store() (*store.Store, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.lib == nil {
		return nil, errors.New("the library is still opening")
	}
	return l.lib, nil
}

// List returns the bookmarks matching the query, newest bookmark first.
func (l *Library) List(q store.Query) (store.ListResult, error) {
	s, err := l.store()
	if err != nil {
		return store.ListResult{}, err
	}
	return s.List(q)
}

// Stats counts bookmarks and describes the last sync.
func (l *Library) Stats() (store.Stats, error) {
	s, err := l.store()
	if err != nil {
		return store.Stats{}, err
	}
	return s.Stats()
}

// Collections lists the user's collections with their bookmark counts.
func (l *Library) Collections() ([]store.Collection, error) {
	s, err := l.store()
	if err != nil {
		return nil, err
	}
	return s.Collections()
}

// Authors lists the authors the user has bookmarked most.
func (l *Library) Authors(limit int) ([]store.Author, error) {
	s, err := l.store()
	if err != nil {
		return nil, err
	}
	return s.Authors(limit)
}

// CreateCollection makes a collection; a non-empty query makes it a smart
// collection that holds whatever matches that search.
func (l *Library) CreateCollection(name string, hue int, query string) (store.Collection, error) {
	s, err := l.store()
	if err != nil {
		return store.Collection{}, err
	}
	return s.CreateCollection(name, hue, query, time.Now())
}

func (l *Library) UpdateCollection(id int64, name string, hue int, query string) error {
	s, err := l.store()
	if err != nil {
		return err
	}
	return s.UpdateCollection(id, name, hue, query)
}

func (l *Library) DeleteCollection(id int64) error {
	s, err := l.store()
	if err != nil {
		return err
	}
	return s.DeleteCollection(id)
}

// SetInCollection adds a bookmark to a collection or takes it out.
func (l *Library) SetInCollection(tweetID string, collectionID int64, in bool) error {
	s, err := l.store()
	if err != nil {
		return err
	}
	return s.SetInCollection(tweetID, collectionID, in, time.Now())
}

func (l *Library) SetNote(tweetID, note string) error {
	s, err := l.store()
	if err != nil {
		return err
	}
	return s.SetNote(tweetID, note)
}

// MarkOpened records that the user opened a bookmark in roost.
func (l *Library) MarkOpened(tweetID string) error {
	s, err := l.store()
	if err != nil {
		return err
	}
	return s.MarkOpened(tweetID, time.Now())
}

// SetAside takes a bookmark out of the sorting queue without filing it, or
// puts it back.
func (l *Library) SetAside(tweetID string, aside bool) error {
	s, err := l.store()
	if err != nil {
		return err
	}
	return s.SetAside(tweetID, aside, time.Now())
}

// Revisit picks bookmarks worth seeing again: posts from this day in
// earlier years, never-opened ones and a random handful.
func (l *Library) Revisit(n int) (store.Revisit, error) {
	s, err := l.store()
	if err != nil {
		return store.Revisit{}, err
	}
	return s.Revisit(time.Now(), n)
}

// Settings returns the user's preferences.
func (l *Library) Settings() (store.Settings, error) {
	s, err := l.store()
	if err != nil {
		return store.Settings{}, err
	}
	return s.Settings()
}

func (l *Library) SaveSettings(v store.Settings) error {
	s, err := l.store()
	if err != nil {
		return err
	}
	if err := s.SaveSettings(v); err != nil {
		return err
	}
	go l.cacheMedia()
	return nil
}

// MediaUsage is how much media is kept on disk.
type MediaUsage struct {
	Files int   `json:"files"`
	Bytes int64 `json:"bytes"`
}

func (l *Library) MediaUsage() (MediaUsage, error) {
	files, bytes, err := l.media.Usage()
	return MediaUsage{files, bytes}, err
}

// cacheMedia downloads any media in the library not yet on disk. One run
// happens at a time; a call during a run waits, then fetches whatever is
// still missing.
func (l *Library) cacheMedia() {
	l.caching.Lock()
	defer l.caching.Unlock()
	s, err := l.store()
	if err != nil {
		return
	}
	settings, err := s.Settings()
	if err != nil {
		log.Printf("media: %v", err)
		return
	}
	urls, err := s.MediaURLs(settings.CacheVideos)
	if err != nil {
		log.Printf("media: %v", err)
		return
	}
	if n := l.media.Prefetch(context.Background(), urls); n > 0 {
		log.Printf("media: kept %d new files", n)
	}
}

// Thread returns the posts in a bookmark's thread, in reading order, or
// none if it stands alone or has not been checked.
func (l *Library) Thread(tweetID string) ([]xparse.Tweet, error) {
	s, err := l.store()
	if err != nil {
		return nil, err
	}
	return s.Thread(tweetID)
}

// LoadThread reads a bookmark's thread from X now and returns it.
func (l *Library) LoadThread(tweetID, url string) ([]xparse.Tweet, error) {
	s, err := l.store()
	if err != nil {
		return nil, err
	}
	l.threading.Lock()
	defer l.threading.Unlock()
	page := openSyncPage(l.main, "about:blank")
	defer page.Close()
	posts, err := xsync.Thread(context.Background(), page, store.ThreadCandidate{ID: tweetID, URL: url}, 0)
	if err != nil {
		return nil, err
	}
	if err := s.SaveThread(tweetID, posts, time.Now()); err != nil {
		return nil, err
	}
	if posts == nil {
		posts = []xparse.Tweet{}
	}
	return posts, nil
}

// captureThreads checks bookmarks that look like part of a thread and keeps
// the threads they belong to. It works through a batch at a gentle pace and
// stops if X is signed out.
func (l *Library) captureThreads() {
	s, err := l.store()
	if err != nil {
		return
	}
	l.threading.Lock()
	defer l.threading.Unlock()
	cands, err := s.ThreadCandidates(200)
	if err != nil || len(cands) == 0 {
		return
	}
	page := openSyncPage(l.main, "about:blank")
	defer page.Close()
	found := 0
	for _, c := range cands {
		posts, err := xsync.Thread(context.Background(), page, c, 0)
		if errors.Is(err, xsync.ErrSignedOut) {
			log.Printf("threads: X is signed out; stopping")
			break
		}
		if err != nil {
			log.Printf("threads: %s: %v", c.ID, err)
			continue
		}
		if err := s.SaveThread(c.ID, posts, time.Now()); err != nil {
			log.Printf("threads: %v", err)
			return
		}
		if posts != nil {
			found++
		}
		time.Sleep(1500 * time.Millisecond)
	}
	log.Printf("threads: checked %d posts, %d threads", len(cands), found)
	if found > 0 {
		ThreadsFound.Broadcast(found)
		go l.cacheMedia()
	}
}

// Links returns the links in a post and what roost saved of each page.
func (l *Library) Links(tweetID string) ([]store.SavedLink, error) {
	s, err := l.store()
	if err != nil {
		return nil, err
	}
	return s.Links(tweetID)
}

// Page returns the saved copy of a linked page, text included.
func (l *Library) Page(url string) (*links.Page, error) {
	s, err := l.store()
	if err != nil {
		return nil, err
	}
	return s.Page(url)
}

// saveLinks fetches the pages bookmarks link to and keeps their text. One
// run happens at a time.
func (l *Library) saveLinks() {
	s, err := l.store()
	if err != nil {
		return
	}
	l.linking.Lock()
	defer l.linking.Unlock()
	urls, err := s.LinkCandidates(500)
	if err != nil || len(urls) == 0 {
		return
	}
	jobs := make(chan string)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range jobs {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				page := links.Fetch(ctx, u)
				cancel()
				if err := s.SaveLink(page, time.Now()); err != nil {
					log.Printf("links: %v", err)
				}
			}
		}()
	}
	for _, u := range urls {
		jobs <- u
	}
	close(jobs)
	wg.Wait()
	log.Printf("links: saved %d pages", len(urls))
}

// AddHighlight keeps a passage from a post, or from a page it links to when
// source is that page's URL.
func (l *Library) AddHighlight(tweetID, source, quote string) (store.Highlight, error) {
	s, err := l.store()
	if err != nil {
		return store.Highlight{}, err
	}
	return s.AddHighlight(tweetID, source, quote, time.Now())
}

func (l *Library) DeleteHighlight(id int64) error {
	s, err := l.store()
	if err != nil {
		return err
	}
	return s.DeleteHighlight(id)
}

func (l *Library) Highlights(tweetID string) ([]store.Highlight, error) {
	s, err := l.store()
	if err != nil {
		return nil, err
	}
	return s.Highlights(tweetID)
}

// Suggest ranks the collections a post might belong in.
func (l *Library) Suggest(tweetID string) ([]store.Suggestion, error) {
	s, err := l.store()
	if err != nil {
		return nil, err
	}
	return s.Suggest(tweetID, 3)
}

// Get returns one bookmark.
func (l *Library) Get(tweetID string) (store.Bookmark, error) {
	s, err := l.store()
	if err != nil {
		return store.Bookmark{}, err
	}
	return s.Get(tweetID)
}

// AllHighlights lists every highlight with its post, newest first.
func (l *Library) AllHighlights() ([]store.Highlight, error) {
	s, err := l.store()
	if err != nil {
		return nil, err
	}
	return s.AllHighlights()
}

// Sync starts bringing the library up to date with X and returns at once;
// SyncProgress reports how it goes. A full sync reads every bookmark and
// marks the ones gone from X as removed.
func (l *Library) Sync(full bool) error {
	s, err := l.store()
	if err != nil {
		return err
	}
	l.mu.Lock()
	if l.syncing {
		l.mu.Unlock()
		return errors.New("a sync is already running")
	}
	l.syncing = true
	l.mu.Unlock()

	go func() {
		defer func() {
			l.mu.Lock()
			l.syncing = false
			l.mu.Unlock()
		}()
		page := openSyncPage(l.main, xsync.BookmarksURL)
		defer page.Close()
		_, err := xsync.Run(context.Background(), page, s, xsync.Options{
			Full: full,
			Progress: func(p xsync.Progress) {
				log.Printf("sync: %s, %d seen, %d new, %d removed %s", p.Phase, p.Seen, p.Added, p.Removed, p.Message)
				SyncProgress.Broadcast(p)
			},
		})
		if err != nil {
			log.Printf("sync: %v", err)
		}
		go l.captureThreads()
		go l.saveLinks()
		l.cacheMedia()
	}()
	return nil
}

// Import adds bookmarks from a file of X's raw Bookmarks responses, a JSON
// array of response bodies, and returns how many were new.
func (l *Library) Import() (int, error) {
	s, err := l.store()
	if err != nil {
		return 0, err
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:  l.main,
		Title:   "Import bookmarks",
		Message: "Choose a file of raw X Bookmarks responses (raw-graphql-responses.json).",
		Filters: []mygo.FileFilter{{Name: "JSON", Extensions: []string{"json"}}},
	})
	if err != nil || len(paths) == 0 {
		return 0, err
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		return 0, err
	}
	var bodies []json.RawMessage
	if err := json.Unmarshal(data, &bodies); err != nil {
		return 0, fmt.Errorf("%s is not a JSON array of responses: %w", filepath.Base(paths[0]), err)
	}
	now := time.Now()
	added := 0
	for _, body := range bodies {
		entries, err := xparse.Parse(body)
		if err != nil {
			return added, err
		}
		n, err := s.Ingest(entries, now)
		if err != nil {
			return added, err
		}
		added += n
	}
	return added, nil
}

// Open opens a URL in the default browser.
func (l *Library) Open(url string) error { return mygo.Shell.OpenExternal(url) }

func main() {
	library := &Library{}
	mygo.Bind(library)

	mygo.App.WhenReady(func() {
		dir, err := mygo.App.Path(mygo.PathUserData)
		if err == nil {
			err = os.MkdirAll(dir, 0o755)
		}
		var lib *store.Store
		if err == nil {
			lib, err = store.Open(filepath.Join(dir, "library.db"))
		}
		if err != nil {
			log.Fatalf("open library: %v", err)
		}
		library.mu.Lock()
		library.lib = lib
		library.mu.Unlock()
		library.media = media.New(filepath.Join(dir, "media"))
		if err := mygo.Protocol.Handle(media.Scheme, library.media); err != nil {
			log.Fatalf("media: %v", err)
		}
		go library.cacheMedia()
		go library.captureThreads()
		go library.saveLinks()
		library.main = mygo.NewWindow(mygo.WindowOptions{
			Title:           "roost",
			Width:           1240,
			Height:          820,
			MinWidth:        760,
			MinHeight:       520,
			TitleBarStyle:   mygo.TitleBarHiddenInset,
			BackgroundColor: "light-dark(#efe9dc, #15130f)", // --paper in src/style.css
			StateKey:        "main",
			URL:             "/",
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

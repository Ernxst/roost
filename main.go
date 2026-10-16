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

	"roost/internal/store"
	"roost/internal/xparse"
	"roost/internal/xsync"

	"github.com/egoist/mygo"
)

// SyncProgress reports each stage of a sync to the page.
var SyncProgress = mygo.NewEvent[xsync.Progress]("syncProgress")

// Library is the bookmark library as the frontend sees it.
type Library struct {
	mu      sync.Mutex
	lib     *store.Store
	main    *mygo.Window
	syncing bool
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

func (l *Library) CreateCollection(name string, hue int) (store.Collection, error) {
	s, err := l.store()
	if err != nil {
		return store.Collection{}, err
	}
	return s.CreateCollection(name, hue, time.Now())
}

func (l *Library) UpdateCollection(id int64, name string, hue int) error {
	s, err := l.store()
	if err != nil {
		return err
	}
	return s.UpdateCollection(id, name, hue)
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
		page := openSyncPage(l.main)
		defer page.Close()
		_, err := xsync.Run(context.Background(), page, s, xsync.Options{
			Full:     full,
			Progress: func(p xsync.Progress) { SyncProgress.Broadcast(p) },
		})
		if err != nil {
			log.Printf("sync: %v", err)
		}
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

// Package media keeps copies of the images and videos in bookmarks on disk,
// so the library still shows them offline and after X deletes the post.
package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Scheme is the URL scheme the page loads cached media from.
const Scheme = "roost-media"

// Cache stores files fetched from X's media hosts under a directory, named
// by a hash of their URL.
type Cache struct {
	dir    string
	client *http.Client

	mu       sync.Mutex
	inflight map[string]*fetch
}

type fetch struct {
	done chan struct{}
	err  error
}

func New(dir string) *Cache {
	return &Cache{
		dir:      dir,
		client:   &http.Client{Timeout: 2 * time.Minute},
		inflight: map[string]*fetch{},
	}
}

// Allowed reports whether u is on one of X's media hosts. The cache fetches
// nothing else, so a page cannot use it to reach arbitrary URLs.
func Allowed(u string) bool {
	p, err := url.Parse(u)
	if err != nil || p.Scheme != "https" {
		return false
	}
	switch p.Hostname() {
	case "pbs.twimg.com", "video.twimg.com", "abs.twimg.com":
		return true
	}
	return false
}

func (c *Cache) path(u string) string {
	sum := sha256.Sum256([]byte(u))
	name := hex.EncodeToString(sum[:])
	ext := path.Ext(strings.SplitN(u, "?", 2)[0])
	if len(ext) > 5 {
		ext = ""
	}
	return filepath.Join(c.dir, name[:2], name+ext)
}

// Has reports whether u is already on disk.
func (c *Cache) Has(u string) bool {
	_, err := os.Stat(c.path(u))
	return err == nil
}

// Fetch makes sure u is on disk and returns its path. Concurrent calls for
// the same URL share one download.
func (c *Cache) Fetch(ctx context.Context, u string) (string, error) {
	if !Allowed(u) {
		return "", fmt.Errorf("media: %s is not an X media URL", u)
	}
	p := c.path(u)
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	c.mu.Lock()
	f, ok := c.inflight[u]
	if !ok {
		f = &fetch{done: make(chan struct{})}
		c.inflight[u] = f
		go func() {
			f.err = c.download(u, p)
			c.mu.Lock()
			delete(c.inflight, u)
			c.mu.Unlock()
			close(f.done)
		}()
	}
	c.mu.Unlock()
	select {
	case <-f.done:
		return p, f.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *Cache) download(u, p string) error {
	resp, err := c.client.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("media: %s: %s", u, resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".part-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

// ServeHTTP serves roost-media://localhost/?u=<original URL> from disk,
// fetching it first if needed. Range requests work, so videos can seek.
func (c *Cache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Query().Get("u")
	p, err := c.Fetch(r.Context(), u)
	if err != nil {
		code := http.StatusBadGateway
		if !Allowed(u) {
			code = http.StatusForbidden
		}
		http.Error(w, err.Error(), code)
		return
	}
	w.Header().Set("Cache-Control", "max-age=31536000, immutable")
	http.ServeFile(w, r, p)
}

// Prefetch downloads every URL not yet on disk, a few at a time, and
// returns how many it fetched. Failures are skipped; a later run retries.
func (c *Cache) Prefetch(ctx context.Context, urls []string) int {
	jobs := make(chan string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	fetched := 0
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range jobs {
				if _, err := c.Fetch(ctx, u); err == nil {
					mu.Lock()
					fetched++
					mu.Unlock()
				}
			}
		}()
	}
	for _, u := range urls {
		if !Allowed(u) || c.Has(u) {
			continue
		}
		select {
		case jobs <- u:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}
	}
	close(jobs)
	wg.Wait()
	return fetched
}

// Usage counts the files on disk and their total size.
func (c *Cache) Usage() (files int, bytes int64, err error) {
	err = filepath.WalkDir(c.dir, func(_ string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			return fs.SkipAll
		}
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files++
		bytes += info.Size()
		return nil
	})
	return files, bytes, err
}

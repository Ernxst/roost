package xsync

import (
	"context"
	"errors"
	"time"

	"roost/internal/store"
	"roost/internal/xparse"
)

// DetailPage is the X page roost opens posts in, one at a time, collecting
// X's TweetDetail responses for each.
type DetailPage interface {
	URL() (string, error)
	Load(url string) error
	// DrainDetail returns the bodies of TweetDetail responses received
	// since the last call.
	DrainDetail() ([]string, error)
	Scroll() error
}

// ErrSignedOut means X wants the user to sign in. Posts are opened in the
// background, so roost stops instead of asking.
var ErrSignedOut = errors.New("signed out of X")

// ErrNothing means X sent no response for a post. A run of them suggests X
// is limiting requests.
var ErrNothing = errors.New("X sent nothing")

// Detail is what X shows of a post opened on its own.
type Detail struct {
	// Post is the post itself, which carries the whole text of its article.
	// It is empty if X left the post out.
	Post xparse.Tweet
	// Thread is the post's thread in reading order, or nil when the post
	// stands alone.
	Thread []xparse.Tweet
}

// Open opens a post and reads it with its thread, scrolling until X stops
// sending more of the thread.
func Open(ctx context.Context, page DetailPage, c store.DetailCandidate, tick time.Duration) (Detail, error) {
	if tick == 0 {
		tick = time.Second
	}
	page.DrainDetail()
	if err := page.Load(c.URL); err != nil {
		return Detail{}, err
	}
	var bodies []string
	var chain []xparse.Tweet
	idle := 0
	for i := 0; i < 20 && idle < 3; i++ {
		select {
		case <-ctx.Done():
			return Detail{}, ctx.Err()
		case <-time.After(tick):
		}
		if u, err := page.URL(); err == nil && isLogin(u) {
			return Detail{}, ErrSignedOut
		}
		got, err := page.DrainDetail()
		if err != nil {
			return Detail{}, err
		}
		bodies = append(bodies, got...)
		if len(bodies) == 0 {
			continue
		}
		next, err := xparse.Thread(bodies, c.ID)
		if err != nil {
			return Detail{}, err
		}
		if len(next) > len(chain) {
			chain, idle = next, 0
		} else {
			idle++
		}
		page.Scroll()
	}
	if len(bodies) == 0 {
		return Detail{}, ErrNothing
	}
	post, _, err := xparse.Post(bodies, c.ID)
	return Detail{Post: post, Thread: chain}, err
}
